package broker

import (
	"context"
)

// Broker is an interface used for asynchronous messaging.
type Broker interface {
	// Init initializes the broker.
	Init(...Option) error
	// Options returns the broker options.
	Options() Options
	// Address returns the broker address.
	Address() string
	// Connect connects to the broker.
	Connect() error
	// Disconnect disconnects from the broker.
	Disconnect() error
	// Publish publishes a message to a topic.
	Publish(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error
	// Subscribe subscribes to a topic.
	Subscribe(topic string, handler Handler, opts ...SubscribeOption) (Subscriber, error)
	// String returns the name of the broker.
	String() string
}

// Handler is used to process messages via a subscription.
type Handler func(context.Context, *Message) error

// Message is a broker message.
type Message struct {
	Header map[string]string
	Body   []byte
}

// Subscriber is a convenience return type for the Subscribe method.
type Subscriber interface {
	// Topic returns the topic of the subscriber.
	Topic() string
	// Unsubscribe unsubscribes from the topic.
	Unsubscribe() error
}

// Option is broker option.
type Option func(*Options)

// Options is broker options.
type Options struct {
	Addrs     []string
	Secure    bool
	Username  string
	Password  string
	Codec     Codec
	Context   context.Context
	TLSConfig interface{}
}

// Codec is used to encode/decode messages.
type Codec interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	String() string
}

// PublishOption is publish option.
type PublishOption func(*PublishOptions)

// PublishOptions is publish options.
type PublishOptions struct {
	Context context.Context
}

// SubscribeOption is subscribe option.
type SubscribeOption func(*SubscribeOptions)

// SubscribeOptions is subscribe options.
type SubscribeOptions struct {
	// AutoAck defaults to true. When a handler returns
	// with a nil error the message is acked.
	AutoAck bool
	// Queue is the queue to subscribe to.
	Queue string
	// Context is the context for the subscription.
	Context context.Context
}

// Addrs sets the broker addresses.
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Secure sets the broker secure option.
func Secure(secure bool) Option {
	return func(o *Options) {
		o.Secure = secure
	}
}

// Auth sets the broker authentication.
func Auth(username, password string) Option {
	return func(o *Options) {
		o.Username = username
		o.Password = password
	}
}

// WithCodec sets the broker codec.
func WithCodec(c Codec) Option {
	return func(o *Options) {
		o.Codec = c
	}
}

// Context sets the broker context.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Queue sets the subscription queue.
func Queue(queue string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = queue
	}
}

// DisableAutoAck disables auto ack.
func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = false
	}
}

// SubscribeContext sets the subscription context.
func SubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Context = ctx
	}
}

// PublishContext sets the publish context.
func PublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}
