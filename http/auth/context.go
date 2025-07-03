package auth

import (
	"log/slog"

	"github.com/alextanhongpin/core/http/contextkey"
)

// Context keys for storing authentication-related data in HTTP request contexts.
// These provide type-safe storage and retrieval of authentication information
// that can be used by downstream handlers and middleware.
var (
	// ClaimsContext stores JWT claims in the request context.
	// Set by BearerHandler after successful token verification.
	// Use: claims, ok := auth.ClaimsContext.Value(ctx)
	ClaimsContext contextkey.Key[*Claims] = "claims"

	// UsernameContext stores the authenticated username in the request context.
	// Set by BasicHandler after successful basic authentication.
	// Use: username, ok := auth.UsernameContext.Value(ctx)
	UsernameContext contextkey.Key[string] = "username"

	// LoggerContext stores a structured logger in the request context.
	// Used by authentication middleware for consistent logging.
	// Use: logger, ok := auth.LoggerContext.Value(ctx)
	LoggerContext contextkey.Key[*slog.Logger] = "logger"
)
