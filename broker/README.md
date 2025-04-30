# New Milli 消息队列系统

New Milli 消息队列系统是一个灵活、可扩展的消息队列集成解决方案，支持多种消息队列系统。

## 支持的消息队列

- **Kafka**: 高吞吐量的分布式发布订阅消息系统
- **RocketMQ**: 阿里巴巴开源的分布式消息中间件
- **RabbitMQ**: 实现了高级消息队列协议(AMQP)的开源消息代理软件

## 快速开始

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "new-milli/broker"
    "new-milli/broker/kafka" // 或 rocketmq, rabbitmq
)

func main() {
    // 创建 Kafka 代理
    b := kafka.New(
        broker.Addrs("localhost:9092"),
    )

    // 连接到代理
    if err := b.Connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer b.Disconnect()

    // 订阅主题
    _, err := b.Subscribe("my-topic", func(ctx context.Context, msg *broker.Message) error {
        fmt.Printf("Received message: %s\n", string(msg.Body))
        return nil
    })

    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // 发布消息
    msg := &broker.Message{
        Header: map[string]string{
            "id": "1",
            "source": "example",
        },
        Body: []byte("Hello, World!"),
    }

    if err := b.Publish(context.Background(), "my-topic", msg); err != nil {
        log.Fatalf("Failed to publish: %v", err)
    }
}
```

## 使用不同的消息队列

### Kafka

```go
import (
    "new-milli/broker"
    "new-milli/broker/kafka"
)

// 创建 Kafka 代理
b := kafka.New(
    broker.Addrs("localhost:9092"),
    broker.Auth("username", "password"), // 可选
    broker.Secure(true), // 可选，启用 TLS
)
```

### RocketMQ

```go
import (
    "new-milli/broker"
    "new-milli/broker/rocketmq"
)

// 创建 RocketMQ 代理
b := rocketmq.New(
    broker.Addrs("localhost:9876"),
    broker.Auth("username", "password"), // 可选
)
```

### RabbitMQ

```go
import (
    "new-milli/broker"
    "new-milli/broker/rabbitmq"
)

// 创建 RabbitMQ 代理
b := rabbitmq.New(
    broker.Addrs("amqp://guest:guest@localhost:5672/"),
    broker.Secure(true), // 可选，使用 amqps://
)
```

## 高级用法

### 自定义订阅选项

```go
// 订阅主题，使用自定义队列名称
sub, err := b.Subscribe("my-topic", handler,
    broker.Queue("my-queue"),
    broker.DisableAutoAck(), // 禁用自动确认
    broker.SubscribeContext(ctx), // 自定义上下文
)

// 取消订阅
sub.Unsubscribe()
```

### 自定义发布选项

```go
// 发布消息，使用自定义上下文
err := b.Publish(ctx, "my-topic", msg,
    broker.PublishContext(ctx),
)
```

### 使用编解码器

```go
// 创建 JSON 编解码器
codec := JsonCodec{}

// 创建代理，使用编解码器
b := kafka.New(
    broker.WithCodec(codec),
)

// 使用编解码器发布结构体
type MyMessage struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

msg := &MyMessage{
    Name: "John",
    Age:  30,
}

// 发布消息
err := b.Publish(ctx, "my-topic", &broker.Message{
    Body: codec.Marshal(msg),
})
```

## 实现自定义编解码器

```go
type JsonCodec struct{}

func (c JsonCodec) Marshal(v interface{}) ([]byte, error) {
    return json.Marshal(v)
}

func (c JsonCodec) Unmarshal(data []byte, v interface{}) error {
    return json.Unmarshal(data, v)
}

func (c JsonCodec) String() string {
    return "json"
}
```

## 消息队列配置

### Kafka 配置

Kafka 代理使用 `github.com/segmentio/kafka-go` 包。默认配置：

- 地址: `localhost:9092`
- 消费者组: 根据主题和队列名称自动生成
- 最小读取字节: 10KB
- 最大读取字节: 10MB

### RocketMQ 配置

RocketMQ 代理使用 `github.com/apache/rocketmq-client-go/v2` 包。默认配置：

- 地址: `localhost:9876`
- 生产者组: `new-milli-producer`
- 消费者组: 根据主题和队列名称自动生成
- 消费者模式: `Clustering`
- 重试次数: 2

### RabbitMQ 配置

RabbitMQ 代理使用 `github.com/rabbitmq/amqp091-go` 包。默认配置：

- 地址: `amqp://guest:guest@localhost:5672/`
- 交换机类型: `fanout`
- 队列持久化: 是
- 自动删除: 否
- 独占队列: 否
