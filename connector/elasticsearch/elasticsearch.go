package elasticsearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/elastic/go-elasticsearch/v8"
	"new-milli/connector"
)

// Config is the configuration for the Elasticsearch connector.
type Config struct {
	connector.Config
	// CloudID is the Elastic Cloud ID.
	CloudID string
	// APIKey is the API key for authentication.
	APIKey string
	// ServiceToken is the service token for authentication.
	ServiceToken string
	// CACert is the CA certificate for TLS.
	CACert string
	// RetryOnStatus is the list of status codes to retry on.
	RetryOnStatus []int
	// MaxRetries is the maximum number of retries.
	MaxRetries int
	// RetryBackoff is the backoff function for retries.
	RetryBackoff func(attempt int) time.Duration
	// CompressRequestBody specifies whether to compress request bodies.
	CompressRequestBody bool
	// DiscoverNodesOnStart specifies whether to discover nodes on start.
	DiscoverNodesOnStart bool
	// DiscoverNodesInterval is the interval for discovering nodes.
	DiscoverNodesInterval time.Duration
	// EnableMetrics specifies whether to enable metrics.
	EnableMetrics bool
	// EnableDebugLogger specifies whether to enable debug logging.
	EnableDebugLogger bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Config: connector.Config{
			Name:            "elasticsearch",
			Address:         "http://localhost:9200",
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
		CloudID:               "",
		APIKey:                "",
		ServiceToken:          "",
		CACert:                "",
		RetryOnStatus:         []int{502, 503, 504, 429},
		MaxRetries:            3,
		RetryBackoff:          nil,
		CompressRequestBody:   false,
		DiscoverNodesOnStart:  true,
		DiscoverNodesInterval: time.Minute * 5,
		EnableMetrics:         false,
		EnableDebugLogger:     false,
	}
}

// Connector is an Elasticsearch connector.
type Connector struct {
	config    *Config
	client    *elasticsearch.Client
	mu        sync.RWMutex
	connected bool
	tlsConfig *tls.Config
}

// New creates a new Elasticsearch connector.
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

	// Create Elasticsearch config
	esConfig := elasticsearch.Config{
		Addresses:             addresses,
		Username:              c.config.Username,
		Password:              c.config.Password,
		CloudID:               c.config.CloudID,
		APIKey:                c.config.APIKey,
		ServiceToken:          c.config.ServiceToken,
		RetryOnStatus:         c.config.RetryOnStatus,
		DisableRetry:          c.config.MaxRetries == 0,
		MaxRetries:            c.config.MaxRetries,
		RetryBackoff:          c.config.RetryBackoff,
		CompressRequestBody:   c.config.CompressRequestBody,
		DiscoverNodesOnStart:  c.config.DiscoverNodesOnStart,
		DiscoverNodesInterval: c.config.DiscoverNodesInterval,
		EnableMetrics:         c.config.EnableMetrics,
		EnableDebugLogger:     c.config.EnableDebugLogger,
	}

	// Set TLS config if enabled
	if c.config.EnableTLS {
		esConfig.Transport = &http.Transport{
			TLSClientConfig: c.tlsConfig,
		}
	}

	// Set CA certificate if provided
	if c.config.CACert != "" {
		esConfig.CACert = []byte(c.config.CACert)
	}

	// Create Elasticsearch client
	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Ping the Elasticsearch server
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	res, err := client.Ping(
		client.Ping.WithContext(ctx),
		client.Ping.WithHuman(),
		client.Ping.WithPretty(),
	)
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to ping Elasticsearch: %s", res.String())
	}

	c.client = client
	c.connected = true
	klog.Infof("Connected to Elasticsearch at %s", c.config.Address)
	return nil
}

// Disconnect disconnects from the database.
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	// Elasticsearch client doesn't have a disconnect method
	c.client = nil
	c.connected = false
	klog.Infof("Disconnected from Elasticsearch at %s", c.config.Address)
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
	res, err := c.client.Ping(
		c.client.Ping.WithContext(ctx),
		c.client.Ping.WithHuman(),
		c.client.Ping.WithPretty(),
	)
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to ping Elasticsearch: %s", res.String())
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

// Elasticsearch returns the underlying Elasticsearch client.
func (c *Connector) Elasticsearch() *elasticsearch.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// setupTLS sets up TLS for the Elasticsearch connection.
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

// WithCloudID sets the Elastic Cloud ID.
func WithCloudID(cloudID string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.CloudID = cloudID
		}
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(apiKey string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.APIKey = apiKey
		}
	}
}

// WithServiceToken sets the service token for authentication.
func WithServiceToken(serviceToken string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ServiceToken = serviceToken
		}
	}
}

// WithCACert sets the CA certificate for TLS.
func WithCACert(caCert string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.CACert = caCert
		}
	}
}

// WithRetryOnStatus sets the list of status codes to retry on.
func WithRetryOnStatus(retryOnStatus []int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.RetryOnStatus = retryOnStatus
		}
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MaxRetries = maxRetries
		}
	}
}

// WithRetryBackoff sets the backoff function for retries.
func WithRetryBackoff(retryBackoff func(attempt int) time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.RetryBackoff = retryBackoff
		}
	}
}

// WithCompressRequestBody specifies whether to compress request bodies.
func WithCompressRequestBody(compress bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.CompressRequestBody = compress
		}
	}
}

// WithDiscoverNodesOnStart specifies whether to discover nodes on start.
func WithDiscoverNodesOnStart(discover bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.DiscoverNodesOnStart = discover
		}
	}
}

// WithDiscoverNodesInterval sets the interval for discovering nodes.
func WithDiscoverNodesInterval(interval time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.DiscoverNodesInterval = interval
		}
	}
}

// WithEnableMetrics specifies whether to enable metrics.
func WithEnableMetrics(enable bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.EnableMetrics = enable
		}
	}
}

// WithEnableDebugLogger specifies whether to enable debug logging.
func WithEnableDebugLogger(enable bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.EnableDebugLogger = enable
		}
	}
}
