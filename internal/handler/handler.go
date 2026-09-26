package handler

import (
	"context"

	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/auth"
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
	p, _ := auth.FromContext(ctx)
	return p.Subject
}
