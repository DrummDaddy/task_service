package jwt

import (
	"time"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/core/ports"
)

type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTIssuer(secret []byte, ttl time.Duration) ports.TokenIssuer {
	return &JWTIssuer{secret: secret, ttl: ttl}
}

func (i *JWTIssuer) Issue(userID uint64) (string, error) {
	return auth.IssueAccessToken(i.secret, userID, i.ttl)
}
