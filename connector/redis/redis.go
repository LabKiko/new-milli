package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/redis/go-redis/v9"
	"new-milli/connector"
)

// Config is the configuration for the Redis connector.
type Config struct {
	connector.Config
	// Mode is the Redis mode (single, sentinel, cluster).
	Mode string
	// MasterName is the name of the Redis Sentinel master.
	MasterName string
	// DB is the Redis database number.
	DB int
	// PoolSize is the maximum number of connections in the pool.
	PoolSize int
	// MinIdleConns is the minimum number of idle connections in the pool.
	MinIdleConns int
	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for socket reads.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes.
	WriteTimeout time.Duration
	// PoolTimeout is the timeout for getting a connection from the pool.
	PoolTimeout time.Duration
	// IdleTimeout is the timeout for idle connections.
	IdleTimeout time.Duration
	// MaxRetries is the maximum number of retries before giving up.
	MaxRetries int
	// MinRetryBackoff is the minimum backoff between retries.
	MinRetryBackoff time.Duration
	// MaxRetryBackoff is the maximum backoff between retries.
	MaxRetryBackoff time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Config: connector.Config{
			Name:            "redis",
			Address:         "localhost:6379",
			Username:        "",
			Password:        "",
			Database:        "",
			ConnectTimeout:  time.Second * 10,
			ReadTimeout:     time.Second * 30,
			WriteTimeout:    time.Second * 30,
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			MaxConnLifetime: time.Hour,
			MaxIdleTime:     time.Minute * 30,
			EnableTLS:       false,
			TLSSkipVerify:   false,
		},
		Mode:            "single",
		MasterName:      "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    0,
		DialTimeout:     time.Second * 5,
		ReadTimeout:     time.Second * 3,
		WriteTimeout:    time.Second * 3,
		PoolTimeout:     time.Second * 4,
		IdleTimeout:     time.Minute * 5,
		MaxRetries:      3,
		MinRetryBackoff: time.Millisecond * 8,
		MaxRetryBackoff: time.Millisecond * 512,
	}
}

// Connector is a Redis connector.
type Connector struct {
	config     *Config
	client     redis.UniversalClient
	mu         sync.RWMutex
	connected  bool
	tlsConfig  *tls.Config
}

// New creates a new Redis connector.
func New(opts ...connector.Option) connector.Connector {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	return &Connector{
		config: config,
	}
}

// Connect connects to the database.
func (c *Connector) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return connector.ErrAlreadyConnected
	}

	// Setup TLS if enabled
	if c.config.EnableTLS {
		if err := c.setupTLS(); err != nil {
			return err
		}
	}

	// Parse addresses
	var addrs []string
	if strings.Contains(c.config.Address, ",") {
		addrs = strings.Split(c.config.Address, ",")
	} else {
		addrs = []string{c.config.Address}
	}

	// Create Redis client options
	opts := &redis.UniversalOptions{
		Addrs:           addrs,
		Username:        c.config.Username,
		Password:        c.config.Password,
		DB:              c.config.DB,
		MasterName:      c.config.MasterName,
		PoolSize:        c.config.PoolSize,
		MinIdleConns:    c.config.MinIdleConns,
		ConnMaxLifetime: c.config.MaxConnLifetime,
		ConnMaxIdleTime: c.config.MaxIdleTime,
		DialTimeout:     c.config.DialTimeout,
		ReadTimeout:     c.config.ReadTimeout,
		WriteTimeout:    c.config.WriteTimeout,
		PoolTimeout:     c.config.PoolTimeout,
		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,
	}

	// Set TLS config if enabled
	if c.config.EnableTLS {
		opts.TLSConfig = c.tlsConfig
	}

	// Create Redis client based on mode
	var client redis.UniversalClient
	switch strings.ToLower(c.config.Mode) {
	case "single":
		client = redis.NewClient(&redis.Options{
			Addr:            addrs[0],
			Username:        opts.Username,
			Password:        opts.Password,
			DB:              opts.DB,
			MaxRetries:      opts.MaxRetries,
			MinRetryBackoff: opts.MinRetryBackoff,
			MaxRetryBackoff: opts.MaxRetryBackoff,
			DialTimeout:     opts.DialTimeout,
			ReadTimeout:     opts.ReadTimeout,
			WriteTimeout:    opts.WriteTimeout,
			PoolSize:        opts.PoolSize,
			MinIdleConns:    opts.MinIdleConns,
			ConnMaxLifetime: opts.ConnMaxLifetime,
			ConnMaxIdleTime: opts.ConnMaxIdleTime,
			PoolTimeout:     opts.PoolTimeout,
			TLSConfig:       opts.TLSConfig,
		})
	case "sentinel":
		if opts.MasterName == "" {
			return fmt.Errorf("master name is required for sentinel mode")
		}
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       opts.MasterName,
			SentinelAddrs:    opts.Addrs,
			SentinelUsername: opts.Username,
			SentinelPassword: opts.Password,
			Username:         opts.Username,
			Password:         opts.Password,
			DB:               opts.DB,
			MaxRetries:       opts.MaxRetries,
			MinRetryBackoff:  opts.MinRetryBackoff,
			MaxRetryBackoff:  opts.MaxRetryBackoff,
			DialTimeout:      opts.DialTimeout,
			ReadTimeout:      opts.ReadTimeout,
			WriteTimeout:     opts.WriteTimeout,
			PoolSize:         opts.PoolSize,
			MinIdleConns:     opts.MinIdleConns,
			ConnMaxLifetime:  opts.ConnMaxLifetime,
			ConnMaxIdleTime:  opts.ConnMaxIdleTime,
			PoolTimeout:      opts.PoolTimeout,
			TLSConfig:        opts.TLSConfig,
		})
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           opts.Addrs,
			Username:        opts.Username,
			Password:        opts.Password,
			MaxRetries:      opts.MaxRetries,
			MinRetryBackoff: opts.MinRetryBackoff,
			MaxRetryBackoff: opts.MaxRetryBackoff,
			DialTimeout:     opts.DialTimeout,
			ReadTimeout:     opts.ReadTimeout,
			WriteTimeout:    opts.WriteTimeout,
			PoolSize:        opts.PoolSize,
			MinIdleConns:    opts.MinIdleConns,
			ConnMaxLifetime: opts.ConnMaxLifetime,
			ConnMaxIdleTime: opts.ConnMaxIdleTime,
			PoolTimeout:     opts.PoolTimeout,
			TLSConfig:       opts.TLSConfig,
		})
	default:
		return fmt.Errorf("unsupported Redis mode: %s", c.config.Mode)
	}

	// Ping the Redis server
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	c.client = client
	c.connected = true
	klog.Infof("Connected to Redis at %s", c.config.Address)
	return nil
}

