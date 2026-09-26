package server

import (
	"github.com/sllt/pi-layout/internal/middleware"
	"github.com/sllt/pi-layout/internal/router"
	"github.com/sllt/pi/pkg/pi"
)

func NewHTTPServer(deps router.RouterDeps) {
	app := deps.App

	// Register global middleware
	app.Use(
		// NoStrictAuth runs globally: extracts JWT claims into context when token is present,
		// but does not reject requests without a token. Strict routes are enforced by
		// group-scoped Pi middleware in internal/router.
		middleware.NoStrictAuth(deps.JWT, deps.Logger),
	)

	// Root route
	app.GET("/", func(ctx *pi.Context) (any, error) {
		return map[string]any{
			":)": "Thank you for using pi!",
		}, nil
	})

	// Register user routes (HTTP)
	router.InitUserRouter(deps)

	// The gRPC user example remains unregistered until it enforces verified
	// identity, resource authorization and the same validation as HTTP.
}
