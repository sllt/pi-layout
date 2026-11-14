package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sllt/pi-layout/internal/bootstrap"
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/router"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/openapi"
	"github.com/sllt/pi/pkg/pi/testkit"
	"github.com/stretchr/testify/require"
)

func requireBearerSecurity(t *testing.T, sec *[]openapi.SecurityRequirement) {
	t.Helper()
	require.NotNil(t, sec)
	require.NotEmpty(t, *sec)
	for _, req := range *sec {
		if _, ok := req["Bearer"]; ok {
			return
		}
	}
	t.Fatalf("expected Bearer security requirement, got %#v", *sec)
}

func TestBuildOpenAPIApp_OfflineSpecAndParity(t *testing.T) {
	app, err := bootstrap.BuildOpenAPIApp()
	require.NoError(t, err)

	spec, err := app.OpenAPISpec()
	require.NoError(t, err)
	require.Equal(t, openapi.Version303, spec.OpenAPI)
	require.Equal(t, "Pi Example API", spec.Info.Title)
	require.Equal(t, "1.0.0", spec.Info.Version)
	require.NotNil(t, spec.Components)
	require.Contains(t, spec.Components.SecuritySchemes, "Bearer")
	require.Equal(t, "http", spec.Components.SecuritySchemes["Bearer"].Type)
	require.Equal(t, "bearer", spec.Components.SecuritySchemes["Bearer"].Scheme)

	require.NotContains(t, spec.Paths, "/api/v1/user/")

	regPath := spec.Paths["/api/v1/register"]
	require.NotNil(t, regPath)
	require.NotNil(t, regPath.Post)
	require.Contains(t, regPath.Post.Responses, "201")
	require.Contains(t, regPath.Post.Responses, "409")
	require.Contains(t, regPath.Post.Tags, "用户模块")

	loginPath := spec.Paths["/api/v1/login"]
	require.NotNil(t, loginPath)
	require.NotNil(t, loginPath.Post)
	login200 := loginPath.Post.Responses["200"]
	require.NotNil(t, login200)
	require.Contains(t, login200.Content, "application/json")
	loginEnvelope := login200.Content["application/json"].Schema
	require.NotNil(t, loginEnvelope)
	loginData := loginEnvelope.Properties["data"]
	require.NotNil(t, loginData)
	require.Equal(t, "#/components/schemas/LoginResponseData", loginData.Ref)
	require.False(t, loginData.Nullable)
	require.Empty(t, loginData.AllOf)
	require.Contains(t, spec.Components.Schemas, "LoginResponseData")
	require.Contains(t, spec.Components.Schemas["LoginResponseData"].Properties, "accessToken")

	userPath := spec.Paths["/api/v1/user"]
	require.NotNil(t, userPath)
	require.NotNil(t, userPath.Get)
	require.Contains(t, userPath.Get.Responses, "200")
	requireBearerSecurity(t, userPath.Get.Security)
	require.NotNil(t, userPath.Put)
	require.Contains(t, userPath.Put.Responses, "204")
	requireBearerSecurity(t, userPath.Put.Security)

	jsonBytes, err := app.OpenAPIJSON()
	require.NoError(t, err)
	var decoded openapi.Document
	require.NoError(t, json.Unmarshal(jsonBytes, &decoded))
	require.Equal(t, openapi.Version303, decoded.OpenAPI)
	require.Contains(t, decoded.Paths, "/api/v1/register")
	require.Contains(t, decoded.Paths, "/api/v1/login")
	require.Contains(t, decoded.Paths, "/api/v1/user")
	require.NotContains(t, decoded.Paths, "/api/v1/user/")

	yamlBytes, err := app.OpenAPIYAML()
	require.NoError(t, err)
	yamlText := string(yamlBytes)
	require.Contains(t, yamlText, "openapi: 3.0.3")
	require.Contains(t, yamlText, "title: Pi Example API")
	require.Contains(t, yamlText, "/api/v1/register:")
	require.Contains(t, yamlText, "/api/v1/login:")
	require.Contains(t, yamlText, "/api/v1/user:")
	require.NotContains(t, yamlText, "/api/v1/user/:")
}

func setupOpenAPIServingApp(t *testing.T, serveSpec bool) *pi.App {
	t.Helper()
	cfgValues := map[string]string{
		"HTTP_ENABLED":       "false",
		"GRPC_ENABLED":       "false",
		"METRICS_ENABLED":    "false",
		"PI_TELEMETRY":       "false",
		"LOG_LEVEL":          "ERROR",
		"USER_GRPC_ENABLED":  "false",
		"JWT_SECRET":         "test-openapi-secret-at-least-32-bytes",
		"OPENAPI_SERVE_SPEC": strconv.FormatBool(serveSpec),
		"OPENAPI_SERVE_UI":   "false",
	}
	app := testkit.New(
		t,
		pi.WithConfig(cfgValues),
		pi.WithExplicitHTTPStatus(),
		pi.WithOpenAPI(bootstrap.DefaultOpenAPIConfig(cfgValues)),
	)
	logger := log.NewLogger(app.Logger())
	tokens, err := jwt.NewJwt(app)
	require.NoError(t, err)

	require.NoError(t, NewHTTPServer(router.RouterDeps{
		App:         app,
		Logger:      logger,
		JWT:         tokens,
		UserHandler: handler.NewUserHandler(handler.NewHandler(logger), nil),
	}))
	return app
}

func TestHTTPServer_OpenAPIServingToggle(t *testing.T) {
	t.Run("ServeSpec disabled returns 404", func(t *testing.T) {
		app := setupOpenAPIServingApp(t, false)
		rec := testkit.Request(t, app, httptest.NewRequest(http.MethodGet, "/.well-known/openapi.json", nil))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("ServeSpec enabled serves OpenAPI 3.0.3 JSON", func(t *testing.T) {
		app := setupOpenAPIServingApp(t, true)
		rec := testkit.Request(t, app, httptest.NewRequest(http.MethodGet, "/.well-known/openapi.json", nil))
		require.Equal(t, http.StatusOK, rec.Code)

		var doc openapi.Document
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
		require.Equal(t, openapi.Version303, doc.OpenAPI)
		require.Contains(t, doc.Paths, "/api/v1/register")
		require.Contains(t, doc.Paths, "/api/v1/login")
		require.Contains(t, doc.Paths, "/api/v1/user")
		require.NotContains(t, doc.Paths, "/api/v1/user/")
	})
}
