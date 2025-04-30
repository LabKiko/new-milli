package govern

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"new-milli/middleware"
	"new-milli/transport"
)

// Server is a govern server for management.
type Server struct {
	opts   *transport.Options
	server *server.Hertz
}

// NewServer creates a new govern server.
func NewServer(opts ...transport.ServerOption) *Server {
	options := &transport.Options{}
	for _, o := range opts {
		o.Apply(options)
	}

	srv := &Server{
		opts: options,
	}

	// Create Hertz server for management
	hertzServer := server.Default(
		server.WithHostPorts(options.Address),
	)

	// Apply middleware
	for _, m := range options.Middleware {
		hertzServer.Use(convertMiddleware(m))
	}

	// Register pprof endpoints
	hertzServer.GET("/debug/pprof/*any", func(ctx context.Context, c *app.RequestContext) {
		// Cannot directly use DefaultServeMux with Hertz
		c.String(http.StatusOK, "Pprof endpoint")
	})

	// Register metrics endpoint
	hertzServer.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		// TODO: Implement metrics endpoint
		c.String(http.StatusOK, "Metrics endpoint")
	})

	// Register health check endpoint
	hertzServer.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.String(http.StatusOK, "OK")
	})

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
