package newMilli

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"new-milli/transport"
)

// AppInfo is application context value.
type AppInfo interface {
	ID() string
	Name() string
	Version() string
}

// App is an application lifecycle manager.
type App struct {
	opts   options
	ctx    context.Context
	cancel func()
}

// New creates a new application.
func New(opts ...Option) (*App, error) {
	o := options{
		ctx:              context.Background(),
		sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT},
		registrarTimeout: 10 * time.Second,
		stopTimeout:      10 * time.Second,
		metadata:         make(map[string]string),
	}

	for _, opt := range opts {
		opt(&o)
	}

	for _, srv := range o.servers {
		srv := srv
		if err := srv.Init(
			transport.ID(o.id),
			transport.Name(o.name),
			transport.Version(o.version),
		); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(o.ctx)
	return &App{
		ctx:    ctx,
		cancel: cancel,
		opts:   o,
	}, nil
}

// ID returns app instance id.
func (a *App) ID() string { return a.opts.id }

// Name returns service name.
func (a *App) Name() string { return a.opts.name }

// Version returns app version.
func (a *App) Version() string { return a.opts.version }

// Run executes all OnStart hooks registered with the application's Lifecycle.
func (a *App) Run() error {
	ctx := NewContext(a.ctx, a)
	eg, ctx := errgroup.WithContext(ctx)
	wg := sync.WaitGroup{}

	// Before start
	for _, fn := range a.opts.beforeStart {
		if err := fn(ctx); err != nil {
			return err
		}
	}

	for _, srv := range a.opts.servers {
		srv := srv
		eg.Go(func() error {
			<-ctx.Done()
			stopCtx, cancel := context.WithTimeout(NewContext(context.Background(), a), a.opts.stopTimeout)
			defer cancel()
			return srv.Stop(stopCtx)
		})
		wg.Add(1)
		eg.Go(func() error {
			wg.Done()
			return srv.Start(ctx)
		})
	}
	wg.Wait()

	// After start
	for _, fn := range a.opts.afterStart {
		if err := fn(ctx); err != nil {
			return err
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, a.opts.sigs...)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-c:
			return a.Stop()
		}
	})

	if err := eg.Wait(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

// Stop gracefully stops the application.
func (a *App) Stop() error {
	ctx := NewContext(a.ctx, a)
	for _, fn := range a.opts.beforeStop {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	if a.cancel != nil {
		a.cancel()
	}
	for _, fn := range a.opts.afterStop {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

type appKey struct{}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, a AppInfo) context.Context {
	return context.WithValue(ctx, appKey{}, a)
}

// FromContext returns the AppInfo value stored in ctx, if any.
func FromContext(ctx context.Context) (a AppInfo, ok bool) {
	a, ok = ctx.Value(appKey{}).(AppInfo)
	return
}
