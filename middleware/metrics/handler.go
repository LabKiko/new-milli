package metrics

import (
	"bytes"
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
)

// Handler returns a Hertz handler that exposes Prometheus metrics.
func Handler() func(ctx context.Context, c *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		data, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error gathering metrics: %v", err)
			return
		}

		c.Header("Content-Type", "text/plain; version=0.0.4")

		// Convert metrics to text format
		buffer := &bytes.Buffer{}
		for _, mf := range data {
			expfmt.MetricFamilyToText(buffer, mf)
		}

		// Write the response
		c.Data(http.StatusOK, "text/plain; version=0.0.4", buffer.Bytes())
	}
}

// HandlerFor returns a Hertz handler that exposes Prometheus metrics for the given gatherer.
func HandlerFor(gatherer prometheus.Gatherer) func(ctx context.Context, c *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		data, err := gatherer.Gather()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error gathering metrics: %v", err)
			return
		}

		c.Header("Content-Type", "text/plain; version=0.0.4")

		// Convert metrics to text format
		buffer := &bytes.Buffer{}
		for _, mf := range data {
			expfmt.MetricFamilyToText(buffer, mf)
		}

		// Write the response
		c.Data(http.StatusOK, "text/plain; version=0.0.4", buffer.Bytes())
	}
}

// HTTPHandler returns an HTTP handler that exposes Prometheus metrics.
func HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// HTTPHandlerFor returns an HTTP handler that exposes Prometheus metrics for the given gatherer.
func HTTPHandlerFor(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}
