package auth

import (
	"log/slog"

	"github.com/alextanhongpin/core/http/contextkey"
)

var (
	ClaimsContext contextkey.Key[*Claims] = "claims"

	LoggerContext contextkey.Key[*slog.Logger] = "logger"
)
