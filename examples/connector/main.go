package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"new-milli/connector"
	"new-milli/connector/clickhouse"
	"new-milli/connector/elasticsearch"
	"new-milli/connector/mongo"
	"new-milli/connector/mysql"
	"new-milli/connector/postgres"
	"new-milli/connector/redis"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [mysql|postgres|redis|mongo|elasticsearch|clickhouse]")
		os.Exit(1)
	}

	connType := os.Args[1]

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create connector
	var conn connector.Connector
	switch connType {
	case "mysql":
		conn = mysql.New(
			mysql.WithAddress("localhost:3306"),
			mysql.WithUsername("root"),
			mysql.WithPassword("password"),
			mysql.WithDatabase("test"),
		)
	case "postgres":
		conn = postgres.New(
			postgres.WithAddress("localhost:5432"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("password"),
			postgres.WithDatabase("test"),
		)
	case "redis":
		conn = redis.New(
			redis.WithAddress("localhost:6379"),
			redis.WithPassword(""),
			redis.WithDB(0),
		)
	case "mongo":
		conn = mongo.New(
			mongo.WithAddress("mongodb://localhost:27017"),
			mongo.WithDatabase("test"),
		)
	case "elasticsearch":
		conn = elasticsearch.New(
			elasticsearch.WithAddress("http://localhost:9200"),
		)
	case "clickhouse":
		conn = clickhouse.New(
			clickhouse.WithAddress("localhost:9000"),
			clickhouse.WithDatabase("default"),
		)
	default:
		fmt.Printf("Unsupported connector type: %s\n", connType)
		os.Exit(1)
	}

	// Connect to the database
	if err := conn.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to %s: %v", connType, err)
	}
	defer conn.Disconnect(context.Background())

	fmt.Printf("Connected to %s\n", connType)

	// Ping the database
	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping %s: %v", connType, err)
	}

	fmt.Printf("Pinged %s successfully\n", connType)

	// Perform database-specific operations
	switch connType {
	case "mysql":
		// Get the MySQL client
		db := conn.(*mysql.Connector).DB()

		// Create a table
		_, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}

		// Insert a row
		_, err = db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "John Doe")
		if err != nil {
			log.Fatalf("Failed to insert row: %v", err)
		}

		// Query rows
		rows, err := db.QueryContext(ctx, "SELECT id, name, created_at FROM users")
		if err != nil {
			log.Fatalf("Failed to query rows: %v", err)
		}
		defer rows.Close()

		// Print rows
		fmt.Println("MySQL users:")
		for rows.Next() {
			var id int
			var name string
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &createdAt); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}
			fmt.Printf("  %d: %s (created at %s)\n", id, name, createdAt)
		}

	case "postgres":
		// Get the PostgreSQL client
		db := conn.(*postgres.Connector).DB()

		// Create a table
		_, err := db.Exec("CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name VARCHAR(255), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}

		// Insert a row
		_, err = db.Exec("INSERT INTO users (name) VALUES ($1)", "Jane Doe")
		if err != nil {
			log.Fatalf("Failed to insert row: %v", err)
		}

		// Query rows
		rows, err := db.QueryContext(ctx, "SELECT id, name, created_at FROM users")
		if err != nil {
			log.Fatalf("Failed to query rows: %v", err)
		}
		defer rows.Close()

		// Print rows
		fmt.Println("PostgreSQL users:")
		for rows.Next() {
			var id int
			var name string
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &createdAt); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}
			fmt.Printf("  %d: %s (created at %s)\n", id, name, createdAt)
		}

	case "redis":
		// Get the Redis client
		client := conn.(*redis.Connector).Redis()

		// Set a key
		err := client.Set(ctx, "greeting", "Hello, Redis!", 0).Err()
		if err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}

		// Get a key
		val, err := client.Get(ctx, "greeting").Result()
		if err != nil {
			log.Fatalf("Failed to get key: %v", err)
		}

		fmt.Printf("Redis key 'greeting': %s\n", val)

	case "mongo":
		// Get the MongoDB client
		client := conn.(*mongo.Connector).Mongo()
		db := conn.(*mongo.Connector).Database()

		// Create a collection
		collection := db.Collection("users")

		// Insert a document
		_, err := collection.InsertOne(ctx, bson.M{
			"name":       "Bob Smith",
			"created_at": time.Now(),
		})
		if err != nil {
			log.Fatalf("Failed to insert document: %v", err)
		}

		// Find documents
		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			log.Fatalf("Failed to find documents: %v", err)
		}
		defer cursor.Close(ctx)

		// Print documents
		fmt.Println("MongoDB users:")
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				log.Fatalf("Failed to decode document: %v", err)
			}
			fmt.Printf("  %v\n", doc)
		}

		// Check for cursor errors
		if err := cursor.Err(); err != nil {
			log.Fatalf("Cursor error: %v", err)
		}

	case "elasticsearch":
		// Get the Elasticsearch client
		client := conn.(*elasticsearch.Connector).Elasticsearch()

		// Create an index
		res, err := client.Indices.Create("users")
		if err != nil {
			log.Fatalf("Failed to create index: %v", err)
		}
		defer res.Body.Close()

		// Index a document
		doc := map[string]interface{}{
			"name":       "Alice Johnson",
			"created_at": time.Now().Format(time.RFC3339),
		}
		res, err = client.Index("users", doc)
		if err != nil {
			log.Fatalf("Failed to index document: %v", err)
		}
		defer res.Body.Close()

		// Search for documents
		queryJSON := `{"query":{"match_all":{}}}`
		res, err = client.Search(
			client.Search.WithIndex("users"),
			client.Search.WithBody(strings.NewReader(queryJSON)),
		)
		if err != nil {
			log.Fatalf("Failed to search documents: %v", err)
		}
		defer res.Body.Close()

		fmt.Printf("Elasticsearch search response: %s\n", res.String())

	case "clickhouse":
		// Get the ClickHouse client
		conn := conn.(*clickhouse.Connector).Conn()

		// Create a table
		err := conn.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS users (
				id       UInt64,
				name     String,
				created_at DateTime
			) ENGINE = MergeTree()
			ORDER BY id
		`)
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}

		// Insert a row
		err = conn.Exec(ctx, "INSERT INTO users (id, name, created_at) VALUES (?, ?, ?)",
			1, "Charlie Brown", time.Now())
		if err != nil {
			log.Fatalf("Failed to insert row: %v", err)
		}

		// Query rows
		rows, err := conn.Query(ctx, "SELECT id, name, created_at FROM users")
		if err != nil {
			log.Fatalf("Failed to query rows: %v", err)
		}
		defer rows.Close()

		// Print rows
		fmt.Println("ClickHouse users:")
		for rows.Next() {
			var id uint64
			var name string
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &createdAt); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}
			fmt.Printf("  %d: %s (created at %s)\n", id, name, createdAt)
		}
	}

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to exit")
	<-sigChan
	fmt.Println("Exiting...")
}
