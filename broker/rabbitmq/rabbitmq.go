package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"new-milli/broker"
)

var (
	_ broker.Broker = (*Broker)(nil)
)

// Broker is a RabbitMQ broker.
type Broker struct {
	sync.RWMutex
	addrs      []string
	connected  bool
	options    broker.Options
	connection *amqp.Connection
	channel    *amqp.Channel
	exchanges  map[string]bool
	subscribers map[string]*subscriber
}

// New creates a new RabbitMQ broker.
func New(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Addrs:   []string{"amqp://guest:guest@localhost:5672/"},
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	return &Broker{
		addrs:       options.Addrs,
		options:     options,
		exchanges:   make(map[string]bool),
		subscribers: make(map[string]*subscriber),
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

	// Connect to RabbitMQ
	conn, err := amqp.Dial(b.addrs[0])
	if err != nil {
		return err
	}

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	b.connection = conn
	b.channel = ch
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

	// Close all subscribers
	for _, sub := range b.subscribers {
		sub.Unsubscribe()
	}

	// Close the channel
	if b.channel != nil {
		b.channel.Close()
	}

	// Close the connection
	if b.connection != nil {
		b.connection.Close()
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
	ch := b.channel
	b.RUnlock()

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	// Ensure the exchange exists
	if err := b.ensureExchange(topic); err != nil {
		return err
	}

	// Create the message
	headers := amqp.Table{}
	for k, v := range msg.Header {
		headers[k] = v
	}

	// Publish the message
	return ch.PublishWithContext(
		options.Context,
		topic, // exchange
		"",    // routing key (empty for fanout)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        msg.Body,
			Headers:     headers,
		},
	)
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

	// Ensure the exchange exists
	if err := b.ensureExchange(topic); err != nil {
		return nil, err
	}

	// Create a queue
	queueName := fmt.Sprintf("%s-%s", topic, options.Queue)
	q, err := b.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, err
	}

	// Bind the queue to the exchange
	err = b.channel.QueueBind(
		q.Name, // queue name
		"",     // routing key (empty for fanout)
		topic,  // exchange
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		return nil, err
	}

	// Create a consumer
	ch, err := b.connection.Channel()
	if err != nil {
		return nil, err
	}

	// Start consuming
	deliveries, err := ch.Consume(
		q.Name,                   // queue
		fmt.Sprintf("%s-%d", q.Name, time.Now().UnixNano()), // consumer
		options.AutoAck,          // auto-ack
		false,                    // exclusive
		false,                    // no-local
		false,                    // no-wait
		nil,                      // args
	)
	if err != nil {
		ch.Close()
		return nil, err
	}

	// Create the subscriber
	sub := &subscriber{
		topic:      topic,
		queue:      options.Queue,
		handler:    handler,
		channel:    ch,
		options:    options,
		deliveries: deliveries,
		done:       make(chan struct{}),
	}

	// Start the subscriber
	go sub.run()

	// Save the subscriber
	b.subscribers[sub.id()] = sub

	return sub, nil
}

// String returns the name of the broker.
func (b *Broker) String() string {
	return "rabbitmq"
}

// ensureExchange ensures that an exchange exists.
func (b *Broker) ensureExchange(name string) error {
	if _, ok := b.exchanges[name]; ok {
		return nil
	}

	err := b.channel.ExchangeDeclare(
		name,     // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return err
	}

	b.exchanges[name] = true
	return nil
}

// subscriber is a RabbitMQ subscriber.
type subscriber struct {
	topic      string
	queue      string
	handler    broker.Handler
	channel    *amqp.Channel
	options    broker.SubscribeOptions
	deliveries <-chan amqp.Delivery
	done       chan struct{}
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes from the topic.
func (s *subscriber) Unsubscribe() error {
	close(s.done)
	return s.channel.Close()
}

// id returns a unique id for the subscriber.
func (s *subscriber) id() string {
	return fmt.Sprintf("%s-%s", s.topic, s.queue)
}

// run runs the subscriber.
func (s *subscriber) run() {
	for {
		select {
		case <-s.done:
			return
		case delivery, ok := <-s.deliveries:
			if !ok {
				return
			}

			// Create the message
			msg := &broker.Message{
				Header: make(map[string]string),
				Body:   delivery.Body,
			}

			// Add headers
			for k, v := range delivery.Headers {
				if value, ok := v.(string); ok {
					msg.Header[k] = value
				}
			}

			// Handle the message
			err := s.handler(s.options.Context, msg)
			if err != nil {
				// Nack the message if auto-ack is disabled
				if !s.options.AutoAck {
					delivery.Nack(false, true)
				}
				continue
			}

			// Ack the message if auto-ack is disabled
			if !s.options.AutoAck {
				delivery.Ack(false)
			}
		}
	}
}
