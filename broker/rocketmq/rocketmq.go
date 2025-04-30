package rocketmq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"new-milli/broker"
)

var (
	_ broker.Broker = (*Broker)(nil)
)

// Broker is a RocketMQ broker.
type Broker struct {
	sync.RWMutex
	addrs     []string
	connected bool
	options   broker.Options
	producer  rocketmq.Producer
	consumers map[string]rocketmq.PushConsumer
}

// New creates a new RocketMQ broker.
func New(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Addrs:   []string{"localhost:9876"},
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	return &Broker{
		addrs:     options.Addrs,
		options:   options,
		consumers: make(map[string]rocketmq.PushConsumer),
	}
}

// Init initializes the broker.
func (b *Broker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}
	return nil
}

// Options returns the broker options.
func (b *Broker) Options() broker.Options {
	return b.options
}

// Address returns the broker address.
func (b *Broker) Address() string {
	return strings.Join(b.addrs, ",")
}

// Connect connects to the broker.
func (b *Broker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.connected {
		return nil
	}

	// Create producer
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(b.addrs),
		producer.WithRetry(2),
		producer.WithGroupName("new-milli-producer"),
	)
	if err != nil {
		return err
	}

	// Start the producer
	if err := p.Start(); err != nil {
		return err
	}

	b.producer = p
	b.connected = true
	return nil
}

// Disconnect disconnects from the broker.
func (b *Broker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if !b.connected {
		return nil
	}

	// Shutdown the producer
	if b.producer != nil {
		if err := b.producer.Shutdown(); err != nil {
			return err
		}
	}

	// Shutdown all consumers
	for _, c := range b.consumers {
		if err := c.Shutdown(); err != nil {
			return err
		}
	}

	b.connected = false
	return nil
}

// Publish publishes a message to a topic.
func (b *Broker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b.RLock()
	if !b.connected {
		b.RUnlock()
		return errors.New("not connected")
	}
	p := b.producer
	b.RUnlock()

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	// Create the message
	rmsg := primitive.NewMessage(topic, msg.Body)

	// Add properties (headers)
	for k, v := range msg.Header {
		rmsg.WithProperty(k, v)
	}

	// Send the message
	_, err := p.SendSync(options.Context, rmsg)
	return err
}

// Subscribe subscribes to a topic.
func (b *Broker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.Lock()
	defer b.Unlock()

	if !b.connected {
		return nil, errors.New("not connected")
	}

	options := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   "default",
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	// Create a unique consumer group name
	groupName := fmt.Sprintf("new-milli-consumer-%s-%s", topic, options.Queue)

	// Create consumer
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(b.addrs),
		consumer.WithGroupName(groupName),
		consumer.WithConsumerModel(consumer.Clustering),
	)
	if err != nil {
		return nil, err
	}

	// Create the subscriber
	sub := &subscriber{
		topic:    topic,
		queue:    options.Queue,
		handler:  handler,
		consumer: c,
		options:  options,
		done:     make(chan struct{}),
	}

	// Register the message handler
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "*",
	}

	err = c.Subscribe(topic, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			// Create the message
			m := &broker.Message{
				Header: make(map[string]string),
				Body:   msg.Body,
			}

			// Add properties (headers)
			for k, v := range msg.GetProperties() {
				m.Header[k] = v
			}

			// Handle the message
			err := handler(ctx, m)
			if err != nil {
				return consumer.ConsumeRetryLater, err
			}
		}
		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		return nil, err
	}

	// Start the consumer
	if err := c.Start(); err != nil {
		return nil, err
	}

	// Save the consumer
	b.consumers[sub.id()] = c

	return sub, nil
}

// String returns the name of the broker.
func (b *Broker) String() string {
	return "rocketmq"
}

// subscriber is a RocketMQ subscriber.
type subscriber struct {
	topic    string
	queue    string
	handler  broker.Handler
	consumer rocketmq.PushConsumer
	options  broker.SubscribeOptions
	done     chan struct{}
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes from the topic.
func (s *subscriber) Unsubscribe() error {
	close(s.done)
	return s.consumer.Shutdown()
}

// id returns a unique id for the subscriber.
func (s *subscriber) id() string {
	return fmt.Sprintf("%s-%s", s.topic, s.queue)
}
