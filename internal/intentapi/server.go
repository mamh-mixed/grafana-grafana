package intentapi

import (
	"context"
	"net"
	"net/http"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/setting"
)

// HTTPServerConfig
type HTTPServerConfig struct {
	ListenAddr string
}

// NewHTTPServerConfig
func NewHTTPServerConfig(cfg *setting.Cfg) (HTTPServerConfig, error) {
	// TODO: parse config
	return HTTPServerConfig{}, nil
}

// HTTPServer
type HTTPServer struct {
	config  HTTPServerConfig
	handler http.Handler
	logger  log.Logger
}

// ProvideHTTPServer
func ProvideHTTPServer(cfg *setting.Cfg, handler http.Handler) (*HTTPServer, error) {
	config, err := NewHTTPServerConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &HTTPServer{
		config: config,
		logger: log.New("intentapi"),
	}, nil
}

// Run
func (s *HTTPServer) Run(ctx context.Context) error {
	// TODO: proper config
	server := &http.Server{
		Handler: s.handler,
	}

	// TODO: unix socket?
	lis, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return err
	}

	// TODO: context & canceling
	// TODO: TLS & HTTP2
	return server.Serve(lis)
}
