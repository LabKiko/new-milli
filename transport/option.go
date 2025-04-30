package transport

import (
	"time"

	"new-milli/middleware"
)

// ServerOption is server option.
type ServerOption interface {
	Apply(o *Options)
}

// ServerOptions is server options.
type ServerOptions func(o *Options)

// Apply applies the ServerOptions to the given Options.
func (f ServerOptions) Apply(o *Options) {
	f(o)
}

// Options is server options.
type Options struct {
	ID               string        // server id
	Name             string        // server name
	Version          string        // server version
	Address          string        // server address
	Timeout          time.Duration // server timeout
	RegisterTTL      time.Duration // The register expiry time
	RegisterInterval time.Duration // The interval on which to register
	Middleware       []middleware.Middleware
}

// ID with server id.
func ID(id string) ServerOption {
	return ServerOptions(func(o *Options) {
		o.ID = id
	})
}

// Name with server name.
func Name(name string) ServerOption {
	return ServerOptions(func(o *Options) {
		o.Name = name
	})
}

// Version with server version.
func Version(version string) ServerOption {
	return ServerOptions(func(o *Options) {
		o.Version = version
	})
}

// Address with server address.
func Address(addr string) ServerOption {
	return ServerOptions(func(o *Options) {
		o.Address = addr
	})
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return ServerOptions(func(o *Options) {
		o.Timeout = timeout
	})
}

// Middleware with server middleware.
func Middleware(m ...middleware.Middleware) ServerOption {
	return ServerOptions(func(o *Options) {
		o.Middleware = append(o.Middleware, m...)
	})
}

// RegisterTTL with server register ttl.
func RegisterTTL(ttl time.Duration) ServerOption {
	return ServerOptions(func(o *Options) {
		o.RegisterTTL = ttl
	})
}

// RegisterInterval with server register interval.
func RegisterInterval(interval time.Duration) ServerOption {
	return ServerOptions(func(o *Options) {
		o.RegisterInterval = interval
	})
}
