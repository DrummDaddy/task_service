package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJWT(t *testing.T) {
	secret := []byte("test-secret")
	token, err := IssueAccessToken(secret, 123, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseAccessToken(secret, token)
	require.NoError(t, err)
	require.Equal(t, uint64(123), claims.UserID)
	require.NotNil(t, claims.ExpiresAt)
}
