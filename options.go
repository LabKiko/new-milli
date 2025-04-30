package newMilli

import (
	"context"
	"os"
	"time"

	"new-milli/transport"
)

// Option is application option.
type Option func(o *options)

// options is application options.
type options struct {
	id               string
	name             string
	version          string
	metadata         map[string]string
	ctx              context.Context
	sigs             []os.Signal
	registrarTimeout time.Duration
	stopTimeout      time.Duration
	servers          []transport.Server
	beforeStart      []func(context.Context) error
	afterStart       []func(context.Context) error
	beforeStop       []func(context.Context) error
	afterStop        []func(context.Context) error
}

// ID with service id.
func ID(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

// Name with service name.
func Name(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// Version with service version.
func Version(version string) Option {
	return func(o *options) {
		o.version = version
	}
}

// Metadata with service metadata.
func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// Context with service context.
func Context(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}

// Signal with service signal.
func Signal(sigs ...os.Signal) Option {
	return func(o *options) {
		o.sigs = sigs
	}
}

// RegistrarTimeout with service registrar timeout.
func RegistrarTimeout(t time.Duration) Option {
	return func(o *options) {
		o.registrarTimeout = t
	}
}

// StopTimeout with service stop timeout.
func StopTimeout(t time.Duration) Option {
	return func(o *options) {
		o.stopTimeout = t
	}
}

// Server with transport servers.
func Server(srv ...transport.Server) Option {
	return func(o *options) {
		o.servers = append(o.servers, srv...)
	}
}

// BeforeStart with service before start hooks.
func BeforeStart(fn func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStart = append(o.beforeStart, fn)
	}
}

// AfterStart with service after start hooks.
func AfterStart(fn func(context.Context) error) Option {
	return func(o *options) {
		o.afterStart = append(o.afterStart, fn)
	}
}

// BeforeStop with service before stop hooks.
func BeforeStop(fn func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStop = append(o.beforeStop, fn)
	}
}

// AfterStop with service after stop hooks.
func AfterStop(fn func(context.Context) error) Option {
	return func(o *options) {
		o.afterStop = append(o.afterStop, fn)
	}
}
