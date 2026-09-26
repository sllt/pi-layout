package router

import (
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"go.uber.org/fx"
)

type RouterDeps struct {
	fx.In

	App         *pi.App
	Logger      *log.Logger
	JWT         *jwt.JWT
	UserHandler *handler.UserHandler
}
