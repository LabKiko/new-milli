package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"new-milli/middleware"
	"new-milli/transport"
)

const (
	tracerName = "new-milli/middleware/tracing"
)

// Option is tracing option.
type Option interface {
	apply(*options)
}

// options is tracing options.
type options struct {
	tracerProvider trace.TracerProvider
	propagators    propagation.TextMapPropagator
	disabled       bool
}

// optionFunc is a function that configures options.
type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

// WithDisabled returns an Option that disables tracing.
func WithDisabled(disabled bool) Option {
	return optionFunc(func(o *options) {
		o.disabled = disabled
	})
}

// WithTracerProvider returns an Option that sets the TracerProvider.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(o *options) {
		o.tracerProvider = provider
	})
}

// WithPropagators returns an Option that sets the TextMapPropagator.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(o *options) {
		o.propagators = propagators
	})
}

// Server returns a middleware that enables tracing for server.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}

	tracer := cfg.tracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion("1.0.0"),
	)

	if cfg.propagators == nil {
		cfg.propagators = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				// Extract the context from the headers
				carrier := headerCarrier{tr.RequestHeader()}
				ctx = cfg.propagators.Extract(ctx, carrier)

				// Start a new span
				ctx, span := tracer.Start(
					ctx,
					tr.Operation(),
					trace.WithSpanKind(trace.SpanKindServer),
					trace.WithAttributes(
						attribute.String("transport.kind", tr.Kind().String()),
					),
				)
				defer span.End()

				// Handle the request
				reply, err = handler(ctx, req)

				// Set the status
				if err != nil {
					span.RecordError(err)
				}

				return reply, err
			}
			return handler(ctx, req)
		}
	}
}

// Client returns a middleware that enables tracing for client.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.disabled {
		return func(handler middleware.Handler) middleware.Handler {
			return handler
		}
	}

	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}

	tracer := cfg.tracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion("1.0.0"),
	)

	if cfg.propagators == nil {
		cfg.propagators = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				// Start a new span
				ctx, span := tracer.Start(
					ctx,
					tr.Operation(),
					trace.WithSpanKind(trace.SpanKindClient),
					trace.WithAttributes(
						attribute.String("transport.kind", tr.Kind().String()),
					),
				)
				defer span.End()

				// Inject the context into the headers
				carrier := headerCarrier{tr.RequestHeader()}
				cfg.propagators.Inject(ctx, carrier)

				// Handle the request
				reply, err = handler(ctx, req)

				// Set the status
				if err != nil {
					span.RecordError(err)
				}

				return reply, err
			}
			return handler(ctx, req)
		}
	}
}

// headerCarrier is a carrier for HTTP headers.
type headerCarrier struct {
	header transport.Header
}

// Get returns the value associated with the passed key.
func (hc headerCarrier) Get(key string) string {
	return hc.header.Get(key)
}

// Set stores the key-value pair.
func (hc headerCarrier) Set(key string, value string) {
	hc.header.Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc headerCarrier) Keys() []string {
	return hc.header.Keys()
}
