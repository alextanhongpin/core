package auth

import (
	"log/slog"

	"github.com/alextanhongpin/core/http/contextkey"
)

var (
	ClaimsContext   contextkey.Key[*Claims]      = "claims"
	UsernameContext contextkey.Key[string]       = "username"
	LoggerContext   contextkey.Key[*slog.Logger] = "logger"
)
