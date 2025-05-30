package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// FileSource is a source that reads from a file
type FileSource struct {
	path          string
	format        string
	watchInterval time.Duration
	done          chan struct{}
	mu            sync.RWMutex
	watching      bool
}

// NewFileSource creates a new FileSource
func NewFileSource(path string, opts ...FileOption) Source {
	options := defaultFileOptions()

	for _, opt := range opts {
		opt(options)
	}

	// Determine format from file extension if not specified
	if options.format == "" {
		options.format = formatFromPath(path)
	}

	return &FileSource{
		path:          path,
		format:        options.format,
		watchInterval: options.watchInterval,
		done:          make(chan struct{}),
	}
}

// Read reads the configuration from the file
func (s *FileSource) Read() (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	return s.unmarshal(data)
}

// Watch watches for changes in the file
func (s *FileSource) Watch() (<-chan struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watching {
		return nil, errors.New("already watching")
	}

	// Check if file exists
	if _, err := os.Stat(s.path); err != nil {
		return nil, err
	}

	s.watching = true
	ch := make(chan struct{})

	go func() {
		defer close(ch)

		lastModTime := time.Time{}
		ticker := time.NewTicker(s.watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(s.path)
				if err != nil {
					continue
				}

				if info.ModTime().After(lastModTime) {
					lastModTime = info.ModTime()
					select {
					case ch <- struct{}{}:
					default:
						// Non-blocking send to prevent goroutine leak
					}
				}
			case <-s.done:
				return
			}
		}
	}()

	return ch, nil
}

// Close stops watching the file
func (s *FileSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watching {
		close(s.done)
		s.watching = false
	}

	return nil
}

// unmarshal unmarshals the data based on the format
func (s *FileSource) unmarshal(data []byte) (map[string]interface{}, error) {
	var nested map[string]interface{}

	switch s.format {
	case "json":
		if err := json.Unmarshal(data, &nested); err != nil {
			return nil, err
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &nested); err != nil {
			return nil, err
		}
	case "toml":
		if err := toml.Unmarshal(data, &nested); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", s.format)
	}

	return flattenMap(nested, ""), nil
}

// flattenMap takes a nested map and flattens it, prefixing keys with dot notation.
func flattenMap(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		newKey := k
		if prefix != "" {
			newKey = prefix + "." + k
		}

		if subMap, ok := v.(map[string]interface{}); ok {
			// If the value is a map, recurse
			for sk, sv := range flattenMap(subMap, newKey) {
				result[sk] = sv
			}
		} else if subMap2, ok := v.(map[interface{}]interface{}); ok {
            // Handle map[interface{}]interface{} which can come from YAML
            genericSubMap := make(map[string]interface{})
            for ik, iv := range subMap2 {
                if strKey, ok := ik.(string); ok {
                    genericSubMap[strKey] = iv
                } else {
                    // Or skip/error if keys are not strings
                }
            }
            for sk, sv := range flattenMap(genericSubMap, newKey) {
				result[sk] = sv
			}
        } else {
			// Otherwise, it's a leaf value
			result[newKey] = v
		}
	}
	return result
}

// formatFromPath determines the format from the file path
func formatFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}

	// Remove the dot
	return ext[1:]
}

// FileOption is a function that configures a FileSource
type FileOption func(*fileOptions)

type fileOptions struct {
	format        string
	watchInterval time.Duration
}

func defaultFileOptions() *fileOptions {
	return &fileOptions{
		watchInterval: 5 * time.Second,
	}
}

// WithFormat sets the format of the file
func WithFormat(format string) FileOption {
	return func(o *fileOptions) {
		o.format = format
	}
}

// WithWatchInterval sets the interval for watching the file
func WithWatchInterval(interval time.Duration) FileOption {
	return func(o *fileOptions) {
		o.watchInterval = interval
	}
}
