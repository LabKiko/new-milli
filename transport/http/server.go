package http

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"new-milli/middleware"
	"new-milli/transport"
)

var (
	_ transport.Server = (*Server)(nil)
)

// Server is an HTTP server wrapper based on Hertz.
type Server struct {
	opts   *transport.Options
	server *server.Hertz
}

// NewServer creates a new HTTP server.
func NewServer(opts ...transport.ServerOption) *Server {
	options := &transport.Options{}
	for _, o := range opts {
		o.Apply(options)
	}

	srv := &Server{
		opts: options,
	}

	// Create Hertz server
	hertzServer := server.Default(
		server.WithHostPorts(options.Address),
	)

	// Apply middleware
	for _, m := range options.Middleware {
		hertzServer.Use(convertMiddleware(m))
	}

	srv.server = hertzServer
	return srv
}

// Init initializes the server.
func (s *Server) Init(opts ...transport.ServerOption) error {
	for _, o := range opts {
		o.Apply(s.opts)
	}
	return nil
}

// Start starts the server.
func (s *Server) Start(ctx context.Context) error {
	return s.server.Run()
}

// Stop stops the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// GetHertzServer returns the underlying Hertz server.
func (s *Server) GetHertzServer() *server.Hertz {
	return s.server
}

// convertMiddleware converts Milli middleware to Hertz middleware.
func convertMiddleware(m middleware.Middleware) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		// Create transport context
		tr := &Transport{
			operation:   string(ctx.Request.URI().Path()),
			reqHeader:   &HeaderCarrier{},
			replyHeader: &HeaderCarrier{},
		}

		// Copy headers from request to our carrier
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			tr.reqHeader.Set(string(key), string(value))
		})

		// Create new context with transport
		newCtx := transport.NewServerContext(c, tr)

		// Create handler
		handler := func(c context.Context, req interface{}) (interface{}, error) {
			// Continue with next handler
			ctx.Next(c)
			return nil, nil
		}

		// Apply middleware
		h := m(handler)

		// Execute handler
		_, err := h(newCtx, nil)
		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}
