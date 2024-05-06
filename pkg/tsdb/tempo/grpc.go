package tempo

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var logger = backend.NewLoggerWith("logger", "tsdb.tempo")

// newGrpcClient creates a new gRPC client to connect to a streaming query service.
// This uses the default google.golang.org/grpc library. One caveat to that is that it does not allow passing the
// default httpClient to the gRPC client. This means that we cannot use the same middleware that we use for
// standard HTTP requests.
// Using other library like connect-go isn't possible right now because Tempo uses non-standard proto compiler which
// makes generating different client difficult. See https://github.com/grafana/grafana/pull/81683
func newGrpcClient(ctx context.Context, settings backend.DataSourceInstanceSettings, opts httpclient.Options) (tempopb.StreamingQuerierClient, error) {
	parsedUrl, err := url.Parse(settings.URL)
	if err != nil {
		logger.Error("Error parsing URL for gRPC client", "error", err, "URL", settings.URL, "function", logEntrypoint())
		return nil, err
	}

	// Make sure we have some default port if none is set. This is required for gRPC to work.
	onlyHost := parsedUrl.Host
	if !strings.Contains(onlyHost, ":") {
		if parsedUrl.Scheme == "http" {
			onlyHost += ":80"
		} else {
			onlyHost += ":443"
		}
	}

	dialOpts, err := getDialOpts(ctx, settings, opts)
	if err != nil {
		logger.Error("Error getting dial options", "error", err, "URL", settings.URL, "function", logEntrypoint())
		return nil, err
	}
	clientConn, err := grpc.Dial(onlyHost, dialOpts...)
	if err != nil {
		logger.Error("Error dialing gRPC client", "error", err, "URL", settings.URL, "function", logEntrypoint())
		return nil, err
	}

	logger.Debug("Instantiating new gRPC client")
	return tempopb.NewStreamingQuerierClient(clientConn), nil
}

// getDialOpts creates options and interceptors (middleware) this should roughly match what we do in
// http_client_provider.go for standard http requests.
func getDialOpts(ctx context.Context, settings backend.DataSourceInstanceSettings, opts httpclient.Options) ([]grpc.DialOption, error) {
	// TODO: Missing middleware TracingMiddleware, DataSourceMetricsMiddleware, ContextualMiddleware,
	//  ResponseLimitMiddleware RedirectLimitMiddleware.
	// Also User agent but that is set before each rpc call as for decoupled DS we have to get it from request context
	// and cannot add it to client here.

	var dialOps []grpc.DialOption

	dialOps = append(dialOps, grpc.WithChainStreamInterceptor(CustomHeadersStreamInterceptor(opts)))
	if settings.BasicAuthEnabled {
		// If basic authentication is enabled, it uses TLS transport credentials and sets the basic authentication header for each RPC call.
		dialOps = append(dialOps, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		dialOps = append(dialOps, grpc.WithPerRPCCredentials(&basicAuth{
			Header: basicHeaderForAuth(opts.BasicAuth.User, opts.BasicAuth.Password),
		}))
	} else {
		// Otherwise, it uses insecure credentials.
		dialOps = append(dialOps, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// The following code is required to make gRPC work with Grafana Cloud PDC
	// (https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/)
	proxyClient, err := settings.ProxyClient(ctx)
	if err != nil {
		logger.Info("Proxy client cannot be retrieved. Not possible to check if secure socks proxy is enabled", "error", err)
		return dialOps, errors.New("proxy client cannot be retrieved")
	}
	if proxyClient.SecureSocksProxyEnabled() { // secure socks proxy is behind a feature flag
		dialer, err := proxyClient.NewSecureSocksProxyContextDialer()
		if err != nil {
			logger.Info("Failure in creating dialer", "error", err)
			return dialOps, errors.New("dialer not created")
		}
		logger.Debug("gRPC dialer instantiated. Appending gRPC dialer to dial options")
		dialOps = append(dialOps, grpc.WithContextDialer(func(ctx context.Context, host string) (net.Conn, error) {
			logger.Debug("Dialing secure socks proxy", "host", host)
			conn, err := dialer.Dial("tcp", host)
			if err != nil {
				logger.Info("Not possible to dial secure socks proxy", "error", err)
				return conn, errors.New("dialing failed")
			}
			select {
			case <-ctx.Done():
				err := errors.New("context canceled")
				logger.Info("Context has been canceled", "error", err)
				return conn, errors.New("context canceled")
			default:
				logger.Debug("Context is still valid")
			}
			return conn, nil
		}))
	}

	logger.Debug("Returning dial options")
	return dialOps, nil
}

// CustomHeadersStreamInterceptor adds custom headers to the outgoing context for each RPC call. Should work similar
// to the CustomHeadersMiddleware in the HTTP client provider.
func CustomHeadersStreamInterceptor(httpOpts httpclient.Options) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if len(httpOpts.Header) != 0 {
			for key, value := range httpOpts.Header {
				for _, v := range value {
					ctx = metadata.AppendToOutgoingContext(ctx, key, v)
				}
			}
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

type basicAuth struct {
	Header string
}

func (c *basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": c.Header,
	}, nil
}

func (c *basicAuth) RequireTransportSecurity() bool {
	return true
}

func basicHeaderForAuth(username, password string) string {
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))))
}
