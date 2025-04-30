package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/segmentio/kafka-go"
	"new-milli/broker"
)

var (
	_ broker.Broker = (*Broker)(nil)
)

// Broker is a Kafka broker.
type Broker struct {
	sync.RWMutex
	addrs     []string
	connected bool
	options   broker.Options
	writers   map[string]*kafka.Writer
	readers   map[string]*kafka.Reader
}

// New creates a new Kafka broker.
func New(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Addrs:   []string{"localhost:9092"},
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	return &Broker{
		addrs:   options.Addrs,
		options: options,
		writers: make(map[string]*kafka.Writer),
		readers: make(map[string]*kafka.Reader),
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

	// Close all writers
	for _, writer := range b.writers {
		writer.Close()
	}

	// Close all readers
	for _, reader := range b.readers {
		reader.Close()
	}

	b.connected = false
	return nil
}

// Publish publishes a message to a topic.
func (b *Broker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	// Get or create the writer
	writer, err := b.getWriter(topic)
	if err != nil {
		return err
	}

	// Create the message
	kmsg := kafka.Message{
		Key:   []byte(topic),
		Value: msg.Body,
	}

	// Add headers
	for k, v := range msg.Header {
		kmsg.Headers = append(kmsg.Headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Write the message
	return writer.WriteMessages(options.Context, kmsg)
}

// Subscribe subscribes to a topic.
func (b *Broker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   "default",
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	// Get or create the reader
	reader, err := b.getReader(topic, options.Queue)
	if err != nil {
		return nil, err
	}

	// Create the subscriber
	sub := &subscriber{
		topic:   topic,
		handler: handler,
		reader:  reader,
		options: options,
		done:    make(chan struct{}),
	}

	// Start the subscriber
	go sub.run()

	return sub, nil
}

// String returns the name of the broker.
func (b *Broker) String() string {
	return "kafka"
}

// getWriter gets or creates a writer for a topic.
func (b *Broker) getWriter(topic string) (*kafka.Writer, error) {
	b.Lock()
	defer b.Unlock()

	// Check if the writer exists
	if writer, ok := b.writers[topic]; ok {
		return writer, nil
	}

	// Create the writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(b.addrs...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// Save the writer
	b.writers[topic] = writer

	return writer, nil
}

// getReader gets or creates a reader for a topic.
func (b *Broker) getReader(topic, group string) (*kafka.Reader, error) {
	b.Lock()
	defer b.Unlock()

	// Create the key
	key := fmt.Sprintf("%s-%s", topic, group)

	// Check if the reader exists
	if reader, ok := b.readers[key]; ok {
		return reader, nil
	}

	// Create the reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  b.addrs,
		Topic:    topic,
		GroupID:  group,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	// Save the reader
	b.readers[key] = reader

	return reader, nil
}

// subscriber is a Kafka subscriber.
type subscriber struct {
	topic   string
	handler broker.Handler
	reader  *kafka.Reader
	options broker.SubscribeOptions
	done    chan struct{}
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes from the topic.
func (s *subscriber) Unsubscribe() error {
	close(s.done)
	return s.reader.Close()
}

// run runs the subscriber.
func (s *subscriber) run() {
	for {
		select {
		case <-s.done:
			return
		default:
			// Read the message
			kmsg, err := s.reader.ReadMessage(s.options.Context)
			if err != nil {
				continue
			}

			// Create the message
			msg := &broker.Message{
				Header: make(map[string]string),
				Body:   kmsg.Value,
			}

			// Add headers
			for _, header := range kmsg.Headers {
				msg.Header[header.Key] = string(header.Value)
			}

			// Handle the message
			err = s.handler(s.options.Context, msg)
			if err != nil {
				// TODO: Handle error
				continue
			}

			// Auto ack
			if s.options.AutoAck {
				// TODO: Implement ack
			}
		}
	}
}
