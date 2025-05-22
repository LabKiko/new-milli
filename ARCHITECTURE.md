# New-Milli Framework Architecture

## 1. Introduction

New-Milli is a Go-based toolkit and framework designed for building modular and resilient applications, with a particular focus on microservices. It provides a set of well-defined components and conventions to simplify development, promote best practices, and accelerate the creation of robust, scalable services.

## 2. Core Philosophy

The design of New-Milli is guided by the following core principles:

*   **Modularity**: Components are designed to be independent and interchangeable, allowing developers to pick and choose parts that fit their needs.
*   **Extensibility**: The framework is built with extensibility in mind, enabling developers to easily add new functionalities or customize existing ones.
*   **Use of Interfaces**: Go interfaces are extensively used to define contracts between components, promoting loose coupling and testability.
*   **Integration of Common Microservice Patterns**: New-Milli aims to provide out-of-the-box support or easy integration for common microservice patterns like service discovery, distributed tracing, metrics, and centralized configuration.

## 3. Main Components

New-Milli consists of several core components that work together to provide a comprehensive application development experience.

### Application Lifecycle (`app.go`)

*   **Role & Features**: The `app.go` component manages the overall lifecycle of a New-Milli application. It handles initialization, startup, graceful shutdown, and coordination of other components. Key features include dependency injection, signal handling for termination, and managing start/stop sequences for services.
*   **Interactions**: It orchestrates other components like Configuration, Logging, Transport, Broker, and Registry during the application's startup and shutdown phases.

### Configuration (`config.go`)

*   **Role & Features**: The Configuration component is responsible for loading and providing access to application settings. It supports various sources like environment variables, configuration files (e.g., YAML, JSON, TOML), and remote configuration providers. It often includes features like type-safe configuration parsing and dynamic reloading.
*   **Interactions**: Almost all other components (Logging, Broker, Connector, Transport, Registry, and the application itself) consume configuration values provided by this component.

### Logging (`logger.go`)

*   **Role & Features**: The Logging component provides a standardized way to log application events and messages. It typically supports structured logging, different log levels (debug, info, warn, error), and various output formats (console, file, remote log aggregators).
*   **Interactions**: Used by virtually all other components and the application's business logic to record diagnostic information and operational events.

### Broker (`broker.go`)

*   **Role & Features**: The Broker component provides an abstraction for message queueing and pub/sub messaging. It allows services to communicate asynchronously. It defines interfaces for publishing messages and subscribing to topics, with implementations for various message brokers (e.g., Kafka, RabbitMQ, NATS).
*   **Interactions**: Business logic within the application uses the Broker to send and receive messages. The App Lifecycle component might manage the broker connections.

### Connector (`connector.go`)

*   **Role & Features**: The Connector component provides abstractions for interacting with external data stores or services, such as databases (SQL, NoSQL), caches, or other APIs. It aims to provide a consistent way to manage connections and perform operations.
*   **Interactions**: Primarily used by the application's business logic to persist and retrieve data or interact with other services. The App Lifecycle might manage the initialization of these connectors.

### Middleware (`middleware.go`)

*   **Role & Features**: Middleware components are pluggable handlers that process requests and responses in a chain. They are typically used for cross-cutting concerns like logging, metrics, tracing, authentication, authorization, and request/response manipulation.
*   **Interactions**: Middleware is primarily used by the Transport component. Requests pass through the middleware chain before reaching the main handler and responses pass through it in reverse.

### Transport (`transport.go`)

*   **Role & Features**: The Transport component is responsible for handling network communication. It abstracts the underlying protocols (e.g., HTTP, gRPC) for receiving requests and sending responses. It defines how services expose their endpoints.
*   **Interactions**: The App Lifecycle component starts and stops transport servers. Transport uses Middleware to process incoming requests and outgoing responses. It routes requests to the appropriate application handlers.

### Registry (`registry.go`)

*   **Role & Features**: The Registry component handles service discovery. Services register themselves with the registry upon startup and can discover other services through it. This is crucial for dynamic environments where service instances can come and go.
*   **Interactions**: The App Lifecycle component interacts with the Registry to register the service during startup and deregister it during shutdown. Client-side load balancing or service-to-service communication might use the Registry to find available service instances.

## 4. Typical Application Workflow

### Startup

1.  **Configuration Loading**: The `Config` component loads settings from files, environment variables, or remote sources.
2.  **Logger Initialization**: The `Logger` is initialized using the loaded configuration.
3.  **Connector Setup**: `Connector` instances (e.g., database connections) are initialized and configured.
4.  **Broker Connection**: Connections to message `Broker`s are established.
5.  **Transport Server Initialization**: The `Transport` server (e.g., HTTP server) is initialized with its routes and configured `Middleware` chain.
6.  **Service Registration**: The application registers itself with the `Registry` (if used).
7.  **App Run**: The `App` component starts all registered services (like the transport server) and blocks until a shutdown signal is received.

### Request Handling (e.g., HTTP)

1.  **Request Reception**: An incoming request (e.g., an HTTP request) hits the `Transport` layer.
2.  **Middleware Chain (Incoming)**: The request passes through the configured `Middleware` chain. Each middleware can inspect, modify, or even short-circuit the request (e.g., for authentication, logging, tracing, metrics collection).
3.  **Handler Logic**: The request reaches the designated application handler (business logic).
4.  **Data Access/Messaging**: The handler may use `Connector`s to interact with databases or other services, and/or use the `Broker` to publish messages or interact with event streams.
5.  **Response Generation**: The handler generates a response.
6.  **Middleware Chain (Outgoing)**: The response passes back through the `Middleware` chain in reverse order, allowing middleware to modify or act on the response (e.g., adding headers, logging the response).
7.  **Response Sending**: The `Transport` layer sends the final response back to the client.

## 5. How Components Fit Together (Interactions)

The components of New-Milli are designed to be cohesive yet loosely coupled:

*   The **`App`** component is central, orchestrating the lifecycle. It directly uses `Transport` to host servers and `Registry` for service registration/discovery.
*   The **`Transport`** layer (e.g., HTTP server) is responsible for exposing the application to the outside world. It heavily relies on **`Middleware`** to process requests and responses in a standardized way before they reach the core business logic.
*   Business logic (which developers build on top of New-Milli) primarily interacts with **`Connector`** components to access data (databases, caches) and **`Broker`** components for asynchronous communication (message queues).
*   Almost all components, including the application logic itself, depend on the **`Config`** component to obtain their settings and the **`Logger`** component for emitting logs.
*   The **`Registry`** allows services to find each other in a distributed environment, often used by clients or gateways interacting with New-Milli services.

This architecture promotes separation of concerns, making applications easier to develop, test, maintain, and scale.
