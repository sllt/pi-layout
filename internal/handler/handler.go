package handler

import (
	"context"

	"github.com/sllt/pi-layout/internal/middleware"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
)

type Handler struct {
	logger *log.Logger
}

func NewHandler(
	logger *log.Logger,
) *Handler {
	return &Handler{
		logger: logger,
	}
}

func GetUserIdFromCtx(ctx context.Context) string {
	v := ctx.Value(middleware.ClaimsKey)
	if v == nil {
		return ""
	}
	claims, ok := v.(*jwt.MyCustomClaims)
	if !ok {
		return ""
	}
	return claims.UserId
}
