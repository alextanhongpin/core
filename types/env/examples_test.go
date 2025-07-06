package env_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alextanhongpin/core/types/env"
)

// Example: Loading basic configuration values
func ExampleMustLoad() {
	// Set some environment variables for the example
	os.Setenv("APP_NAME", "MyApp")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("DEBUG_MODE", "true")
	os.Setenv("MAX_CONNECTIONS", "100")

	// Load different types
	appName := env.MustLoad[string]("APP_NAME")
	port := env.MustLoad[int]("APP_PORT")
	debug := env.MustLoad[bool]("DEBUG_MODE")
	maxConn := env.MustLoad[int]("MAX_CONNECTIONS")

	fmt.Printf("App: %s, Port: %d, Debug: %t, MaxConn: %d\n",
		appName, port, debug, maxConn)

	// Output: App: MyApp, Port: 8080, Debug: true, MaxConn: 100
}

// Example: Graceful handling with Load and defaults
func ExampleLoad() {
	os.Setenv("API_TIMEOUT", "30s")
	// CACHE_SIZE is not set

	// Load with error handling
	timeout, err := env.LoadDuration("API_TIMEOUT")
	if err != nil {
		log.Printf("Failed to get timeout: %v", err)
		return
	}

	// Load with default value
	cacheSize := env.LoadOr[int]("CACHE_SIZE", 1000)

	// Check if variable exists
	hasRedisURL := env.IsSet("REDIS_URL")

	fmt.Printf("Timeout: %v, Cache Size: %d, Has Redis: %t\n",
		timeout, cacheSize, hasRedisURL)

	// Output: Timeout: 30s, Cache Size: 1000, Has Redis: false
}

// Example: Loading arrays/slices from environment
func ExampleMustLoadSlice() {
	os.Setenv("ALLOWED_HOSTS", "localhost,127.0.0.1,example.com")
	os.Setenv("RATE_LIMITS", "10 100 1000")
	os.Setenv("FEATURE_FLAGS", "auth,logging,metrics")

	hosts := env.MustLoadSlice[string]("ALLOWED_HOSTS", ",")
	limits := env.MustLoadSlice[int]("RATE_LIMITS", " ")
	flags := env.MustLoadSlice[string]("FEATURE_FLAGS", ",")

	fmt.Printf("Hosts: %v\n", hosts)
	fmt.Printf("Limits: %v\n", limits)
	fmt.Printf("Flags: %v\n", flags)

	// Output:
	// Hosts: [localhost 127.0.0.1 example.com]
	// Limits: [10 100 1000]
	// Flags: [auth logging metrics]
}

// Real-world example: Web server configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	TLSEnabled   bool          `json:"tls_enabled"`
	TLSCertFile  string        `json:"tls_cert_file,omitempty"`
	TLSKeyFile   string        `json:"tls_key_file,omitempty"`
}

