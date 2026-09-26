package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/sllt/pi/pkg/pi/config"
	"github.com/stretchr/testify/require"
)

func TestJWTContract(t *testing.T) {
	secret := "a-test-key-with-more-than-thirty-two-bytes"
	j, err := New(config.NewMockConfig(map[string]string{"JWT_SECRET": secret}))
	require.NoError(t, err)
	encoded, err := j.Issue("user-a")
	require.NoError(t, err)
	claims, err := j.ParseToken(encoded)
	require.NoError(t, err)
	require.Equal(t, "user-a", claims.Subject)
	_, err = j.GenToken("user-a", time.Now().Add(90*24*time.Hour))
	require.Error(t, err)
	for _, change := range []string{"algorithm", "issuer", "audience", "expiry", "subject", "noexpiry"} {
		t.Run(change, func(t *testing.T) {
			c := *claims
			method := jwtlib.SigningMethod(jwtlib.SigningMethodHS256)
			switch change {
			case "algorithm":
				method = jwtlib.SigningMethodHS384
			case "issuer":
				c.Issuer = "other"
			case "audience":
				c.Audience = jwtlib.ClaimStrings{"other"}
			case "expiry":
				c.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-time.Hour))
			case "subject":
				c.Subject = "user-b"
			case "noexpiry":
				c.ExpiresAt = nil
			}
			token, err := jwtlib.NewWithClaims(method, c).SignedString([]byte(secret))
			require.NoError(t, err)
			_, err = j.ParseToken(token)
			require.Error(t, err)
		})
	}
	for _, secret := range []string{"", "short", "replace-with-a-strong-random-secret"} {
		_, err := New(config.NewMockConfig(map[string]string{"JWT_SECRET": secret}))
		require.Error(t, err)
	}
}
