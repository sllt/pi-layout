package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi/pkg/pi/config"
	"github.com/stretchr/testify/require"
)

func TestBearerOnlyIdentity(t *testing.T) {
	j, err := jwt.New(config.NewMockConfig(map[string]string{"JWT_SECRET": "test-secret-with-at-least-thirty-two-bytes"}))
	require.NoError(t, err)
	token, err := j.Issue("user-a")
	require.NoError(t, err)
	for _, transport := range []string{"bearer", "cookie", "query", "raw", "wrong-scheme"} {
		t.Run(transport, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			switch transport {
			case "bearer":
				r.Header.Set("Authorization", "Bearer "+token)
			case "cookie":
				r.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
			case "query":
				r.URL.RawQuery = "accessToken=" + token
			case "raw":
				r.Header.Set("Authorization", token)
			case "wrong-scheme":
				r.Header.Set("Authorization", "Basic "+token)
			}
			h := NoStrictAuth(j, testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, _ := r.Context().Value(ClaimsKey).(*jwt.MyCustomClaims)
				if transport == "bearer" {
					require.NotNil(t, claims)
					require.Equal(t, "user-a", claims.Subject)
				} else {
					require.Nil(t, claims)
				}
			}))
			h.ServeHTTP(httptest.NewRecorder(), r)
		})
	}
}
