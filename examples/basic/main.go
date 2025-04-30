package main

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"new-milli"
	"new-milli/middleware/logging"
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
			recovery.Server(),
			tracing.Server(),
			logging.Server(),
		),
	)

	// Register routes

	hertzServer := httpServer.GetHertzServer()
	hertzServer.GET("/hello", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "Hello, World!")
	})

	// Create application
	app, err := newMilli.New(
		newMilli.Name("example"),
		newMilli.Version("v1.0.0"),
		newMilli.Server(httpServer),
		newMilli.StopTimeout(time.Second*5),
		newMilli.BeforeStart(func(ctx context.Context) error {
			log.Println("Before start")
			return nil
		}),
		newMilli.AfterStart(func(ctx context.Context) error {
			log.Println("After start")
			return nil
		}),
		newMilli.BeforeStop(func(ctx context.Context) error {
			log.Println("Before stop")
			return nil
		}),
		newMilli.AfterStop(func(ctx context.Context) error {
			log.Println("After stop")
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
