package mongo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"new-milli/connector"
)

// Config is the configuration for the MongoDB connector.
type Config struct {
	connector.Config
	// ReplicaSet is the name of the replica set.
	ReplicaSet string
	// AuthSource is the name of the database used for authentication.
	AuthSource string
	// AuthMechanism is the authentication mechanism.
	AuthMechanism string
	// Direct specifies whether to connect directly to the server.
	Direct bool
	// RetryWrites specifies whether to retry writes.
	RetryWrites bool
	// RetryReads specifies whether to retry reads.
	RetryReads bool
	// MaxPoolSize is the maximum number of connections in the pool.
	MaxPoolSize uint64
	// MinPoolSize is the minimum number of connections in the pool.
	MinPoolSize uint64
	// MaxConnIdleTime is the maximum idle time for a connection.
	MaxConnIdleTime time.Duration
	// ReadPreference is the read preference.
	ReadPreference string
	// ReadConcern is the read concern.
	ReadConcern string
	// WriteConcern is the write concern.
	WriteConcern string
	// AppName is the application name.
	AppName string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Config: connector.Config{
			Name:            "mongo",
			Address:         "mongodb://localhost:27017",
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
		ReplicaSet:      "",
		AuthSource:      "admin",
		AuthMechanism:   "",
		Direct:          false,
		RetryWrites:     true,
		RetryReads:      true,
		MaxPoolSize:     100,
		MinPoolSize:     0,
		MaxConnIdleTime: time.Minute * 30,
		ReadPreference:  "primary",
		ReadConcern:     "local",
		WriteConcern:    "majority",
		AppName:         "new-milli",
	}
}

// Connector is a MongoDB connector.
type Connector struct {
	config    *Config
	client    *mongo.Client
	db        *mongo.Database
	mu        sync.RWMutex
	connected bool
	tlsConfig *tls.Config
}

// New creates a new MongoDB connector.
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

	// Create client options
	clientOptions := options.Client().
		ApplyURI(c.config.Address).
		SetConnectTimeout(c.config.ConnectTimeout).
		SetMaxConnIdleTime(c.config.MaxIdleTime).
		SetMaxConnecting(uint64(c.config.MaxOpenConns)).
		SetMaxPoolSize(c.config.MaxPoolSize).
		SetMinPoolSize(c.config.MinPoolSize).
		SetRetryWrites(c.config.RetryWrites).
		SetRetryReads(c.config.RetryReads).
		SetDirect(c.config.Direct).
		SetAppName(c.config.AppName)

	// Set credentials if username and password are provided
	if c.config.Username != "" && c.config.Password != "" {
		clientOptions.SetAuth(options.Credential{
			Username:      c.config.Username,
			Password:      c.config.Password,
			AuthSource:    c.config.AuthSource,
			AuthMechanism: c.config.AuthMechanism,
		})
	}

	// Set replica set if provided
	if c.config.ReplicaSet != "" {
		clientOptions.SetReplicaSet(c.config.ReplicaSet)
	}

	// Set TLS config if enabled
	if c.config.EnableTLS {
		clientOptions.SetTLSConfig(c.tlsConfig)
	}

	// Set read preference
	switch c.config.ReadPreference {
	case "primary":
		clientOptions.SetReadPreference(readpref.Primary())
	case "primaryPreferred":
		clientOptions.SetReadPreference(readpref.PrimaryPreferred())
	case "secondary":
		clientOptions.SetReadPreference(readpref.Secondary())
	case "secondaryPreferred":
		clientOptions.SetReadPreference(readpref.SecondaryPreferred())
	case "nearest":
		clientOptions.SetReadPreference(readpref.Nearest())
	}

	// Set read concern
	if c.config.ReadConcern != "" {
		clientOptions.SetReadConcern(&readconcern.ReadConcern{Level: c.config.ReadConcern})
	}

	// Set write concern
	if c.config.WriteConcern != "" {
		clientOptions.SetWriteConcern(&writeconcern.WriteConcern{W: c.config.WriteConcern})
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the MongoDB server
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Set the database if provided
	var db *mongo.Database
	if c.config.Database != "" {
		db = client.Database(c.config.Database)
	}

	c.client = client
	c.db = db
	c.connected = true
	klog.Infof("Connected to MongoDB at %s", c.config.Address)
	return nil
}

// Disconnect disconnects from the database.
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	c.client = nil
	c.db = nil
	c.connected = false
	klog.Infof("Disconnected from MongoDB at %s", c.config.Address)
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
	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
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

// Mongo returns the underlying MongoDB client.
func (c *Connector) Mongo() *mongo.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Database returns the MongoDB database.
func (c *Connector) Database() *mongo.Database {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// Collection returns a MongoDB collection.
func (c *Connector) Collection(name string) *mongo.Collection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.db == nil {
		return nil
	}
	return c.db.Collection(name)
}

// setupTLS sets up TLS for the MongoDB connection.
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

// WithReplicaSet sets the name of the replica set.
func WithReplicaSet(replicaSet string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ReplicaSet = replicaSet
		}
	}
}

// WithAuthSource sets the name of the database used for authentication.
func WithAuthSource(authSource string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.AuthSource = authSource
		}
	}
}

// WithAuthMechanism sets the authentication mechanism.
func WithAuthMechanism(authMechanism string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.AuthMechanism = authMechanism
		}
	}
}

// WithDirect specifies whether to connect directly to the server.
func WithDirect(direct bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Direct = direct
		}
	}
}

// WithRetryWrites specifies whether to retry writes.
func WithRetryWrites(retryWrites bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.RetryWrites = retryWrites
		}
	}
}

// WithRetryReads specifies whether to retry reads.
func WithRetryReads(retryReads bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.RetryReads = retryReads
		}
	}
}

// WithMaxPoolSize sets the maximum number of connections in the pool.
func WithMaxPoolSize(maxPoolSize uint64) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxPoolSize = maxPoolSize
		}
	}
}

// WithMinPoolSize sets the minimum number of connections in the pool.
func WithMinPoolSize(minPoolSize uint64) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MinPoolSize = minPoolSize
		}
	}
}

// WithMaxConnIdleTime sets the maximum idle time for a connection.
func WithMaxConnIdleTime(maxConnIdleTime time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxConnIdleTime = maxConnIdleTime
		}
	}
}

// WithReadPreference sets the read preference.
func WithReadPreference(readPreference string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ReadPreference = readPreference
		}
	}
}

// WithReadConcern sets the read concern.
func WithReadConcern(readConcern string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ReadConcern = readConcern
		}
	}
}

// WithWriteConcern sets the write concern.
func WithWriteConcern(writeConcern string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.WriteConcern = writeConcern
		}
	}
}

// WithAppName sets the application name.
func WithAppName(appName string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.AppName = appName
		}
	}
}
