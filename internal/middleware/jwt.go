package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/auth"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func StrictAuth(j *jwt.JWT, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := bearerToken(r)
			if tokenString == "" {
				logger.Warn("missing bearer token")
				errcode.WriteHTTPError(w, r, errcode.ErrUnauthorized)
				return
			}

			claims, err := j.ParseToken(tokenString)
			if err != nil {
				logger.Warn("invalid bearer token")
				errcode.WriteHTTPError(w, r, errcode.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			ctx = auth.WithPrincipal(ctx, auth.Principal{Subject: claims.Subject})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NoStrictAuth(j *jwt.JWT, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := bearerToken(r)
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := j.ParseToken(tokenString)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			ctx = auth.WithPrincipal(ctx, auth.Principal{Subject: claims.Subject})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	if len(r.Header.Values("Authorization")) != 1 {
		return ""
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
