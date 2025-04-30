package registry

import (
	"context"
	"errors"
	"time"
)

// Registry is service registry.
type Registry interface {
	// Register the registration.
	Register(ctx context.Context, service *ServiceInfo) error
	// Deregister the registration.
	Deregister(ctx context.Context, service *ServiceInfo) error
	// GetService return the service instances in memory according to the service name.
	GetService(ctx context.Context, serviceName string) ([]*ServiceInfo, error)
	// Watch creates a watcher according to the service name.
	Watch(ctx context.Context, serviceName string) (Watcher, error)
}

// ServiceInfo is service info.
type ServiceInfo struct {
	ID        string            // service id
	Name      string            // service name
	Version   string            // service version
	Metadata  map[string]string // service metadata
	Endpoints []string          // service endpoints
	Nodes     []*Node           // service nodes
}

// Node is service node.
type Node struct {
	ID       string            // node id
	Address  string            // node address
	Metadata map[string]string // node metadata
}

// Watcher is service watcher.
type Watcher interface {
	// Next returns services in the following two cases:
	// 1.the first time to watch and the service instance list is not empty.
	// 2.any service instance changes found.
	// if the above two conditions are not met, it will block until context deadline exceeded or canceled
	Next() ([]*ServiceInfo, error)
	// Stop the watcher.
	Stop() error
}

var (
	ErrNotFound = errors.New("service not found")
	ErrWatchCanceled = errors.New("watch canceled")
)

// Option is registry option.
type Option func(*Options)

// Options is registry options.
type Options struct {
	Timeout  time.Duration
	Context  context.Context
	Addrs    []string
	Secure   bool
	Username string
	Password string
}

// Timeout with registry timeout.
func Timeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// Addrs with registry addresses.
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Secure with registry secure option.
func Secure(secure bool) Option {
	return func(o *Options) {
		o.Secure = secure
	}
}

// Auth with registry authentication.
func Auth(username, password string) Option {
	return func(o *Options) {
		o.Username = username
		o.Password = password
	}
}
