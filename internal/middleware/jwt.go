package middleware

import (
	"context"
	"net/http"

	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func StrictAuth(j *jwt.JWT, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				logger.Warnf("No token url=%s", r.URL)
				errcode.WriteHTTPError(w, r, errcode.ErrUnauthorized)
				return
			}

			claims, err := j.ParseToken(tokenString)
			if err != nil {
				logger.Errorf("token error url=%s err=%v", r.URL, err)
				errcode.WriteHTTPError(w, r, errcode.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NoStrictAuth(j *jwt.JWT, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				cookie, err := r.Cookie("accessToken")
				if err == nil {
					tokenString = cookie.Value
				}
			}
			if tokenString == "" {
				tokenString = r.URL.Query().Get("accessToken")
			}
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
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
