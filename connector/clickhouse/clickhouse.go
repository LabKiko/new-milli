package clickhouse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/cloudwego/kitex/pkg/klog"
	"new-milli/connector"
)

// Config is the configuration for the ClickHouse connector.
type Config struct {
	connector.Config
	// Params is the parameters for the ClickHouse connection string.
	Params map[string]string
	// Compression is the compression method.
	Compression string
	// Debug enables debug mode.
	Debug bool
	// Settings is the ClickHouse settings.
	Settings map[string]interface{}
	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration
	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
	// ConnOpenStrategy is the connection open strategy.
	ConnOpenStrategy clickhouse.ConnOpenStrategy
	// BlockBufferSize is the block buffer size.
	BlockBufferSize uint8
	// MaxCompressionBuffer is the maximum compression buffer size.
	MaxCompressionBuffer int
	// MaxExecutionTime is the maximum execution time.
	MaxExecutionTime time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Config: connector.Config{
			Name:            "clickhouse",
			Address:         "localhost:9000",
			Username:        "default",
			Password:        "",
			Database:        "default",
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
		Params:               make(map[string]string),
		Compression:          "lz4",
		Debug:                false,
		Settings:             make(map[string]interface{}),
		DialTimeout:          time.Second * 10,
		ConnMaxLifetime:      time.Hour,
		ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10 * 1024 * 1024, // 10MB
		MaxExecutionTime:     time.Minute,
	}
}

// Connector is a ClickHouse connector.
type Connector struct {
	config     *Config
	conn       driver.Conn
	db         *sql.DB
	mu         sync.RWMutex
	connected  bool
	tlsConfig  *tls.Config
}

// New creates a new ClickHouse connector.
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
	var addresses []string
	if strings.Contains(c.config.Address, ",") {
		addresses = strings.Split(c.config.Address, ",")
	} else {
		addresses = []string{c.config.Address}
	}

	// Create ClickHouse options
	options := &clickhouse.Options{
		Addr: addresses,
		Auth: clickhouse.Auth{
			Database: c.config.Database,
			Username: c.config.Username,
			Password: c.config.Password,
		},
		Settings: c.config.Settings,
		Compression: &clickhouse.Compression{
			Method: c.config.Compression,
		},
		Debug:                c.config.Debug,
		DialTimeout:          c.config.DialTimeout,
		MaxOpenConns:         c.config.MaxOpenConns,
		MaxIdleConns:         c.config.MaxIdleConns,
		ConnMaxLifetime:      c.config.ConnMaxLifetime,
		ConnOpenStrategy:     c.config.ConnOpenStrategy,
		BlockBufferSize:      c.config.BlockBufferSize,
		MaxCompressionBuffer: c.config.MaxCompressionBuffer,
		ReadTimeout:          c.config.ReadTimeout,
		WriteTimeout:         c.config.WriteTimeout,
		MaxExecutionTime:     c.config.MaxExecutionTime,
	}

	// Set TLS config if enabled
	if c.config.EnableTLS {
		options.TLS = c.tlsConfig
	}

	// Connect to ClickHouse
	conn, err := clickhouse.Open(options)
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Ping the ClickHouse server
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Create SQL DB
	db := clickhouse.OpenDB(options)
	db.SetMaxIdleConns(c.config.MaxIdleConns)
	db.SetMaxOpenConns(c.config.MaxOpenConns)
	db.SetConnMaxLifetime(c.config.MaxConnLifetime)
	db.SetConnMaxIdleTime(c.config.MaxIdleTime)

	c.conn = conn
	c.db = db
	c.connected = true
	klog.Infof("Connected to ClickHouse at %s", c.config.Address)
	return nil
}

// Disconnect disconnects from the database.
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close ClickHouse connection: %w", err)
	}

	if err := c.db.Close(); err != nil {
		return fmt.Errorf("failed to close ClickHouse DB: %w", err)
	}

	c.conn = nil
	c.db = nil
	c.connected = false
	klog.Infof("Disconnected from ClickHouse at %s", c.config.Address)
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
	if err := c.conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping ClickHouse: %w", err)
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
	return c.conn
}

// Conn returns the underlying ClickHouse connection.
func (c *Connector) Conn() driver.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// DB returns the underlying SQL DB.
func (c *Connector) DB() *sql.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// setupTLS sets up TLS for the ClickHouse connection.
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

// WithDatabase sets the database.
func WithDatabase(database string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Database = database
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

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxIdleConns = n
		}
	}
}

// WithMaxOpenConns sets the maximum number of open connections.
func WithMaxOpenConns(n int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxOpenConns = n
		}
	}
}

// WithMaxConnLifetime sets the maximum lifetime of a connection.
func WithMaxConnLifetime(d time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxConnLifetime = d
		}
	}
}

// WithMaxIdleTime sets the maximum idle time of a connection.
func WithMaxIdleTime(d time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxIdleTime = d
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

// WithParams sets the parameters for the ClickHouse connection string.
func WithParams(params map[string]string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Params = params
		}
	}
}

// WithParam sets a parameter for the ClickHouse connection string.
func WithParam(key, value string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			if conn.Params == nil {
				conn.Params = make(map[string]string)
			}
			conn.Params[key] = value
		}
	}
}

// WithCompression sets the compression method.
func WithCompression(compression string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Compression = compression
		}
	}
}

// WithDebug enables debug mode.
func WithDebug(debug bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Debug = debug
		}
	}
}

// WithSettings sets the ClickHouse settings.
func WithSettings(settings map[string]interface{}) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Settings = settings
		}
	}
}

// WithSetting sets a ClickHouse setting.
func WithSetting(key string, value interface{}) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			if conn.Settings == nil {
				conn.Settings = make(map[string]interface{})
			}
			conn.Settings[key] = value
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

// WithConnOpenStrategy sets the connection open strategy.
func WithConnOpenStrategy(strategy clickhouse.ConnOpenStrategy) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ConnOpenStrategy = strategy
		}
	}
}

// WithBlockBufferSize sets the block buffer size.
func WithBlockBufferSize(size uint8) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.BlockBufferSize = size
		}
	}
}

// WithMaxCompressionBuffer sets the maximum compression buffer size.
func WithMaxCompressionBuffer(size int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxCompressionBuffer = size
		}
	}
}

// WithMaxExecutionTime sets the maximum execution time.
func WithMaxExecutionTime(timeout time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxExecutionTime = timeout
		}
	}
}
