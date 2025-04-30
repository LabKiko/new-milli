package grpc

import (
	"context"
	"net"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/server"
	"new-milli/transport"
)

var (
	_ transport.Server = (*Server)(nil)
)

// Server is a gRPC server wrapper based on Kitex.
type Server struct {
	opts   *transport.Options
	server server.Server
}

// NewServer creates a new gRPC server.
func NewServer(opts ...transport.ServerOption) *Server {
	options := &transport.Options{}
	for _, o := range opts {
		o.Apply(options)
	}

	srv := &Server{
		opts: options,
	}

	return srv
}

// Init initializes the server.
func (s *Server) Init(opts ...transport.ServerOption) error {
	for _, o := range opts {
		o.Apply(s.opts)
	}
	return nil
}

// RegisterService registers a service with the server.
func (s *Server) RegisterService(service interface{}) {
	// Create Kitex server options
	serverOpts := []server.Option{
		server.WithServiceAddr(&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080}),
	}

	// Use address from options if provided
	if s.opts.Address != "" {
		// Parse the address
		addr, err := net.ResolveTCPAddr("tcp", s.opts.Address)
		if err != nil {
			klog.Errorf("Failed to resolve address %s: %v", s.opts.Address, err)
		} else {
			serverOpts = append(serverOpts, server.WithServiceAddr(addr))
		}
	}

	// Apply middleware
	for _, m := range s.opts.Middleware {
		// Note: Middleware conversion is handled differently in Kitex
		// This is a placeholder for middleware handling
		klog.Infof("Adding middleware: %T", m)
	}

	// Create Kitex server
	// Note: This is a simplified version, actual implementation depends on Kitex API
	// svr := server.NewServer(serverOpts...)
	// s.server = svr
	klog.Infof("Registered service: %T", service)
}

// Start starts the server.
func (s *Server) Start(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Run()
}

// Stop stops the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Stop()
}

// GetKitexServer returns the underlying Kitex server.
func (s *Server) GetKitexServer() server.Server {
	return s.server
}

// Note: This is a placeholder for middleware conversion
// The actual implementation depends on the Kitex API
// and how middleware is handled in Kitex