func LoadServerConfig() *ServerConfig {
	// Validate required variables exist
	env.MustExist("PORT")

	return &ServerConfig{
		Host:         env.LoadOr[string]("HOST", "0.0.0.0"),
		Port:         env.MustLoad[int]("PORT"),
		ReadTimeout:  env.LoadDurationOr("READ_TIMEOUT", 10*time.Second),
		WriteTimeout: env.LoadDurationOr("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  env.LoadDurationOr("IDLE_TIMEOUT", 120*time.Second),
		TLSEnabled:   env.LoadOr[bool]("TLS_ENABLED", false),
		TLSCertFile:  env.LoadOr[string]("TLS_CERT_FILE", ""),
		TLSKeyFile:   env.LoadOr[string]("TLS_KEY_FILE", ""),
	}
}

func ExampleServerConfig() {
	// Set up environment for the example
	os.Setenv("PORT", "8443")
	os.Setenv("READ_TIMEOUT", "15s")
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", "/etc/ssl/server.crt")

	config := LoadServerConfig()

	fmt.Printf("Server Config:\n")
	fmt.Printf("  Host: %s\n", config.Host)
	fmt.Printf("  Port: %d\n", config.Port)
	fmt.Printf("  Read Timeout: %v\n", config.ReadTimeout)
	fmt.Printf("  Write Timeout: %v\n", config.WriteTimeout)
	fmt.Printf("  TLS Enabled: %t\n", config.TLSEnabled)
	fmt.Printf("  TLS Cert: %s\n", config.TLSCertFile)

	// Output:
	// Server Config:
	//   Host: 0.0.0.0
	//   Port: 8443
	//   Read Timeout: 15s
	//   Write Timeout: 10s
	//   TLS Enabled: true
	//   TLS Cert: /etc/ssl/server.crt
}

// Real-world example: Database configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Database        string        `json:"database"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	MaxConnections  int           `json:"max_connections"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	SSLMode         string        `json:"ssl_mode"`
	TimeZone        string        `json:"timezone"`
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
	// Check for required variables
	requiredVars := []string{"DB_HOST", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD"}
	for _, v := range requiredVars {
		if !env.IsSet(v) {
			return nil, fmt.Errorf("required environment variable %s is not set", v)
		}
	}

	// Load configuration with proper error handling
	host, err := env.Load[string]("DB_HOST")
	if err != nil {
		return nil, err
	}

	database, err := env.Load[string]("DB_DATABASE")
	if err != nil {
		return nil, err
	}

	username, err := env.Load[string]("DB_USERNAME")
	if err != nil {
		return nil, err
	}

	password, err := env.Load[string]("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	return &DatabaseConfig{
		Driver:          env.LoadOr[string]("DB_DRIVER", "postgres"),
		Host:            host,
		Port:            env.LoadOr[int]("DB_PORT", 5432),
		Database:        database,
		Username:        username,
		Password:        password,
		MaxConnections:  env.LoadOr[int]("DB_MAX_CONNECTIONS", 25),
		MaxIdleConns:    env.LoadOr[int]("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: env.LoadDurationOr("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		SSLMode:         env.LoadOr[string]("DB_SSL_MODE", "require"),
		TimeZone:        env.LoadOr[string]("DB_TIMEZONE", "UTC"),
	}, nil
}

func ExampleDatabaseConfig() {
	// Set up environment
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_DATABASE", "myapp")
	os.Setenv("DB_USERNAME", "dbuser")
	os.Setenv("DB_PASSWORD", "secretpass")
	os.Setenv("DB_MAX_CONNECTIONS", "50")
	os.Setenv("DB_CONN_MAX_LIFETIME", "1h")

	config, err := LoadDatabaseConfig()
	if err != nil {
		log.Printf("Failed to load database config: %v", err)
		return
	}

	fmt.Printf("Database Config:\n")
	fmt.Printf("  Driver: %s\n", config.Driver)
	fmt.Printf("  Host: %s:%d\n", config.Host, config.Port)
	fmt.Printf("  Database: %s\n", config.Database)
	fmt.Printf("  Max Connections: %d\n", config.MaxConnections)
	fmt.Printf("  Connection Lifetime: %v\n", config.ConnMaxLifetime)

	// Output:
	// Database Config:
	//   Driver: postgres
	//   Host: localhost:5432
	//   Database: myapp
	//   Max Connections: 50
	//   Connection Lifetime: 1h0m0s
}

// Real-world example: Microservice configuration
type ServiceConfig struct {
	ServiceName      string   `json:"service_name"`
	Version          string   `json:"version"`
	Environment      string   `json:"environment"`
	LogLevel         string   `json:"log_level"`
	MetricsEnabled   bool     `json:"metrics_enabled"`
	TracingEnabled   bool     `json:"tracing_enabled"`
	UpstreamServices []string `json:"upstream_services"`
	RateLimits       []int    `json:"rate_limits"`
}

func LoadServiceConfig() *ServiceConfig {
	// Use panic for critical missing config - fail fast
	serviceName := env.MustLoad[string]("SERVICE_NAME")

	return &ServiceConfig{
		ServiceName:      serviceName,
		Version:          env.LoadOr[string]("SERVICE_VERSION", "unknown"),
		Environment:      env.LoadOr[string]("ENVIRONMENT", "development"),
		LogLevel:         env.LoadOr[string]("LOG_LEVEL", "info"),
		MetricsEnabled:   env.LoadOr[bool]("METRICS_ENABLED", true),
		TracingEnabled:   env.LoadOr[bool]("TRACING_ENABLED", false),
		UpstreamServices: env.LoadSliceOr[string]("UPSTREAM_SERVICES", ",", []string{}),
		RateLimits:       env.LoadSliceOr[int]("RATE_LIMITS", " ", []int{100, 1000, 10000}),
	}
}

func ExampleServiceConfig() {
	// Set up environment
	os.Setenv("SERVICE_NAME", "user-service")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("LOG_LEVEL", "warn")
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("UPSTREAM_SERVICES", "auth-service,notification-service,payment-service")
	os.Setenv("RATE_LIMITS", "50 500 5000")

	config := LoadServiceConfig()

	fmt.Printf("Service Config:\n")
	fmt.Printf("  Name: %s\n", config.ServiceName)
	fmt.Printf("  Environment: %s\n", config.Environment)
	fmt.Printf("  Log Level: %s\n", config.LogLevel)
	fmt.Printf("  Tracing: %t\n", config.TracingEnabled)
	fmt.Printf("  Upstream Services: %v\n", config.UpstreamServices)
	fmt.Printf("  Rate Limits: %v\n", config.RateLimits)

	// Output:
	// Service Config:
	//   Name: user-service
	//   Environment: production
	//   Log Level: warn
	//   Tracing: true
	//   Upstream Services: [auth-service notification-service payment-service]
	//   Rate Limits: [50 500 5000]
}

// Real-world example: Application startup with validation
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

func ExampleAppConfig() {
	// Set up complete environment
	envVars := map[string]string{
		"SERVICE_NAME": "my-service",
		"PORT":         "8080",
		"DB_HOST":      "localhost",
		"DB_DATABASE":  "myapp",
		"DB_USERNAME":  "user",
		"DB_PASSWORD":  "pass",
		"ENVIRONMENT":  "staging",
		"LOG_LEVEL":    "debug",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	config, err := LoadAppConfig()
	if err != nil {
		log.Printf("Failed to load app config: %v", err)
		return
	}

	fmt.Printf("Application Configuration Loaded Successfully:\n")
	fmt.Printf("  Service: %s (env: %s)\n", config.Service.ServiceName, config.Service.Environment)
	fmt.Printf("  Server: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("  Database: %s@%s:%d/%s\n",
		config.Database.Username, config.Database.Host,
		config.Database.Port, config.Database.Database)

	// Output:
	// Application Configuration Loaded Successfully:
	//   Service: my-service (env: staging)
	//   Server: 0.0.0.0:8080
	//   Database: user@localhost:5432/myapp
}
