package config

import (
	"errors"
	"sync"
)

var (
	// ErrNotFound is returned when a key is not found
	ErrNotFound = errors.New("key not found in config")
	// ErrInvalidType is returned when a type assertion fails
	ErrInvalidType = errors.New("invalid type assertion")
)

// Config is the interface for configuration
type Config interface {
	// Get returns the value associated with the key
	Get(key string) (interface{}, error)
	// Set sets the value for the key
	Set(key string, value interface{}) error
	// GetString returns the value associated with the key as a string
	GetString(key string) (string, error)
	// GetInt returns the value associated with the key as an int
	GetInt(key string) (int, error)
	// GetBool returns the value associated with the key as a bool
	GetBool(key string) (bool, error)
	// GetFloat returns the value associated with the key as a float64
	GetFloat(key string) (float64, error)
	// GetStringMap returns the value associated with the key as a map[string]interface{}
	GetStringMap(key string) (map[string]interface{}, error)
	// GetStringSlice returns the value associated with the key as a []string
	GetStringSlice(key string) ([]string, error)
	// GetStringMapString returns the value associated with the key as a map[string]string
	GetStringMapString(key string) (map[string]string, error)
	// Has checks if the key exists
	Has(key string) bool
	// Load loads configuration from a source
	Load() error
	// Watch watches for changes in the configuration
	Watch() (<-chan struct{}, error)
	// Close closes the configuration
	Close() error
}

// DefaultConfig is the default implementation of Config
type DefaultConfig struct {
	sync.RWMutex
	values map[string]interface{}
	source Source
}

// NewConfig creates a new Config with the given source
func NewConfig(source Source) Config {
	return &DefaultConfig{
		values: make(map[string]interface{}),
		source: source,
	}
}

// Get returns the value associated with the key
func (c *DefaultConfig) Get(key string) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()

	if value, ok := c.values[key]; ok {
		return value, nil
	}

	return nil, ErrNotFound
}

// Set sets the value for the key
func (c *DefaultConfig) Set(key string, value interface{}) error {
	c.Lock()
	defer c.Unlock()

	c.values[key] = value
	return nil
}

// GetString returns the value associated with the key as a string
func (c *DefaultConfig) GetString(key string) (string, error) {
	value, err := c.Get(key)
	if err != nil {
		return "", err
	}

	if str, ok := value.(string); ok {
		return str, nil
	}

	return "", ErrInvalidType
}

// GetInt returns the value associated with the key as an int
func (c *DefaultConfig) GetInt(key string) (int, error) {
	value, err := c.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	}

	return 0, ErrInvalidType
}

// GetBool returns the value associated with the key as a bool
func (c *DefaultConfig) GetBool(key string) (bool, error) {
	value, err := c.Get(key)
	if err != nil {
		return false, err
	}

	if b, ok := value.(bool); ok {
		return b, nil
	}

	return false, ErrInvalidType
}

// GetFloat returns the value associated with the key as a float64
func (c *DefaultConfig) GetFloat(key string) (float64, error) {
	value, err := c.Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	}

	return 0, ErrInvalidType
}

// GetStringMap returns the value associated with the key as a map[string]interface{}
func (c *DefaultConfig) GetStringMap(key string) (map[string]interface{}, error) {
	value, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	if m, ok := value.(map[string]interface{}); ok {
		return m, nil
	}

	return nil, ErrInvalidType
}

// GetStringSlice returns the value associated with the key as a []string
func (c *DefaultConfig) GetStringSlice(key string) ([]string, error) {
	value, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			} else {
				return nil, ErrInvalidType
			}
		}
		return result, nil
	}

	return nil, ErrInvalidType
}

// GetStringMapString returns the value associated with the key as a map[string]string
func (c *DefaultConfig) GetStringMapString(key string) (map[string]string, error) {
	value, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case map[string]string:
		return v, nil
	case map[string]interface{}:
		result := make(map[string]string)
		for k, val := range v {
			if str, ok := val.(string); ok {
				result[k] = str
			} else {
				return nil, ErrInvalidType
			}
		}
		return result, nil
	}

	return nil, ErrInvalidType
}

// Has checks if the key exists
func (c *DefaultConfig) Has(key string) bool {
	c.RLock()
	defer c.RUnlock()

	_, ok := c.values[key]
	return ok
}

// Load loads configuration from a source
func (c *DefaultConfig) Load() error {
	c.Lock()
	defer c.Unlock()

	values, err := c.source.Read()
	if err != nil {
		return err
	}

	c.values = values
	return nil
}

// Watch watches for changes in the configuration
func (c *DefaultConfig) Watch() (<-chan struct{}, error) {
	return c.source.Watch()
}

// Close closes the configuration
func (c *DefaultConfig) Close() error {
	return c.source.Close()
}
