# Env - Environment Variable Management

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/env.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/env)

Package `env` provides utilities for loading and parsing environment variables with type safety and validation. It supports common Go types, graceful error handling, and helpful error messages for configuration issues.

## Features

- **Type-Safe Parsing**: Load environment variables as specific Go types with compile-time safety
- **Graceful Error Handling**: Choose between panic-on-error or return-error patterns
- **Default Values**: Provide fallback values for optional configuration
- **Slice Support**: Parse comma-separated or custom-delimited lists
- **Duration Support**: Built-in support for time.Duration parsing
- **Validation Helpers**: Check if variables exist or are set
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Installation

```bash
go get github.com/alextanhongpin/core/types/env
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/alextanhongpin/core/types/env"
)

func main() {
    // Load required config (panics if missing)
    port := env.Load[int]("PORT")
    
    // Load optional config with defaults
    host := env.GetWithDefault[string]("HOST", "localhost")
    timeout := env.GetDurationWithDefault("TIMEOUT", 30*time.Second)
    
    // Load arrays
    allowedHosts := env.GetSliceWithDefault[string]("ALLOWED_HOSTS", ",", []string{"localhost"})
    
    fmt.Printf("Server: %s:%d, Timeout: %v, Hosts: %v\n", 
        host, port, timeout, allowedHosts)
}
```

## API Reference

### Loading Functions

#### Panic-on-Error (Fail Fast)

These functions panic if the environment variable is missing or invalid. Use for critical configuration that should cause the application to fail at startup.

- **`Load[T](name string) T`** - Load required variable
- **`LoadDuration(name string) time.Duration`** - Load required duration
- **`LoadSlice[T](name, sep string) []T`** - Load required slice
- **`MustExist(names ...string)`** - Ensure variables exist

```go
// Application will panic if PORT is not set or invalid
port := env.Load[int]("PORT")
timeout := env.LoadDuration("REQUEST_TIMEOUT")
```

#### Error-Returning (Graceful)

These functions return errors for missing or invalid variables. Use when you want to handle configuration errors gracefully.

- **`Get[T](name string) (T, error)`** - Load with error handling
- **`GetDuration(name string) (time.Duration, error)`** - Load duration with error handling
- **`GetSlice[T](name, sep string) ([]T, error)`** - Load slice with error handling

```go
port, err := env.Get[int]("PORT")
if err != nil {
    log.Printf("Failed to get port: %v", err)
    port = 8080 // fallback
}
```

#### Default Values

These functions provide fallback values if the environment variable is missing or invalid.

- **`GetWithDefault[T](name string, defaultValue T) T`** - Load with default
- **`GetDurationWithDefault(name string, defaultValue time.Duration) time.Duration`** - Load duration with default
- **`GetSliceWithDefault[T](name, sep string, defaultValue []T) []T`** - Load slice with default

```go
host := env.GetWithDefault[string]("HOST", "0.0.0.0")
maxConns := env.GetWithDefault[int]("MAX_CONNECTIONS", 100)
```

### Utility Functions

- **`Exists(name string) bool`** - Check if variable exists (even if empty)
- **`IsSet(name string) bool`** - Check if variable exists and is non-empty
- **`Parse[T](str string) (T, error)`** - Parse string to specific type

## Supported Types

The package supports all basic Go types through the `Parseable` interface:

- **Strings**: `string`
- **Integers**: `int`, `int8`, `int16`, `int32`, `int64`
- **Unsigned Integers**: `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr`
- **Floats**: `float32`, `float64`
- **Complex**: `complex64`, `complex128`
- **Boolean**: `bool` (accepts "true", "false", "1", "0", "t", "f", "T", "F")
- **Duration**: `time.Duration` (via dedicated functions)

## Real-World Examples

### Web Server Configuration

