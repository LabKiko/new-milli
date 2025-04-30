# New Milli 数据库连接器

New Milli 数据库连接器是一个灵活、可扩展的数据库连接解决方案，支持多种数据库系统。

## 支持的数据库

- **MySQL**: 流行的关系型数据库
- **PostgreSQL**: 功能强大的开源关系型数据库
- **Redis**: 高性能的键值存储数据库
- **MongoDB**: 文档型NoSQL数据库
- **Elasticsearch**: 分布式搜索和分析引擎
- **ClickHouse**: 列式存储分析型数据库

## 快速开始

### 基本用法

```go
package main

import (
    "context"
    "log"
    "time"
    
    "new-milli/connector/mysql"
)

func main() {
    // 创建 MySQL 连接器
    conn := mysql.New(
        mysql.WithAddress("localhost:3306"),
        mysql.WithUsername("root"),
        mysql.WithPassword("password"),
        mysql.WithDatabase("test"),
    )
    
    // 连接到数据库
    ctx := context.Background()
    if err := conn.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Disconnect(ctx)
    
    // 获取底层客户端
    db := conn.(*mysql.Connector).DB()
    
    // 使用客户端
    rows, err := db.Query("SELECT * FROM users")
    if err != nil {
        log.Fatalf("Failed to query: %v", err)
    }
    defer rows.Close()
    
    // 处理结果
    for rows.Next() {
        // ...
    }
}
```

## 连接器接口

所有连接器都实现了以下接口：

```go
type Connector interface {
    // Connect connects to the database.
    Connect(ctx context.Context) error
    // Disconnect disconnects from the database.
    Disconnect(ctx context.Context) error
    // Ping checks if the database is reachable.
    Ping(ctx context.Context) error
    // IsConnected returns true if the connector is connected.
    IsConnected() bool
    // Name returns the name of the connector.
    Name() string
    // Client returns the underlying client.
    Client() interface{}
}
```

## 连接器注册表

可以使用连接器注册表管理多个连接器：

```go
// 注册连接器
connector.Register("mysql1", mysqlConn)
connector.Register("redis1", redisConn)

// 获取连接器
mysqlConn, ok := connector.Get("mysql1")
if ok {
    // 使用连接器
}

// 列出所有连接器
connectors := connector.List()
for name, conn := range connectors {
    fmt.Printf("Connector: %s, Connected: %v\n", name, conn.IsConnected())
}

// 关闭所有连接器
if err := connector.Close(ctx); err != nil {
    log.Fatalf("Failed to close connectors: %v", err)
}
```

## 连接器详解

### MySQL 连接器

```go
// 创建 MySQL 连接器
conn := mysql.New(
    mysql.WithAddress("localhost:3306"),
    mysql.WithUsername("root"),
    mysql.WithPassword("password"),
    mysql.WithDatabase("test"),
    mysql.WithConnectTimeout(time.Second * 10),
    mysql.WithMaxIdleConns(10),
    mysql.WithMaxOpenConns(100),
    mysql.WithMaxConnLifetime(time.Hour),
    mysql.WithParseTime(true),
    mysql.WithTLS(false),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层客户端
db := conn.(*mysql.Connector).DB()

// 执行查询
rows, err := db.Query("SELECT * FROM users")
```

### PostgreSQL 连接器 (使用 GORM)

```go
// 创建 PostgreSQL 连接器
conn := postgres.New(
    postgres.WithAddress("localhost:5432"),
    postgres.WithUsername("postgres"),
    postgres.WithPassword("password"),
    postgres.WithDatabase("test"),
    postgres.WithSSLMode("disable"),
    postgres.WithTimezone("UTC"),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层 GORM 客户端
db := conn.(*postgres.Connector).DB()

// 定义模型
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string
    CreatedAt time.Time
}

// 自动迁移
db.AutoMigrate(&User{})

// 创建记录
db.Create(&User{Name: "John Doe"})

// 查询记录
var users []User
db.Find(&users)
```

### Redis 连接器

```go
// 创建 Redis 连接器
conn := redis.New(
    redis.WithAddress("localhost:6379"),
    redis.WithPassword(""),
    redis.WithDB(0),
    redis.WithMode("single"), // 或 "sentinel", "cluster"
    redis.WithPoolSize(10),
    redis.WithMaxRetries(3),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层客户端
client := conn.(*redis.Connector).Redis()

// 设置键值
err := client.Set(ctx, "key", "value", 0).Err()

// 获取键值
val, err := client.Get(ctx, "key").Result()
```

### MongoDB 连接器

```go
// 创建 MongoDB 连接器
conn := mongo.New(
    mongo.WithAddress("mongodb://localhost:27017"),
    mongo.WithDatabase("test"),
    mongo.WithUsername(""),
    mongo.WithPassword(""),
    mongo.WithReplicaSet(""),
    mongo.WithRetryWrites(true),
    mongo.WithRetryReads(true),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层客户端
client := conn.(*mongo.Connector).Mongo()
db := conn.(*mongo.Connector).Database()
collection := conn.(*mongo.Connector).Collection("users")

// 插入文档
_, err := collection.InsertOne(ctx, bson.M{"name": "John Doe"})

// 查询文档
cursor, err := collection.Find(ctx, bson.M{})
defer cursor.Close(ctx)

// 处理结果
var results []bson.M
if err := cursor.All(ctx, &results); err != nil {
    log.Fatalf("Failed to get results: %v", err)
}
```

