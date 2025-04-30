package config

import (
	"sync"
)

// MemorySource is a source that stores configuration in memory
type MemorySource struct {
	values map[string]interface{}
	mu     sync.RWMutex
	ch     chan struct{}
}

// NewMemorySource creates a new MemorySource
func NewMemorySource(values map[string]interface{}) Source {
	if values == nil {
		values = make(map[string]interface{})
	}
	
	return &MemorySource{
		values: values,
		ch:     make(chan struct{}),
	}
}

// Read reads the configuration from memory
func (s *MemorySource) Read() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Create a copy to prevent modification
	result := make(map[string]interface{}, len(s.values))
	for k, v := range s.values {
		result[k] = v
	}
	
	return result, nil
}

// Watch watches for changes in memory
func (s *MemorySource) Watch() (<-chan struct{}, error) {
	return s.ch, nil
}

// Close closes the source
func (s *MemorySource) Close() error {
	return nil
}

// Set sets a value in memory
func (s *MemorySource) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.values[key] = value
	
	// Notify watchers
	select {
	case s.ch <- struct{}{}:
	default:
		// Non-blocking send to prevent goroutine leak
	}
}

// Delete deletes a value from memory
func (s *MemorySource) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.values, key)
	
	// Notify watchers
	select {
	case s.ch <- struct{}{}:
	default:
		// Non-blocking send to prevent goroutine leak
	}
}
