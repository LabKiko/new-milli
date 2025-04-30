package connector

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotConnected is returned when the connector is not connected.
	ErrNotConnected = errors.New("connector not connected")
	// ErrAlreadyConnected is returned when the connector is already connected.
	ErrAlreadyConnected = errors.New("connector already connected")
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrNotSupported is returned when a feature is not supported.
	ErrNotSupported = errors.New("feature not supported")
)

// Connector is the interface for database connectors.
type Connector interface {
	// Connect connects to the database.
	Connect(ctx context.Context) error
	// Disconnect disconnects from the database.
	Disconnect(ctx context.Context) error
	// Ping checks if the database is reachable.
	Ping(ctx context.Context) error
	// IsConnected returns true if the connector is connected.
	IsConnected() bool
	// Name returns the name of the connector.
	Name() string
	// Client returns the underlying client.
	Client() interface{}
}

// Option is a function that configures a connector.
type Option func(interface{})

// Config is the base configuration for connectors.
type Config struct {
	// Name is the name of the connector.
	Name string
	// Address is the address of the database.
	Address string
	// Username is the username for authentication.
	Username string
	// Password is the password for authentication.
	Password string
	// Database is the name of the database.
	Database string
	// ConnectTimeout is the timeout for connecting to the database.
	ConnectTimeout time.Duration
	// ReadTimeout is the timeout for read operations.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for write operations.
	WriteTimeout time.Duration
	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int
	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int
	// MaxConnLifetime is the maximum lifetime of a connection.
	MaxConnLifetime time.Duration
	// MaxIdleTime is the maximum idle time of a connection.
	MaxIdleTime time.Duration
	// EnableTLS enables TLS for the connection.
	EnableTLS bool
	// TLSCertPath is the path to the TLS certificate.
	TLSCertPath string
	// TLSKeyPath is the path to the TLS key.
	TLSKeyPath string
	// TLSCAPath is the path to the TLS CA certificate.
	TLSCAPath string
	// TLSSkipVerify skips TLS verification.
	TLSSkipVerify bool
}

// Registry is a registry of connectors.
type Registry struct {
	connectors map[string]Connector
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]Connector),
	}
}

// Register registers a connector.
func (r *Registry) Register(name string, connector Connector) {
	r.connectors[name] = connector
}

// Get returns a connector by name.
func (r *Registry) Get(name string) (Connector, bool) {
	connector, ok := r.connectors[name]
	return connector, ok
}

// List returns all registered connectors.
func (r *Registry) List() map[string]Connector {
	return r.connectors
}

// Close closes all registered connectors.
func (r *Registry) Close(ctx context.Context) error {
	var lastErr error
	for _, connector := range r.connectors {
		if connector.IsConnected() {
			if err := connector.Disconnect(ctx); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// global is the global registry.
var global = NewRegistry()

// Register registers a connector in the global registry.
func Register(name string, connector Connector) {
	global.Register(name, connector)
}

// Get returns a connector by name from the global registry.
func Get(name string) (Connector, bool) {
	return global.Get(name)
}

// List returns all registered connectors from the global registry.
func List() map[string]Connector {
	return global.List()
}

// Close closes all registered connectors in the global registry.
func Close(ctx context.Context) error {
	return global.Close(ctx)
}