// Disconnect disconnects from the database.
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	c.client = nil
	c.connected = false
	klog.Infof("Disconnected from Redis at %s", c.config.Address)
	return nil
}

// Ping checks if the database is reachable.
func (c *Connector) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	return nil
}

// IsConnected returns true if the connector is connected.
func (c *Connector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Name returns the name of the connector.
func (c *Connector) Name() string {
	return c.config.Name
}

// Client returns the underlying client.
func (c *Connector) Client() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Redis returns the underlying Redis client.
func (c *Connector) Redis() redis.UniversalClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// setupTLS sets up TLS for the Redis connection.
func (c *Connector) setupTLS() error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.TLSSkipVerify,
	}

	if !c.config.TLSSkipVerify {
		// Load CA certificate
		if c.config.TLSCAPath != "" {
			caCert, err := os.ReadFile(c.config.TLSCAPath)
			if err != nil {
				return fmt.Errorf("failed to read CA certificate: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to append CA certificate")
			}

			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate and key
		if c.config.TLSCertPath != "" && c.config.TLSKeyPath != "" {
			cert, err := tls.LoadX509KeyPair(c.config.TLSCertPath, c.config.TLSKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load client certificate and key: %w", err)
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	c.tlsConfig = tlsConfig
	return nil
}

// WithConfig sets the configuration.
func WithConfig(config *Config) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			*conn = *config
		}
	}
}

// WithAddress sets the address.
func WithAddress(address string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Address = address
		}
	}
}

// WithUsername sets the username.
func WithUsername(username string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Username = username
		}
	}
}

// WithPassword sets the password.
func WithPassword(password string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Password = password
		}
	}
}

// WithConnectTimeout sets the connect timeout.
func WithConnectTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ConnectTimeout = timeout
		}
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ReadTimeout = timeout
		}
	}
}

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.WriteTimeout = timeout
		}
	}
}

// WithTLS enables TLS for the connection.
func WithTLS(enable bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.EnableTLS = enable
		}
	}
}

// WithTLSSkipVerify sets whether to skip TLS verification.
func WithTLSSkipVerify(skip bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.TLSSkipVerify = skip
		}
	}
}

// WithTLSCertPath sets the path to the TLS certificate.
func WithTLSCertPath(path string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.TLSCertPath = path
		}
	}
}

// WithTLSKeyPath sets the path to the TLS key.
func WithTLSKeyPath(path string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.TLSKeyPath = path
		}
	}
}

// WithTLSCAPath sets the path to the TLS CA certificate.
func WithTLSCAPath(path string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.TLSCAPath = path
		}
	}
}

// WithMode sets the Redis mode.
func WithMode(mode string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Mode = mode
		}
	}
}

// WithMasterName sets the name of the Redis Sentinel master.
func WithMasterName(name string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MasterName = name
		}
	}
}

// WithDB sets the Redis database number.
func WithDB(db int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.DB = db
		}
	}
}

// WithPoolSize sets the maximum number of connections in the pool.
func WithPoolSize(size int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.PoolSize = size
		}
	}
}

// WithMinIdleConns sets the minimum number of idle connections in the pool.
func WithMinIdleConns(n int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MinIdleConns = n
		}
	}
}

// WithDialTimeout sets the timeout for establishing new connections.
func WithDialTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.DialTimeout = timeout
		}
	}
}

// WithPoolTimeout sets the timeout for getting a connection from the pool.
func WithPoolTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.PoolTimeout = timeout
		}
	}
}

// WithIdleTimeout sets the timeout for idle connections.
func WithIdleTimeout(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.IdleTimeout = timeout
		}
	}
}

// WithMaxRetries sets the maximum number of retries before giving up.
func WithMaxRetries(n int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxRetries = n
		}
	}
}

// WithMinRetryBackoff sets the minimum backoff between retries.
func WithMinRetryBackoff(d time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MinRetryBackoff = d
		}
	}
}

// WithMaxRetryBackoff sets the maximum backoff between retries.
func WithMaxRetryBackoff(d time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxRetryBackoff = d
		}
	}
}
