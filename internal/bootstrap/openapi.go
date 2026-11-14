package bootstrap

import (
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/router"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/openapi"
)

// DefaultOpenAPIConfig returns the OpenAPI 3.0.3 metadata and runtime serving
// configuration derived from application config values.
func DefaultOpenAPIConfig(values map[string]string) pi.OpenAPIConfig {
	return pi.OpenAPIConfig{
		Title:       "Pi Example API",
		Version:     "1.0.0",
		Description: "Example API server built with Pi framework.",
		ServeSpec:   values["OPENAPI_SERVE_SPEC"] == "true",
		ServeUI:     values["OPENAPI_SERVE_UI"] == "true",
		SecuritySchemes: map[string]openapi.SecurityScheme{
			"Bearer": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Bearer JWT token",
			},
		},
	}
}

// BuildOpenAPIApp constructs a zero-I/O Pi application with all HTTP routes
// registered for offline OpenAPI specification generation without DB or network setup.
func BuildOpenAPIApp() (*pi.App, error) {
	app, err := pi.Build(
		pi.WithConfig(map[string]string{
			"HTTP_ENABLED":    "false",
			"GRPC_ENABLED":    "false",
			"METRICS_ENABLED": "false",
			"PI_TELEMETRY":    "false",
			"LOG_LEVEL":       "ERROR",
		}),
		pi.WithExplicitHTTPStatus(),
		pi.WithOpenAPI(DefaultOpenAPIConfig(nil)),
	)
	if err != nil {
		return nil, err
	}

	if err := router.InitUserRouter(router.RouterDeps{
		App:         app,
		UserHandler: handler.NewUserHandler(nil, nil),
	}); err != nil {
		return nil, err
	}

	return app, nil
}
