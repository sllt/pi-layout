package router

import (
	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/auth"
)

func requireAuthMiddleware() pi.PiMiddleware {
	return func(next pi.Handler) pi.Handler {
		return func(ctx *pi.Context) (any, error) {
			if _, err := auth.Require(ctx); err != nil {
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
	apiV1.POST("/register", deps.UserHandler.Register)
	apiV1.POST("/login", deps.UserHandler.Login)

	// Token required
	userGroup := apiV1.Group("/user").UseMiddleware(requireAuthMiddleware()).WithDoc(pi.DocSecurity("Bearer"))
	userGroup.GET("", deps.UserHandler.GetProfile)
	userGroup.PUT("", deps.UserHandler.UpdateProfile)

	return deps.App.RouteError()
}
