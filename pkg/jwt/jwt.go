package jwt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/config"
)

type JWT struct {
	key              []byte
	issuer, audience string
	ttl              time.Duration
}
type MyCustomClaims struct {
	UserId string `json:"userId"`
	jwt.RegisteredClaims
}

type environment struct{}

func (environment) Get(k string) string { return os.Getenv(k) }
func (environment) GetOrDefault(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}

// NewJwt is the Fx bridge. Invalid credentials/configuration fail construction.
func NewJwt(app *pi.App) (*JWT, error) {
	var cfg config.Config = environment{}
	if app != nil {
		cfg = app.Config
	}
	return New(cfg)
}

func New(cfg config.Config) (*JWT, error) {
	secret := cfg.Get("JWT_SECRET")
	if len(secret) < 32 || strings.Contains(strings.ToLower(secret), "replace-with") || strings.Contains(strings.ToLower(secret), "change-me") {
		return nil, errors.New("JWT_SECRET must be a non-placeholder key of at least 32 bytes")
	}
	ttl, err := time.ParseDuration(cfg.GetOrDefault("JWT_TTL", "1h"))
	if err != nil || ttl <= 0 {
		return nil, errors.New("JWT_TTL must be a positive duration")
	}
	issuer, audience := cfg.GetOrDefault("JWT_ISSUER", "pi-layout"), cfg.GetOrDefault("JWT_AUDIENCE", "pi-api")
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, errors.New("JWT issuer and audience are required")
	}
	return &JWT{key: []byte(secret), issuer: issuer, audience: audience, ttl: ttl}, nil
}

func (j *JWT) Issue(userID string) (string, error) { return j.GenToken(userID, time.Now().Add(j.ttl)) }

func (j *JWT) GenToken(userID string, expiresAt time.Time) (string, error) {
	now := time.Now()
	if strings.TrimSpace(userID) == "" || !expiresAt.After(now) || expiresAt.After(now.Add(j.ttl)) {
		return "", errors.New("invalid token subject or expiry exceeds JWT_TTL")
	}
	claims := MyCustomClaims{UserId: userID, RegisteredClaims: jwt.RegisteredClaims{
		Subject: userID, Issuer: j.issuer, Audience: jwt.ClaimStrings{j.audience},
		ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.key)
}

// ParseToken accepts the encoded JWT, not cookie/query transport or a scheme.
func (j *JWT) ParseToken(encoded string) (*MyCustomClaims, error) {
	if strings.TrimSpace(encoded) == "" {
		return nil, errors.New("token is empty")
	}
	token, err := jwt.ParseWithClaims(encoded, &MyCustomClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unsupported token algorithm")
		}
		return j.key, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(j.issuer), jwt.WithAudience(j.audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok || !token.Valid || claims.Subject == "" || claims.Subject != claims.UserId {
		return nil, errors.New("invalid token subject")
	}
	return claims, nil
}
