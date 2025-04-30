package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/juju/ratelimit"
	"new-milli/middleware"
	"new-milli/transport"
)

var (
	// ErrLimitExceed is returned when the rate limit is exceeded.
	ErrLimitExceed = errors.New("rate limit exceeded")
)

// Option is rate limit option.
type Option func(*options)

// options is rate limit options.
type options struct {
	disabled   bool
	capacity   int64
	rate       float64
	waitIfFull bool
}

// WithDisabled returns an Option that disables rate limiting.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithCapacity returns an Option that sets the bucket capacity.
func WithCapacity(capacity int64) Option {
	return func(o *options) {
		o.capacity = capacity
	}
}

// WithRate returns an Option that sets the fill rate.
func WithRate(rate float64) Option {
	return func(o *options) {
		o.rate = rate
	}
}

// WithWaitIfFull returns an Option that sets whether to wait if the bucket is full.
func WithWaitIfFull(wait bool) Option {
	return func(o *options) {
		o.waitIfFull = wait
	}
}

// Server returns a middleware that enables rate limiting for server.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{
		capacity:   100,
		rate:       100,
		waitIfFull: false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	// Create a token bucket
	bucket := ratelimit.NewBucketWithRate(cfg.rate, cfg.capacity)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				operation string
				kind      string
			)

			if tr, ok := transport.FromServerContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			// Take a token from the bucket
			var taken bool
			if cfg.waitIfFull {
				// Wait for a token to be available
				bucket.Wait(1)
				taken = true
			} else {
				// Try to take a token without waiting
				taken = bucket.TakeAvailable(1) > 0
			}

			if !taken {
				klog.CtxWarnf(ctx, "[%s] %s %s rate limit exceeded", kind, "server", operation)
				return nil, ErrLimitExceed
			}

			// Handle the request
			return handler(ctx, req)
		}
	}
}

// Client returns a middleware that enables rate limiting for client.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{
		capacity:   100,
		rate:       100,
		waitIfFull: false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	// Create a token bucket
	bucket := ratelimit.NewBucketWithRate(cfg.rate, cfg.capacity)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				operation string
				kind      string
			)

			if tr, ok := transport.FromClientContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			// Take a token from the bucket
			var taken bool
			if cfg.waitIfFull {
				// Wait for a token to be available
				bucket.Wait(1)
				taken = true
			} else {
				// Try to take a token without waiting
				taken = bucket.TakeAvailable(1) > 0
			}

			if !taken {
				klog.CtxWarnf(ctx, "[%s] %s %s rate limit exceeded", kind, "client", operation)
				return nil, ErrLimitExceed
			}

			// Handle the request
			return handler(ctx, req)
		}
	}
}

// NewLimiter creates a new rate limiter.
func NewLimiter(rate float64, capacity int64) *ratelimit.Bucket {
	return ratelimit.NewBucketWithRate(rate, capacity)
}

// NewLimiterWithClock creates a new rate limiter with a custom clock.
func NewLimiterWithClock(rate float64, capacity int64, clock ratelimit.Clock) *ratelimit.Bucket {
	return ratelimit.NewBucketWithRateAndClock(rate, capacity, clock)
}

// Wait waits for n tokens to be available.
func Wait(ctx context.Context, limiter *ratelimit.Bucket, n int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		limiter.Wait(n)
		return nil
	}
}

// Allow returns true if n tokens are available.
func Allow(limiter *ratelimit.Bucket, n int64) bool {
	return limiter.TakeAvailable(n) > 0
}

// AllowN returns true if n tokens are available at the specified time.
// This is a best-effort implementation since the underlying bucket doesn't
// support time-based token availability checks.
func AllowN(limiter *ratelimit.Bucket, now time.Time, n int64) bool {
	// Since we can't check token availability at a specific time,
	// we'll use TakeAvailable which checks current availability.
	return limiter.TakeAvailable(n) > 0
}
