package intentapi

import (
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/setting"
)

// ApiserverProxyConfig
type ApiserverProxyConfig struct {
	KubeApiserverHost   string
	KubeApiserverScheme string
}

// NewApiserverProxyConfig
func NewApiserverProxyConfig(cfg *setting.Cfg) (ApiserverProxyConfig, error) {
	// TODO: parse config
	return ApiserverProxyConfig{}, nil
}

// ApiserverProxy
type ApiserverProxy struct {
	config ApiserverProxyConfig
	proxy  *httputil.ReverseProxy
	logger log.Logger
}

// ProvideApiserverProxy
func ProvideApiserverProxy(cfg *setting.Cfg) (*ApiserverProxy, error) {
	config, err := NewApiserverProxyConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &ApiserverProxy{
		config: config,
		logger: log.New("intentapi.proxy"),
	}, nil
}

// ServeHTTP
func (p *ApiserverProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	proxy := p.getProxy()
	proxy.ServeHTTP(w, req)
}

func (p *ApiserverProxy) getProxy() *httputil.ReverseProxy {
	if p.proxy != nil {
		p.proxy = &httputil.ReverseProxy{
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				// TODO: proper error handling.
				p.logger.Error("error handling request", err, r)
				w.WriteHeader(http.StatusBadGateway)
			},
			Director: func(req *http.Request) {
				// TODO: tracing & logging
				// TODO: double-check that this is enough.
				req.URL.Scheme = p.config.KubeApiserverScheme
				req.URL.Host = p.config.KubeApiserverHost
			},
			// TODO: dial context tracing / conntrack metrics.
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          10000,
				MaxIdleConnsPerHost:   1000, // see https://github.com/golang/go/issues/13801
				IdleConnTimeout:       90 * time.Second,
				DisableKeepAlives:     false,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	}

	return p.proxy
}
