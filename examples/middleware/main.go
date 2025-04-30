package main

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sony/gobreaker"
	"new-milli"
	"new-milli/middleware/circuitbreaker"
	"new-milli/middleware/logging"
	"new-milli/middleware/metrics"
	"new-milli/middleware/ratelimit"
	"new-milli/middleware/recovery"
	"new-milli/middleware/tracing"
	"new-milli/transport"
	"new-milli/transport/http"
)

func main() {
	// Create HTTP server
	httpServer := http.NewServer(
		transport.Address(":8000"),
		transport.Middleware(
			// Recovery middleware should be the first to catch panics
			recovery.Server(),
			// Tracing middleware for distributed tracing
			tracing.Server(),
			// Metrics middleware for monitoring
			metrics.Server(
				metrics.WithNamespace("example"),
				metrics.WithSubsystem("http"),
			),
			// Rate limiting middleware to prevent overload
			ratelimit.Server(
				ratelimit.WithRate(100),
				ratelimit.WithCapacity(100),
			),
			// Circuit breaker middleware for fault tolerance
			circuitbreaker.Server(
				circuitbreaker.WithTimeout(time.Second * 10),
				circuitbreaker.WithMaxRequests(100),
				circuitbreaker.WithReadyToTrip(func(counts gobreaker.Counts) bool {
					failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
					return counts.Requests >= 10 && failureRatio >= 0.5
				}),
			),
			// Logging middleware should be the last to log the final result
			logging.Server(
				logging.WithSlowThreshold(time.Millisecond * 500),
			),
		),
	)

	// Get Hertz server instance
	hertzServer := httpServer.GetHertzServer()

	// Register routes
	hertzServer.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "Hello, World!")
	})

	// Register metrics endpoint
	hertzServer.GET("/metrics", metrics.Handler())

	// Create application
	app, err := newMilli.New(
		newMilli.Name("middleware-example"),
		newMilli.Version("v1.0.0"),
		newMilli.Server(httpServer),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Run application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