```go
type ServerConfig struct {
    Host         string        `json:"host"`
    Port         int           `json:"port"`
    ReadTimeout  time.Duration `json:"read_timeout"`
    WriteTimeout time.Duration `json:"write_timeout"`
    TLSEnabled   bool          `json:"tls_enabled"`
    TLSCertFile  string        `json:"tls_cert_file,omitempty"`
}

func LoadServerConfig() *ServerConfig {
    // Validate required variables exist
    env.MustExist("PORT")

    return &ServerConfig{
        Host:         env.GetWithDefault[string]("HOST", "0.0.0.0"),
        Port:         env.Load[int]("PORT"),
        ReadTimeout:  env.GetDurationWithDefault("READ_TIMEOUT", 10*time.Second),
        WriteTimeout: env.GetDurationWithDefault("WRITE_TIMEOUT", 10*time.Second),
        TLSEnabled:   env.GetWithDefault[bool]("TLS_ENABLED", false),
        TLSCertFile:  env.GetWithDefault[string]("TLS_CERT_FILE", ""),
    }
}
```

### Database Configuration

```go
type DatabaseConfig struct {
    Host            string        `json:"host"`
    Port            int           `json:"port"`
    Database        string        `json:"database"`
    Username        string        `json:"username"`
    Password        string        `json:"password"`
    MaxConnections  int           `json:"max_connections"`
    ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
    SSLMode         string        `json:"ssl_mode"`
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
    // Check for required variables
    requiredVars := []string{"DB_HOST", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD"}
    for _, v := range requiredVars {
        if !env.IsSet(v) {
            return nil, fmt.Errorf("required environment variable %s is not set", v)
        }
    }

    // Load with error handling
    host, err := env.Get[string]("DB_HOST")
    if err != nil {
        return nil, err
    }

    return &DatabaseConfig{
        Host:            host,
        Port:            env.GetWithDefault[int]("DB_PORT", 5432),
        Database:        env.Load[string]("DB_DATABASE"),
        Username:        env.Load[string]("DB_USERNAME"),
        Password:        env.Load[string]("DB_PASSWORD"),
        MaxConnections:  env.GetWithDefault[int]("DB_MAX_CONNECTIONS", 25),
        ConnMaxLifetime: env.GetDurationWithDefault("DB_CONN_MAX_LIFETIME", 30*time.Minute),
        SSLMode:         env.GetWithDefault[string]("DB_SSL_MODE", "require"),
    }, nil
}
```

### Microservice Configuration

```go
type ServiceConfig struct {
    ServiceName      string   `json:"service_name"`
    Environment      string   `json:"environment"`
    LogLevel         string   `json:"log_level"`
    MetricsEnabled   bool     `json:"metrics_enabled"`
    UpstreamServices []string `json:"upstream_services"`
    RateLimits       []int    `json:"rate_limits"`
}

func LoadServiceConfig() *ServiceConfig {
    return &ServiceConfig{
        ServiceName:      env.Load[string]("SERVICE_NAME"),
        Environment:      env.GetWithDefault[string]("ENVIRONMENT", "development"),
        LogLevel:         env.GetWithDefault[string]("LOG_LEVEL", "info"),
        MetricsEnabled:   env.GetWithDefault[bool]("METRICS_ENABLED", true),
        UpstreamServices: env.GetSliceWithDefault[string]("UPSTREAM_SERVICES", ",", []string{}),
        RateLimits:       env.GetSliceWithDefault[int]("RATE_LIMITS", " ", []int{100, 1000}),
    }
}
```

### Complete Application Configuration

```go
type AppConfig struct {
    Server   *ServerConfig
    Database *DatabaseConfig
    Service  *ServiceConfig
}

func LoadAppConfig() (*AppConfig, error) {
    // Validate all critical environment variables are present
    criticalVars := []string{
        "SERVICE_NAME", "PORT",
        "DB_HOST", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD",
    }

    var missing []string
    for _, v := range criticalVars {
        if !env.IsSet(v) {
            missing = append(missing, v)
        }
    }

    if len(missing) > 0 {
        return nil, fmt.Errorf("missing required environment variables: %v", missing)
    }

    // Load configurations
    dbConfig, err := LoadDatabaseConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load database config: %w", err)
    }

    return &AppConfig{
        Server:   LoadServerConfig(),
        Database: dbConfig,
        Service:  LoadServiceConfig(),
    }, nil
}

func main() {
    config, err := LoadAppConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Start application with validated configuration
    startServer(config)
}
```

