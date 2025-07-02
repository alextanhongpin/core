# Architecture Decision Record: Go HTTP Core Package

## Status
Accepted

## Context
We need a robust, production-ready HTTP package for Go applications that provides essential functionality for building HTTP services. The package should be modular, well-tested, and follow Go best practices while providing common patterns needed in production environments.

## Decision
We will build a modular HTTP package with the following components:

### Core Components

1. **Authentication (`auth/`)**
   - Basic HTTP authentication
   - Bearer token authentication
   - JWT token validation
   - Context-based auth state management

2. **Request Handling (`request/`)**
   - Type-safe request parameter extraction
   - JSON/form data parsing
   - Value validation and trimming

3. **Response Management (`response/`)**
   - Structured JSON responses
   - Error handling and formatting
   - Content negotiation

4. **Middleware (`middleware/`)**
   - Request logging
   - Panic recovery
   - CORS handling
   - Rate limiting

5. **Health Checks (`health/`)**
   - Liveness endpoints
   - Readiness endpoints
   - Service dependency monitoring

6. **Server Management (`server/`)**
   - Graceful shutdown
   - Signal handling
   - Configuration management

7. **Utilities**
   - Request chaining (`chain/`)
   - Context keys (`contextkey/`)
   - Request ID generation (`requestid/`)
   - Template rendering (`templ/`)
   - Webhook verification (`webhook/`)
   - Cursor-based pagination (`pagination/`)

### Design Principles

1. **Modularity**: Each component can be used independently
2. **Type Safety**: Strong typing throughout the API
3. **Context Propagation**: Proper use of Go context for request scoping
4. **Error Handling**: Consistent error patterns and responses
5. **Testing**: Comprehensive test coverage with table-driven tests
6. **Performance**: Minimal allocations and efficient processing

## Rationale

### Why These Components?

1. **Authentication**: Essential for any production API
2. **Request/Response**: Common patterns for HTTP handling
3. **Middleware**: Cross-cutting concerns that every service needs
4. **Health Checks**: Required for container orchestration and monitoring
5. **Server**: Graceful shutdown is critical for production deployments

### Design Decisions

1. **Context-First Approach**: All components use Go context for request scoping and cancellation
2. **Interface-Based Design**: Components depend on interfaces, not concrete types
3. **Functional Options**: Configuration through functional options pattern
4. **Structured Errors**: Consistent error handling with typed errors
5. **HTTP-First**: Designed specifically for HTTP services, not generic

### Technology Choices

1. **Standard Library**: Built on Go's standard `net/http` package
2. **Minimal Dependencies**: Only essential external dependencies
3. **Go Modules**: Modern Go module structure
4. **Testing**: Standard Go testing with table-driven tests

## Consequences

### Positive
- Modular design allows picking and choosing components
- Strong typing reduces runtime errors
- Comprehensive testing ensures reliability
- Production-ready patterns out of the box
- Consistent API across all components

### Negative
- Learning curve for the specific patterns used
- More opinionated than generic HTTP libraries
- Requires understanding of Go context patterns

## Implementation Details

### Directory Structure
```
http/
├── auth/           # Authentication components
├── chain/          # Request chaining utilities
├── contextkey/     # Context key management
├── handler/        # Base handler patterns
├── health/         # Health check endpoints
├── middleware/     # HTTP middleware
├── pagination/     # Cursor-based pagination
├── request/        # Request parsing and validation
├── requestid/      # Request ID generation
├── response/       # Response formatting
├── server/         # Server management
├── templ/          # Template rendering
└── webhook/        # Webhook verification
```

### Testing Strategy
- Unit tests for all components
- Integration tests for complex interactions
- HTTP test files for endpoint testing
- Table-driven test patterns
- Mock interfaces where appropriate

### Error Handling
- Typed errors for different failure modes
- Consistent HTTP status code mapping
- Structured error responses
- Context cancellation support

## Alternatives Considered

1. **Single Monolithic Package**: Rejected due to lack of modularity
2. **Generic HTTP Framework**: Rejected as too generic for our specific needs
3. **External Framework (gin, echo, etc.)**: Rejected to maintain control and minimize dependencies
4. **Microservice Per Component**: Rejected as too complex for a library

## References
- [Go HTTP Best Practices](https://golang.org/doc/effective_go.html)
- [Go Context Patterns](https://blog.golang.org/context)
- [REST API Design Guidelines](https://restfulapi.net/)
- [HTTP Status Codes](https://httpstatuses.com/)

## Review Notes
- Architecture reviewed for production readiness
- Security patterns validated
- Performance characteristics analyzed
- Testing coverage verified

---
*ADR Author*: GitHub Copilot  
*Date*: July 2, 2025  
*Version*: 1.0
