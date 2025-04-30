package config

// Source is the interface for configuration sources
type Source interface {
	// Read reads the configuration from the source
	Read() (map[string]interface{}, error)
	// Watch watches for changes in the source
	Watch() (<-chan struct{}, error)
	// Close closes the source
	Close() error
}

// CompositeSource is a source that combines multiple sources
type CompositeSource struct {
	sources []Source
}

// NewCompositeSource creates a new CompositeSource
func NewCompositeSource(sources ...Source) Source {
	return &CompositeSource{
		sources: sources,
	}
}

// Read reads the configuration from all sources
func (s *CompositeSource) Read() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Read from each source in order, later sources override earlier ones
	for _, source := range s.sources {
		values, err := source.Read()
		if err != nil {
			return nil, err
		}

		// Merge values
		for k, v := range values {
			result[k] = v
		}
	}

	return result, nil
}

// Watch watches for changes in any source
func (s *CompositeSource) Watch() (<-chan struct{}, error) {
	ch := make(chan struct{})
	
	for _, source := range s.sources {
		sourceCh, err := source.Watch()
		if err != nil {
			return nil, err
		}
		
		if sourceCh != nil {
			go func(sourceCh <-chan struct{}) {
				for range sourceCh {
					// Notify when any source changes
					select {
					case ch <- struct{}{}:
					default:
						// Non-blocking send to prevent goroutine leak
					}
				}
			}(sourceCh)
		}
	}
	
	return ch, nil
}

// Close closes all sources
func (s *CompositeSource) Close() error {
	for _, source := range s.sources {
		if err := source.Close(); err != nil {
			return err
		}
	}
	
	return nil
}
