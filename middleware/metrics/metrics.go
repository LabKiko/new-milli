package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"new-milli/middleware"
	"new-milli/transport"
)

var (
	// DefaultBuckets is the default histogram buckets.
	DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
)

// Option is metrics option.
type Option func(*options)

// options is metrics options.
type options struct {
	disabled        bool
	namespace       string
	subsystem       string
	buckets         []float64
	constLabels     prometheus.Labels
	registry        prometheus.Registerer
	labelNames      []string
	labelValuesFunc func(ctx context.Context) []string
}

// WithDisabled returns an Option that disables metrics.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithNamespace returns an Option that sets the namespace.
func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.namespace = namespace
	}
}

// WithSubsystem returns an Option that sets the subsystem.
func WithSubsystem(subsystem string) Option {
	return func(o *options) {
		o.subsystem = subsystem
	}
}

// WithBuckets returns an Option that sets the buckets.
func WithBuckets(buckets []float64) Option {
	return func(o *options) {
		o.buckets = buckets
	}
}

// WithConstLabels returns an Option that sets the constant labels.
func WithConstLabels(labels prometheus.Labels) Option {
	return func(o *options) {
		o.constLabels = labels
	}
}

// WithRegistry returns an Option that sets the registry.
func WithRegistry(registry prometheus.Registerer) Option {
	return func(o *options) {
		o.registry = registry
	}
}

// WithLabelNames returns an Option that sets the label names.
func WithLabelNames(names ...string) Option {
	return func(o *options) {
		o.labelNames = names
	}
}

// WithLabelValuesFunc returns an Option that sets the function that returns the label values.
func WithLabelValuesFunc(fn func(ctx context.Context) []string) Option {
	return func(o *options) {
		o.labelValuesFunc = fn
	}
}

// Server returns a middleware that enables metrics for server.
func Server(opts ...Option) middleware.Middleware {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "server",
		buckets:     DefaultBuckets,
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{"kind", "operation", "status"},
		labelValuesFunc: func(ctx context.Context) []string {
			var (
				kind      = "unknown"
				operation = "unknown"
				status    = "unknown"
			)

			if tr, ok := transport.FromServerContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			return []string{kind, operation, status}
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

	// Create metrics
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "requests_total",
			Help:        "Total number of requests processed.",
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "request_duration_seconds",
			Help:        "Request duration in seconds.",
			Buckets:     cfg.buckets,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	requestInFlight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "requests_in_flight",
			Help:        "Number of requests in flight.",
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames[:len(cfg.labelNames)-1], // Remove status label
	)

	// Register metrics
	cfg.registry.MustRegister(requestCounter, requestDuration, requestInFlight)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				start  = time.Now()
				labels = cfg.labelValuesFunc(ctx)
			)

			// Increment in-flight counter
			inFlightLabels := labels[:len(labels)-1] // Remove status label
			requestInFlight.WithLabelValues(inFlightLabels...).Inc()
			defer requestInFlight.WithLabelValues(inFlightLabels...).Dec()

			// Handle the request
			reply, err = handler(ctx, req)

			// Set the status
			if err != nil {
				labels[len(labels)-1] = "error"
			} else {
				labels[len(labels)-1] = "success"
			}

			// Increment request counter
			requestCounter.WithLabelValues(labels...).Inc()

			// Observe request duration
			requestDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())

			return reply, err
		}
	}
}

// Client returns a middleware that enables metrics for client.
func Client(opts ...Option) middleware.Middleware {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "client",
		buckets:     DefaultBuckets,
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{"kind", "operation", "status"},
		labelValuesFunc: func(ctx context.Context) []string {
			var (
				kind      = "unknown"
				operation = "unknown"
				status    = "unknown"
			)

			if tr, ok := transport.FromClientContext(ctx); ok {
				kind = tr.Kind().String()
				operation = tr.Operation()
			}

			return []string{kind, operation, status}
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

	// Create metrics
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "requests_total",
			Help:        "Total number of requests processed.",
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "request_duration_seconds",
			Help:        "Request duration in seconds.",
			Buckets:     cfg.buckets,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	requestInFlight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        "requests_in_flight",
			Help:        "Number of requests in flight.",
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames[:len(cfg.labelNames)-1], // Remove status label
	)

	// Register metrics
	cfg.registry.MustRegister(requestCounter, requestDuration, requestInFlight)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				start  = time.Now()
				labels = cfg.labelValuesFunc(ctx)
			)

			// Increment in-flight counter
			inFlightLabels := labels[:len(labels)-1] // Remove status label
			requestInFlight.WithLabelValues(inFlightLabels...).Inc()
			defer requestInFlight.WithLabelValues(inFlightLabels...).Dec()

			// Handle the request
			reply, err = handler(ctx, req)

			// Set the status
			if err != nil {
				labels[len(labels)-1] = "error"
			} else {
				labels[len(labels)-1] = "success"
			}

			// Increment request counter
			requestCounter.WithLabelValues(labels...).Inc()

			// Observe request duration
			requestDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())

			return reply, err
		}
	}
}

// NewCounter creates a new counter.
func NewCounter(name, help string, opts ...Option) *prometheus.CounterVec {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "",
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        name,
			Help:        help,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	cfg.registry.MustRegister(counter)
	return counter
}

// NewGauge creates a new gauge.
func NewGauge(name, help string, opts ...Option) *prometheus.GaugeVec {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "",
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        name,
			Help:        help,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	cfg.registry.MustRegister(gauge)
	return gauge
}

// NewHistogram creates a new histogram.
func NewHistogram(name, help string, opts ...Option) *prometheus.HistogramVec {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "",
		buckets:     DefaultBuckets,
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        name,
			Help:        help,
			Buckets:     cfg.buckets,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	cfg.registry.MustRegister(histogram)
	return histogram
}

// NewSummary creates a new summary.
func NewSummary(name, help string, opts ...Option) *prometheus.SummaryVec {
	cfg := options{
		namespace:   "new_milli",
		subsystem:   "",
		constLabels: prometheus.Labels{},
		registry:    prometheus.DefaultRegisterer,
		labelNames:  []string{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:   cfg.namespace,
			Subsystem:   cfg.subsystem,
			Name:        name,
			Help:        help,
			ConstLabels: cfg.constLabels,
		},
		cfg.labelNames,
	)

	cfg.registry.MustRegister(summary)
	return summary
}
