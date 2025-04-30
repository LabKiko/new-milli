package consul

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"new-milli/registry"
)

var (
	_ registry.Registry = (*Registry)(nil)
	_ registry.Watcher  = (*watcher)(nil)
)

// Registry is consul registry.
type Registry struct {
	client  *api.Client
	options registry.Options
	sync.RWMutex
	registrations map[string]*api.AgentServiceRegistration
}

// New creates a new consul registry.
func New(opts ...registry.Option) (registry.Registry, error) {
	options := registry.Options{
		Timeout: time.Second * 10,
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	// Default to localhost
	if len(options.Addrs) == 0 {
		options.Addrs = []string{"127.0.0.1:8500"}
	}

	// Create consul client
	config := api.DefaultConfig()
	config.Address = options.Addrs[0]
	if options.Secure {
		config.Scheme = "https"
	}
	if len(options.Username) > 0 && len(options.Password) > 0 {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: options.Username,
			Password: options.Password,
		}
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Registry{
		client:        client,
		options:       options,
		registrations: make(map[string]*api.AgentServiceRegistration),
	}, nil
}

// Register registers a service.
func (r *Registry) Register(ctx context.Context, service *registry.ServiceInfo) error {
	if len(service.Nodes) == 0 {
		return fmt.Errorf("require at least one node")
	}

	// Create check
	check := &api.AgentServiceCheck{
		TTL:                            fmt.Sprintf("%ds", 30),
		DeregisterCriticalServiceAfter: "1m",
	}

	r.Lock()
	defer r.Unlock()

	// Register each node
	for _, node := range service.Nodes {
		registration := &api.AgentServiceRegistration{
			ID:      node.ID,
			Name:    service.Name,
			Tags:    []string{service.Version},
			Address: node.Address,
			Meta:    node.Metadata,
			Check:   check,
		}

		// Register the service
		if err := r.client.Agent().ServiceRegister(registration); err != nil {
			return err
		}

		// Save the registration
		r.registrations[node.ID] = registration
	}

	return nil
}

// Deregister deregisters a service.
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInfo) error {
	r.Lock()
	defer r.Unlock()

	for _, node := range service.Nodes {
		// Deregister the service
		if err := r.client.Agent().ServiceDeregister(node.ID); err != nil {
			return err
		}

		// Delete the registration
		delete(r.registrations, node.ID)
	}

	return nil
}

// GetService gets a service.
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInfo, error) {
	services, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	serviceMap := make(map[string]*registry.ServiceInfo)
	for _, service := range services {
		// Get the version from the tags
		version := "latest"
		if len(service.Service.Tags) > 0 {
			version = service.Service.Tags[0]
		}

		// Get or create the service
		s, ok := serviceMap[version]
		if !ok {
			s = &registry.ServiceInfo{
				Name:     service.Service.Service,
				Version:  version,
				Metadata: service.Service.Meta,
			}
			serviceMap[version] = s
		}

		// Add the node
		s.Nodes = append(s.Nodes, &registry.Node{
			ID:       service.Service.ID,
			Address:  fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port),
			Metadata: service.Service.Meta,
		})
	}

	// Convert the map to a slice
	var result []*registry.ServiceInfo
	for _, service := range serviceMap {
		result = append(result, service)
	}

	return result, nil
}

// Watch creates a watcher.
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return newWatcher(ctx, r, serviceName)
}

// watcher is a service watcher.
type watcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	r      *Registry
	name   string
	done   chan struct{}
	ch     chan []*registry.ServiceInfo
}

// newWatcher creates a new watcher.
func newWatcher(ctx context.Context, r *Registry, name string) (*watcher, error) {
	ctx, cancel := context.WithCancel(ctx)
	w := &watcher{
		ctx:    ctx,
		cancel: cancel,
		r:      r,
		name:   name,
		done:   make(chan struct{}),
		ch:     make(chan []*registry.ServiceInfo, 1),
	}

	// Get initial services
	services, err := r.GetService(ctx, name)
	if err != nil && err != registry.ErrNotFound {
		return nil, err
	}

	// Send initial services
	if err != registry.ErrNotFound {
		w.ch <- services
	}

	// Start watching for changes
	go w.watch()

	return w, nil
}

// watch watches for service changes.
func (w *watcher) watch() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			services, err := w.r.GetService(w.ctx, w.name)
			if err != nil {
				continue
			}
			select {
			case w.ch <- services:
			default:
			}
		}
	}
}

// Next returns the next service update.
func (w *watcher) Next() ([]*registry.ServiceInfo, error) {
	select {
	case <-w.ctx.Done():
		return nil, registry.ErrWatchCanceled
	case services := <-w.ch:
		return services, nil
	}
}

// Stop stops the watcher.
func (w *watcher) Stop() error {
	w.cancel()
	close(w.done)
	return nil
}
