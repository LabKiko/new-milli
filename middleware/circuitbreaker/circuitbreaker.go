package circuitbreaker

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/sony/gobreaker"
	"new-milli/middleware"
	"new-milli/transport"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Option is circuit breaker option.
type Option func(*options)

// options is circuit breaker options.
type options struct {
	disabled           bool
	name               string
	maxRequests        uint32
	interval           time.Duration
	timeout            time.Duration
	readyToTrip        func(counts gobreaker.Counts) bool
	onStateChange      func(name string, from gobreaker.State, to gobreaker.State)
	isSuccessful       func(err error) bool
	fallbackHandler    func(ctx context.Context, req interface{}) (interface{}, error)
	circuitBreakerName func(ctx context.Context) string
}

// WithDisabled returns an Option that disables circuit breaking.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithName returns an Option that sets the circuit breaker name.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithMaxRequests returns an Option that sets the maximum number of requests allowed to pass through when the circuit breaker is half-open.
func WithMaxRequests(maxRequests uint32) Option {
	return func(o *options) {
		o.maxRequests = maxRequests
	}
}

// WithInterval returns an Option that sets the cyclic period of the closed state.
func WithInterval(interval time.Duration) Option {
	return func(o *options) {
		o.interval = interval
	}
}

// WithTimeout returns an Option that sets the period of the open state.
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithReadyToTrip returns an Option that sets the function that decides whether to trip the circuit.
func WithReadyToTrip(fn func(counts gobreaker.Counts) bool) Option {
	return func(o *options) {
		o.readyToTrip = fn
	}
}

// WithOnStateChange returns an Option that sets the function that is called when the circuit breaker changes state.
func WithOnStateChange(fn func(name string, from gobreaker.State, to gobreaker.State)) Option {
	return func(o *options) {
		o.onStateChange = fn
	}
}

// WithIsSuccessful returns an Option that sets the function that decides whether a request is successful.
func WithIsSuccessful(fn func(err error) bool) Option {
	return func(o *options) {
		o.isSuccessful = fn
	}
}

// WithFallbackHandler returns an Option that sets the fallback handler.
func WithFallbackHandler(fn func(ctx context.Context, req interface{}) (interface{}, error)) Option {
	return func(o *options) {
		o.fallbackHandler = fn
	}
}

// WithCircuitBreakerName returns an Option that sets the function that returns the circuit breaker name.
func WithCircuitBreakerName(fn func(ctx context.Context) string) Option {
	return func(o *options) {
		o.circuitBreakerName = fn
	}
}

// Server returns a middleware that enables circuit breaking for server.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{
		name:        "server",
		maxRequests: 100,
		interval:    time.Minute,
		timeout:     time.Minute,
		readyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		},
		onStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			klog.Infof("Circuit breaker %s changed from %s to %s", name, from, to)
		},
		isSuccessful: func(err error) bool {
			return err == nil
		},
		fallbackHandler: func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, ErrCircuitOpen
		},
		circuitBreakerName: func(ctx context.Context) string {
			if tr, ok := transport.FromServerContext(ctx); ok {
				return "server_" + tr.Operation()
			}
			return "server"
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	// Create a circuit breaker registry
	registry := make(map[string]*gobreaker.CircuitBreaker)

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

			// Get the circuit breaker name
			name := cfg.circuitBreakerName(ctx)

			// Get or create the circuit breaker
			var cb *gobreaker.CircuitBreaker
			var ok bool
			if cb, ok = registry[name]; !ok {
				cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
					Name:          name,
					MaxRequests:   cfg.maxRequests,
					Interval:      cfg.interval,
					Timeout:       cfg.timeout,
					ReadyToTrip:   cfg.readyToTrip,
					OnStateChange: cfg.onStateChange,
					IsSuccessful:  cfg.isSuccessful,
				})
				registry[name] = cb
			}

			// Execute the request with the circuit breaker
			result, err := cb.Execute(func() (interface{}, error) {
				return handler(ctx, req)
			})

			// If the circuit is open, use the fallback handler
			if err == gobreaker.ErrOpenState {
				klog.CtxWarnf(ctx, "[%s] %s %s circuit breaker is open", kind, "server", operation)
				return cfg.fallbackHandler(ctx, req)
			}

			return result, err
		}
	}
}

// Client returns a middleware that enables circuit breaking for client.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{
		name:        "client",
		maxRequests: 100,
		interval:    time.Minute,
		timeout:     time.Minute,
		readyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		},
		onStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			klog.Infof("Circuit breaker %s changed from %s to %s", name, from, to)
		},
		isSuccessful: func(err error) bool {
			return err == nil
		},
		fallbackHandler: func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, ErrCircuitOpen
		},
		circuitBreakerName: func(ctx context.Context) string {
			if tr, ok := transport.FromClientContext(ctx); ok {
				return "client_" + tr.Operation()
			}
			return "client"
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	// Create a circuit breaker registry
	registry := make(map[string]*gobreaker.CircuitBreaker)

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

			// Get the circuit breaker name
			name := cfg.circuitBreakerName(ctx)

			// Get or create the circuit breaker
			var cb *gobreaker.CircuitBreaker
			var ok bool
			if cb, ok = registry[name]; !ok {
				cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
					Name:          name,
					MaxRequests:   cfg.maxRequests,
					Interval:      cfg.interval,
					Timeout:       cfg.timeout,
					ReadyToTrip:   cfg.readyToTrip,
					OnStateChange: cfg.onStateChange,
					IsSuccessful:  cfg.isSuccessful,
				})
				registry[name] = cb
			}

			// Execute the request with the circuit breaker
			result, err := cb.Execute(func() (interface{}, error) {
				return handler(ctx, req)
			})

			// If the circuit is open, use the fallback handler
			if err == gobreaker.ErrOpenState {
				klog.CtxWarnf(ctx, "[%s] %s %s circuit breaker is open", kind, "client", operation)
				return cfg.fallbackHandler(ctx, req)
			}

			return result, err
		}
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, opts ...Option) *gobreaker.CircuitBreaker {
	cfg := options{
		name:        name,
		maxRequests: 100,
		interval:    time.Minute,
		timeout:     time.Minute,
		readyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		},
		onStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			klog.Infof("Circuit breaker %s changed from %s to %s", name, from, to)
		},
		isSuccessful: func(err error) bool {
			return err == nil
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:          cfg.name,
		MaxRequests:   cfg.maxRequests,
		Interval:      cfg.interval,
		Timeout:       cfg.timeout,
		ReadyToTrip:   cfg.readyToTrip,
		OnStateChange: cfg.onStateChange,
		IsSuccessful:  cfg.isSuccessful,
	})
}
