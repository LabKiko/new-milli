package config

import (
	"os"
	"strings"
)

// EnvSource is a source that reads from environment variables
type EnvSource struct {
	prefix string
}

// NewEnvSource creates a new EnvSource
func NewEnvSource(prefix string) Source {
	return &EnvSource{
		prefix: prefix,
	}
}

// Read reads the configuration from environment variables
func (s *EnvSource) Read() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := parts[0]
		value := parts[1]
		
		// Check if the key has the prefix
		if s.prefix != "" && !strings.HasPrefix(key, s.prefix) {
			continue
		}
		
		// Remove the prefix
		if s.prefix != "" {
			key = strings.TrimPrefix(key, s.prefix)
		}
		
		// Convert to lowercase and replace underscores with dots
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", ".")
		
		result[key] = value
	}
	
	return result, nil
}

// Watch watches for changes in environment variables
// Note: This is a no-op as environment variables don't change during runtime
func (s *EnvSource) Watch() (<-chan struct{}, error) {
	return nil, nil
}

// Close closes the source
func (s *EnvSource) Close() error {
	return nil
}
