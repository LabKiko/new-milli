package config

import (
	"sync"
)

var (
	// global is the global configuration manager
	global *Manager
	once   sync.Once
)

// Manager manages multiple configurations
type Manager struct {
	configs map[string]Config
	mu      sync.RWMutex
}

// NewManager creates a new Manager
func NewManager() *Manager {
	return &Manager{
		configs: make(map[string]Config),
	}
}

// Global returns the global configuration manager
func Global() *Manager {
	once.Do(func() {
		global = NewManager()
	})
	
	return global
}

// Register registers a configuration with the manager
func (m *Manager) Register(name string, config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.configs[name] = config
}

// Get returns a configuration by name
func (m *Manager) Get(name string) Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.configs[name]
}

// LoadAll loads all configurations
func (m *Manager) LoadAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, config := range m.configs {
		if err := config.Load(); err != nil {
			return err
		}
	}
	
	return nil
}

// CloseAll closes all configurations
func (m *Manager) CloseAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, config := range m.configs {
		if err := config.Close(); err != nil {
			return err
		}
	}
	
	return nil
}
