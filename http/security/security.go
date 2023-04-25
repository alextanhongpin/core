package security

import (
	"github.com/alextanhongpin/core/http/contextkey"
	"github.com/golang-jwt/jwt/v5"
)

var AuthContext contextkey.ContextKey[jwt.Claims] = "auth_ctx"
