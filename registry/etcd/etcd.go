package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"new-milli/registry"
)

var (
	_ registry.Registry = (*Registry)(nil)
	_ registry.Watcher  = (*watcher)(nil)
)

// Registry is etcd registry.
type Registry struct {
	client  *clientv3.Client
	options registry.Options
	sync.RWMutex
	leases map[string]clientv3.LeaseID
}

// New creates a new etcd registry.
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
		options.Addrs = []string{"127.0.0.1:2379"}
	}

	// Create etcd client
	config := clientv3.Config{
		Endpoints:   options.Addrs,
		DialTimeout: options.Timeout,
	}
	if options.Secure {
		// TODO: Add TLS configuration
	}
	if len(options.Username) > 0 && len(options.Password) > 0 {
		config.Username = options.Username
		config.Password = options.Password
	}

	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	return &Registry{
		client:  client,
		options: options,
		leases:  make(map[string]clientv3.LeaseID),
	}, nil
}

// Register registers a service.
func (r *Registry) Register(ctx context.Context, service *registry.ServiceInfo) error {
	if len(service.Nodes) == 0 {
		return fmt.Errorf("require at least one node")
	}

	r.Lock()
	defer r.Unlock()

	// Create lease
	leaseResp, err := r.client.Grant(ctx, 30)
	if err != nil {
		return err
	}

	// Register each node
	for _, node := range service.Nodes {
		// Create service data
		data := map[string]interface{}{
			"id":       node.ID,
			"name":     service.Name,
			"version":  service.Version,
			"address":  node.Address,
			"metadata": node.Metadata,
		}

		// Marshal the data
		dataByte, err := json.Marshal(data)
		if err != nil {
			return err
		}

		// Create the key
		key := path.Join("/services", service.Name, node.ID)

		// Put the key
		_, err = r.client.Put(ctx, key, string(dataByte), clientv3.WithLease(leaseResp.ID))
		if err != nil {
			return err
		}

		// Save the lease
		r.leases[node.ID] = leaseResp.ID
	}

	// Keep the lease alive
	go r.keepAlive(leaseResp.ID)

	return nil
}

// Deregister deregisters a service.
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInfo) error {
	r.Lock()
	defer r.Unlock()

	for _, node := range service.Nodes {
		// Create the key
		key := path.Join("/services", service.Name, node.ID)

		// Delete the key
		_, err := r.client.Delete(ctx, key)
		if err != nil {
			return err
		}

		// Revoke the lease
		leaseID, ok := r.leases[node.ID]
		if ok {
			r.client.Revoke(ctx, leaseID)
			delete(r.leases, node.ID)
		}
	}

	return nil
}

// GetService gets a service.
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInfo, error) {
	// Create the key
	key := path.Join("/services", serviceName)

	// Get the keys
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, registry.ErrNotFound
	}

	serviceMap := make(map[string]*registry.ServiceInfo)
	for _, kv := range resp.Kvs {
		// Unmarshal the data
		var data map[string]interface{}
		if err := json.Unmarshal(kv.Value, &data); err != nil {
			continue
		}

		// Get the version
		version, _ := data["version"].(string)
		if version == "" {
			version = "latest"
		}

		// Get or create the service
		s, ok := serviceMap[version]
		if !ok {
			s = &registry.ServiceInfo{
				Name:    serviceName,
				Version: version,
			}
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				s.Metadata = make(map[string]string)
				for k, v := range metadata {
					s.Metadata[k] = fmt.Sprintf("%v", v)
				}
			}
			serviceMap[version] = s
		}

		// Add the node
		node := &registry.Node{
			ID:      data["id"].(string),
			Address: data["address"].(string),
		}
		if metadata, ok := data["metadata"].(map[string]interface{}); ok {
			node.Metadata = make(map[string]string)
			for k, v := range metadata {
				node.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
		s.Nodes = append(s.Nodes, node)
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

// keepAlive keeps the lease alive.
func (r *Registry) keepAlive(leaseID clientv3.LeaseID) {
	kaCh, err := r.client.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return
	}
	for range kaCh {
		// Just drain the channel
	}
}

// watcher is a service watcher.
type watcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	r      *Registry
	name   string
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
		ch:     make(chan []*registry.ServiceInfo, 1),
	}

	// Create the key
	key := path.Join("/services", name)

	// Watch the key
	watchCh := r.client.Watch(ctx, key, clientv3.WithPrefix())

	// Start the watch
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-watchCh:
				services, err := r.GetService(ctx, name)
				if err != nil {
					continue
				}
				select {
				case w.ch <- services:
				default:
				}
			}
		}
	}()

	return w, nil
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
	return nil
}
