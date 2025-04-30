package logging

import (
	"context"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"new-milli/middleware"
	"new-milli/transport"
)

// Option is logging option.
type Option func(*options)

// options is logging options.
type options struct {
	disabled      bool
	level         klog.Level
	slowThreshold time.Duration
}

// WithDisabled returns an Option that disables logging.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithLevel returns an Option that sets the log level.
func WithLevel(level klog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithSlowThreshold returns an Option that sets the slow threshold.
func WithSlowThreshold(threshold time.Duration) Option {
	return func(o *options) {
		o.slowThreshold = threshold
	}
}

// Server returns a middleware that enables logging for server.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{
		level:         klog.LevelInfo,
		slowThreshold: time.Millisecond * 500,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
				start     = time.Now()
			)

			if tr, ok := transport.FromServerContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			// Handle the request
			reply, err = handler(ctx, req)

			// Calculate the duration
			duration := time.Since(start)

			// Set the code and reason
			if err != nil {
				code = 500
				reason = err.Error()
			} else {
				code = 200
				reason = "OK"
			}

			// Log the request
			if duration > cfg.slowThreshold {
				klog.CtxWarnf(ctx, "[%s] %s %s %d %s %s", kind, "server", operation, code, reason, duration)
			} else {
				klog.CtxInfof(ctx, "[%s] %s %s %d %s %s", kind, "server", operation, code, reason, duration)
			}

			return reply, err
		}
	}
}

// Client returns a middleware that enables logging for client.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{
		level:         klog.LevelInfo,
		slowThreshold: time.Millisecond * 500,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
				start     = time.Now()
			)

			if tr, ok := transport.FromClientContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			// Handle the request
			reply, err = handler(ctx, req)

			// Calculate the duration
			duration := time.Since(start)

			// Set the code and reason
			if err != nil {
				code = 500
				reason = err.Error()
			} else {
				code = 200
				reason = "OK"
			}

			// Log the request
			if duration > cfg.slowThreshold {
				klog.CtxWarnf(ctx, "[%s] %s %s %d %s %s", kind, "client", operation, code, reason, duration)
			} else {
				klog.CtxInfof(ctx, "[%s] %s %s %d %s %s", kind, "client", operation, code, reason, duration)
			}

			return reply, err
		}
	}
}
