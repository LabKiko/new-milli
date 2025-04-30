package recovery

import (
	"context"
	"fmt"
	"runtime"

	"github.com/cloudwego/kitex/pkg/klog"
	"new-milli/middleware"
)

// Option is recovery option.
type Option func(*options)

// options is recovery options.
type options struct {
	disabled        bool
	stackSize       int
	disableStack    bool
	disablePrint    bool
	recoveryHandler func(ctx context.Context, err interface{}) error
}

// WithDisabled returns an Option that disables recovery.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithStackSize returns an Option that sets the stack size.
func WithStackSize(size int) Option {
	return func(o *options) {
		o.stackSize = size
	}
}

// WithDisableStackAll returns an Option that disables stack trace.
func WithDisableStackAll(disable bool) Option {
	return func(o *options) {
		o.disableStack = disable
	}
}

// WithDisablePrintStack returns an Option that disables printing stack trace.
func WithDisablePrintStack(disable bool) Option {
	return func(o *options) {
		o.disablePrint = disable
	}
}

// WithRecoveryHandler returns an Option that sets the recovery handler.
func WithRecoveryHandler(handler func(ctx context.Context, err interface{}) error) Option {
	return func(o *options) {
		o.recoveryHandler = handler
	}
}

// Server returns a middleware that recovers from panics.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{
		stackSize: 4 << 10, // 4KB
		recoveryHandler: func(ctx context.Context, err interface{}) error {
			return fmt.Errorf("panic: %v", err)
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

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					// Log the stack
					stack := make([]byte, cfg.stackSize)
					stack = stack[:runtime.Stack(stack, !cfg.disableStack)]
					if !cfg.disablePrint {
						klog.CtxErrorf(ctx, "[Recovery] panic: %v\n%s", r, stack)
					}

					// Call the recovery handler
					err = cfg.recoveryHandler(ctx, r)
				}
			}()

			return handler(ctx, req)
		}
	}
}

// Client returns a middleware that recovers from panics.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{
		stackSize: 4 << 10, // 4KB
		recoveryHandler: func(ctx context.Context, err interface{}) error {
			return fmt.Errorf("panic: %v", err)
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

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					// Log the stack
					stack := make([]byte, cfg.stackSize)
					stack = stack[:runtime.Stack(stack, !cfg.disableStack)]
					if !cfg.disablePrint {
						klog.CtxErrorf(ctx, "[Recovery] panic: %v\n%s", r, stack)
					}

					// Call the recovery handler
					err = cfg.recoveryHandler(ctx, r)
				}
			}()

			return handler(ctx, req)
		}
	}
}
