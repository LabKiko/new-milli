package mysql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	gormlogger "gorm.io/gorm/logger"
	"os"

	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"new-milli/connector"
	"new-milli/logger"
)

// Config is the configuration for the MySQL connector.
type Config struct {
	connector.Config
	// Params is the parameters for the MySQL connection string.
	Params map[string]string
	// ParseTime parses time values to time.Time.
	ParseTime bool
	// Loc is the location for time.Time values.
	Loc *time.Location
	// Collation is the collation for the connection.
	Collation string
	// AllowNativePasswords allows the native password authentication method.
	AllowNativePasswords bool
	// AllowOldPasswords allows the old password authentication method.
	AllowOldPasswords bool
	// ClientFoundRows returns the number of found rows instead of affected rows.
	ClientFoundRows bool
	// MultiStatements allows multiple statements in one query.
	MultiStatements bool
	// RejectReadOnly rejects read-only connections.
	RejectReadOnly bool
	// GormConfig is the GORM configuration.
	GormConfig *gorm.Config
	// Logger is the logger for the connector.
	Logger logger.Logger
	// LogLevel is the log level for GORM.
	LogLevel logger.Level
	// SlowThreshold is the threshold for slow queries.
	SlowThreshold time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	// Create a database-specific logger
	dbLogger := logger.New(nil).WithFields(logger.F("component", "mysql"))

	return &Config{
		Config: connector.Config{
			Name:            "mysql",
			Address:         "localhost:3306",
			Username:        "root",
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
		Params:               make(map[string]string),
		ParseTime:            true,
		Loc:                  time.UTC,
		Collation:            "utf8mb4_general_ci",
		AllowNativePasswords: true,
		AllowOldPasswords:    false,
		ClientFoundRows:      false,
		MultiStatements:      false,
		RejectReadOnly:       false,
		Logger:               dbLogger,
		LogLevel:             logger.InfoLevel,
		SlowThreshold:        time.Second,
	}
}

// Connector is a MySQL connector.
type Connector struct {
	config    *Config
	db        *gorm.DB
	sqlDB     *sql.DB
	mu        sync.RWMutex
	connected bool
	tlsConfig *tls.Config
	dsn       string
}

// New creates a new MySQL connector.
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

	// Build DSN
	c.dsn = c.buildDSN()

	// Setup TLS if enabled
	if c.config.EnableTLS {
		if err := c.setupTLS(); err != nil {
			return err
		}
	}

	// Configure GORM
	gormConfig := c.config.GormConfig
	if gormConfig == nil {
		// Use our custom logger adapter with default settings
		gormLogger := logger.NewGormLogger(c.config.Logger).
			WithSlowThreshold(c.config.SlowThreshold).
			WithLogLevel(gormlogger.LogLevel(c.config.LogLevel)).
			WithIgnoreRecordNotFoundError(true)

		// Add trace information if available in the context
		if traceInfo := logger.TraceInfoFromContext(ctx); traceInfo != nil {
			c.config.Logger = c.config.Logger.WithTraceInfo(traceInfo)
		}

		gormConfig = &gorm.Config{
			Logger: gormLogger,
		}
	}

	// Open connection
	db, err := gorm.Open(mysql.Open(c.dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Get the underlying SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(c.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(c.config.MaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(c.config.MaxIdleTime)

	// Ping the database
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}

	c.db = db
	c.sqlDB = sqlDB
	c.connected = true
	c.config.Logger.Infof("Connected to MySQL at %s", c.config.Address)
	return nil
}

// Disconnect disconnects from the database.
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return connector.ErrNotConnected
	}

	if err := c.sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close MySQL connection: %w", err)
	}

	c.db = nil
	c.sqlDB = nil
	c.connected = false
	c.config.Logger.Infof("Disconnected from MySQL at %s", c.config.Address)
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
	if err := c.sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
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
	return c.db
}

// DB returns the underlying GORM database.
func (c *Connector) DB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// buildDSN builds the MySQL DSN.
func (c *Connector) buildDSN() string {
	// Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		c.config.Username,
		c.config.Password,
		c.config.Address,
		c.config.Database)

	// Add parameters
	params := make(map[string]string)

	// Add default parameters
	params["timeout"] = c.config.ConnectTimeout.String()
	params["readTimeout"] = c.config.ReadTimeout.String()
	params["writeTimeout"] = c.config.WriteTimeout.String()
	params["parseTime"] = fmt.Sprintf("%t", c.config.ParseTime)
	params["loc"] = c.config.Loc.String()
	params["collation"] = c.config.Collation
	params["allowNativePasswords"] = fmt.Sprintf("%t", c.config.AllowNativePasswords)
	params["allowOldPasswords"] = fmt.Sprintf("%t", c.config.AllowOldPasswords)
	params["clientFoundRows"] = fmt.Sprintf("%t", c.config.ClientFoundRows)
	params["multiStatements"] = fmt.Sprintf("%t", c.config.MultiStatements)
	params["rejectReadOnly"] = fmt.Sprintf("%t", c.config.RejectReadOnly)

	// Add TLS parameter if enabled
	if c.config.EnableTLS {
		if c.config.TLSSkipVerify {
			params["tls"] = "skip-verify"
		} else {
			params["tls"] = "true"
		}
	}

	// Add custom parameters
	for k, v := range c.config.Params {
		params[k] = v
	}

	// Build query string
	if len(params) > 0 {
		dsn += "?"
		first := true
		for k, v := range params {
			if !first {
				dsn += "&"
			}
			dsn += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
	}

	return dsn
}

// setupTLS sets up TLS for the MySQL connection.
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

// WithParseTime sets whether to parse time values to time.Time.
func WithParseTime(parse bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ParseTime = parse
		}
	}
}

// WithLoc sets the location for time.Time values.
func WithLoc(loc *time.Location) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Loc = loc
		}
	}
}

// WithCollation sets the collation for the connection.
func WithCollation(collation string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Collation = collation
		}
	}
}

// WithAllowNativePasswords sets whether to allow the native password authentication method.
func WithAllowNativePasswords(allow bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.AllowNativePasswords = allow
		}
	}
}

// WithAllowOldPasswords sets whether to allow the old password authentication method.
func WithAllowOldPasswords(allow bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.AllowOldPasswords = allow
		}
	}
}

// WithClientFoundRows sets whether to return the number of found rows instead of affected rows.
func WithClientFoundRows(enable bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.ClientFoundRows = enable
		}
	}
}

// WithMultiStatements sets whether to allow multiple statements in one query.
func WithMultiStatements(enable bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.MultiStatements = enable
		}
	}
}

// WithRejectReadOnly sets whether to reject read-only connections.
func WithRejectReadOnly(reject bool) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.RejectReadOnly = reject
		}
	}
}

// WithParams sets the parameters for the MySQL connection string.
func WithParams(params map[string]string) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Params = params
		}
	}
}

// WithParam sets a parameter for the MySQL connection string.
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

// WithGormConfig sets the GORM configuration.
func WithGormConfig(config *gorm.Config) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.GormConfig = config
		}
	}
}

// WithLogLevel sets the log level for GORM.
func WithLogLevel(level logger.Level) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.LogLevel = level
		}
	}
}

// WithSlowThreshold sets the threshold for slow queries.
func WithSlowThreshold(threshold time.Duration) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.SlowThreshold = threshold
		}
	}
}

// WithLogger sets the logger.
func WithLogger(log logger.Logger) connector.Option {
	return func(c interface{}) {
		if conn, ok := c.(*Config); ok {
			conn.Logger = log
		}
	}
}
