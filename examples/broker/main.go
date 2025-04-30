package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"new-milli/broker"
	"new-milli/broker/kafka"
	"new-milli/broker/rabbitmq"
	"new-milli/broker/rocketmq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [kafka|rocketmq|rabbitmq]")
		os.Exit(1)
	}

	brokerType := os.Args[1]

	var b broker.Broker
	switch brokerType {
	case "kafka":
		b = kafka.New(
			broker.Addrs("localhost:9092"),
		)
	case "rocketmq":
		b = rocketmq.New(
			broker.Addrs("localhost:9876"),
		)
	case "rabbitmq":
		b = rabbitmq.New(
			broker.Addrs("amqp://guest:guest@localhost:5672/"),
		)
	default:
		fmt.Printf("Unsupported broker type: %s\n", brokerType)
		os.Exit(1)
	}

	// Connect to the broker
	if err := b.Connect(); err != nil {
		log.Fatalf("Failed to connect to %s: %v", brokerType, err)
	}
	defer b.Disconnect()

	fmt.Printf("Connected to %s broker\n", brokerType)

	// Create a topic
	topic := "new-milli-example"

	// Subscribe to the topic
	_, err := b.Subscribe(topic, func(ctx context.Context, msg *broker.Message) error {
		fmt.Printf("Received message: %s\n", string(msg.Body))
		for k, v := range msg.Header {
			fmt.Printf("Header: %s=%s\n", k, v)
		}
		return nil
	}, broker.Queue("example-queue"))

	if err != nil {
		log.Fatalf("Failed to subscribe to topic %s: %v", topic, err)
	}

	fmt.Printf("Subscribed to topic: %s\n", topic)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish a message
	msg := &broker.Message{
		Header: map[string]string{
			"id":        "1",
			"timestamp": time.Now().Format(time.RFC3339),
			"source":    "new-milli-example",
		},
		Body: []byte("Hello, World!"),
	}

	if err := b.Publish(ctx, topic, msg); err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Printf("Published message to topic: %s\n", topic)

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to exit")
	<-sigChan
	fmt.Println("Exiting...")
}