### Elasticsearch 连接器

```go
// 创建 Elasticsearch 连接器
conn := elasticsearch.New(
    elasticsearch.WithAddress("http://localhost:9200"),
    elasticsearch.WithUsername(""),
    elasticsearch.WithPassword(""),
    elasticsearch.WithCloudID(""),
    elasticsearch.WithAPIKey(""),
    elasticsearch.WithMaxRetries(3),
    elasticsearch.WithCompressRequestBody(true),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层客户端
client := conn.(*elasticsearch.Connector).Elasticsearch()

// 创建索引
res, err := client.Indices.Create("my-index")

// 索引文档
doc := map[string]interface{}{
    "title":   "Test Document",
    "content": "This is a test document",
}
res, err = client.Index("my-index", strings.NewReader(fmt.Sprintf("%v", doc)))

// 搜索文档
query := map[string]interface{}{
    "query": map[string]interface{}{
        "match": map[string]interface{}{
            "title": "test",
        },
    },
}
res, err = client.Search(
    client.Search.WithIndex("my-index"),
    client.Search.WithBody(strings.NewReader(fmt.Sprintf("%v", query))),
)
```

### ClickHouse 连接器

```go
// 创建 ClickHouse 连接器
conn := clickhouse.New(
    clickhouse.WithAddress("localhost:9000"),
    clickhouse.WithDatabase("default"),
    clickhouse.WithUsername("default"),
    clickhouse.WithPassword(""),
    clickhouse.WithCompression("lz4"),
    clickhouse.WithMaxOpenConns(10),
    clickhouse.WithMaxExecutionTime(time.Minute),
)

// 连接到数据库
if err := conn.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// 获取底层客户端
client := conn.(*clickhouse.Connector).Conn()
db := conn.(*clickhouse.Connector).DB()

// 创建表
err := client.Exec(ctx, `
    CREATE TABLE IF NOT EXISTS events (
        id       UInt64,
        name     String,
        timestamp DateTime
    ) ENGINE = MergeTree()
    ORDER BY id
`)

// 插入数据
err = client.Exec(ctx, "INSERT INTO events (id, name, timestamp) VALUES (?, ?, ?)",
    1, "click", time.Now())

// 查询数据
rows, err := client.Query(ctx, "SELECT id, name, timestamp FROM events WHERE id = ?", 1)
defer rows.Close()

// 处理结果
for rows.Next() {
    var id uint64
    var name string
    var timestamp time.Time
    if err := rows.Scan(&id, &name, &timestamp); err != nil {
        log.Fatalf("Failed to scan row: %v", err)
    }
    fmt.Printf("Event: %d, %s, %s\n", id, name, timestamp)
}
```

## 连接池配置

所有连接器都支持连接池配置：

```go
// 设置连接池参数
conn := mysql.New(
    mysql.WithMaxIdleConns(10),    // 最大空闲连接数
    mysql.WithMaxOpenConns(100),   // 最大打开连接数
    mysql.WithMaxConnLifetime(time.Hour), // 连接最大生命周期
    mysql.WithMaxIdleTime(time.Minute * 30), // 连接最大空闲时间
)
```

## TLS 配置

所有连接器都支持 TLS 配置：

```go
// 设置 TLS 参数
conn := mysql.New(
    mysql.WithTLS(true),                  // 启用 TLS
    mysql.WithTLSSkipVerify(false),       // 是否跳过验证
    mysql.WithTLSCertPath("/path/to/cert.pem"), // 客户端证书路径
    mysql.WithTLSKeyPath("/path/to/key.pem"),   // 客户端密钥路径
    mysql.WithTLSCAPath("/path/to/ca.pem"),     // CA 证书路径
)
```

## 超时配置

所有连接器都支持超时配置：

```go
// 设置超时参数
conn := mysql.New(
    mysql.WithConnectTimeout(time.Second * 10), // 连接超时
    mysql.WithReadTimeout(time.Second * 30),    // 读取超时
    mysql.WithWriteTimeout(time.Second * 30),   // 写入超时
)
```

## 自定义连接器

可以通过实现 `Connector` 接口来创建自定义连接器：

```go
type MyConnector struct {
    // ...
}

func (c *MyConnector) Connect(ctx context.Context) error {
    // ...
}

func (c *MyConnector) Disconnect(ctx context.Context) error {
    // ...
}

func (c *MyConnector) Ping(ctx context.Context) error {
    // ...
}

func (c *MyConnector) IsConnected() bool {
    // ...
}

func (c *MyConnector) Name() string {
    // ...
}

func (c *MyConnector) Client() interface{} {
    // ...
}
```
