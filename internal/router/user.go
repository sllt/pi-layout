package router

import (
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi/pkg/pi"
)

func requireAuthMiddleware() pi.PiMiddleware {
	return func(next pi.Handler) pi.Handler {
		return func(ctx *pi.Context) (any, error) {
			if handler.GetUserIdFromCtx(ctx) == "" {
				return nil, errcode.ErrUnauthorized
			}
			return next(ctx)
		}
	}
}

// InitUserRouter registers user routes on the Pi app.
// Token parsing is handled globally via NoStrictAuth middleware (which sets claims
// when a token is present but doesn't reject requests without one).
// Protected routes use a group-scoped Pi middleware for strict auth.
func InitUserRouter(deps RouterDeps) error {
	apiV1 := deps.App.Group("/api/v1").WithDoc(pi.DocTags("用户模块"))

	// No authentication required
	if err := pi.Register(apiV1, deps.UserHandler.Register); err != nil {
		return err
	}
	if err := pi.Register(apiV1, deps.UserHandler.Login); err != nil {
		return err
	}

	// Token required
	userGroup := apiV1.Group("/user").UseMiddleware(requireAuthMiddleware()).WithDoc(pi.DocSecurity("Bearer"))
	if err := pi.Register(userGroup, deps.UserHandler.GetProfile); err != nil {
		return err
	}
	if err := pi.Register(userGroup, deps.UserHandler.UpdateProfile); err != nil {
		return err
	}

	return deps.App.RouteError()
}