## Common Patterns

### Fail-Fast Startup Validation

```go
func init() {
    // Ensure critical environment variables are set at startup
    env.MustExist("DATABASE_URL", "API_KEY", "SERVICE_NAME")
}
```

### Graceful Degradation

```go
func setupOptionalFeatures() {
    // Enable optional features based on environment
    if env.GetWithDefault[bool]("REDIS_ENABLED", false) {
        setupRedis()
    }
    
    if env.GetWithDefault[bool]("METRICS_ENABLED", false) {
        setupMetrics()
    }
}
```

### Environment-Specific Defaults

```go
func getLogLevel() string {
    environment := env.GetWithDefault[string]("ENVIRONMENT", "development")
    
    switch environment {
    case "production":
        return env.GetWithDefault[string]("LOG_LEVEL", "warn")
    case "staging":
        return env.GetWithDefault[string]("LOG_LEVEL", "info")
    default:
        return env.GetWithDefault[string]("LOG_LEVEL", "debug")
    }
}
```

### Configuration Validation

```go
func validateConfig() error {
    port := env.GetWithDefault[int]("PORT", 8080)
    if port < 1 || port > 65535 {
        return fmt.Errorf("invalid port: %d", port)
    }

    logLevel := env.GetWithDefault[string]("LOG_LEVEL", "info")
    validLevels := []string{"debug", "info", "warn", "error"}
    if !contains(validLevels, logLevel) {
        return fmt.Errorf("invalid log level: %s", logLevel)
    }

    return nil
}
```

## Error Handling

The package provides two error handling patterns:

### 1. Panic-on-Error (Fail Fast)
Use when the application cannot continue without the configuration:

```go
// Will panic if PORT is not set or invalid
port := env.Load[int]("PORT")
```

### 2. Error-Returning (Graceful)
Use when you want to handle configuration errors gracefully:

```go
port, err := env.Get[int]("PORT")
if err != nil {
    log.Printf("PORT not set, using default: %v", err)
    port = 8080
}
```

### Error Types

- **`ErrNotSet`**: Environment variable is not set
- **`ErrParseFailed`**: Environment variable value cannot be parsed to the requested type

## Best Practices

1. **Validate Early**: Use `MustExist()` or validation functions at application startup
2. **Use Appropriate Error Handling**: Panic for critical config, return errors for optional config
3. **Provide Sensible Defaults**: Use `GetWithDefault` for optional configuration
4. **Group Related Config**: Create config structs for related settings
5. **Document Environment Variables**: Maintain a list of all environment variables your app uses
6. **Use Type Safety**: Leverage the generic functions to catch type errors at compile time

## Environment Variable Naming Conventions

- Use UPPER_CASE with underscores
- Group related variables with prefixes (e.g., `DB_HOST`, `DB_PORT`)
- Use descriptive names (`MAX_CONNECTIONS` vs `MAX_CONN`)
- Include units in duration variables (`TIMEOUT_SECONDS` vs `TIMEOUT`)

Example environment file:
```bash
# Server Configuration
HOST=0.0.0.0
PORT=8080
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s

# Database Configuration  
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=myapp
DB_USERNAME=dbuser
DB_PASSWORD=secretpass
DB_MAX_CONNECTIONS=25

# Service Configuration
SERVICE_NAME=user-service
ENVIRONMENT=production
LOG_LEVEL=info
METRICS_ENABLED=true
```

## License

MIT License - see LICENSE file for details.
