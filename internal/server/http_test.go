package server

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/router"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	mockservice "github.com/sllt/pi-layout/test/mocks/service"
	"github.com/sllt/pi/pkg/pi"
	"github.com/stretchr/testify/require"
)

func TestHTTPServer_DefaultUserBoundary(t *testing.T) {
	// An occupied gRPC port makes accidental service registration fail startup.
	// This checks the real composition root, without relying on private App fields.
	lc := net.ListenConfig{}
	grpcPort, err := lc.Listen(t.Context(), "tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = grpcPort.Close() })

	t.Chdir(t.TempDir())
	require.NoError(t, os.Mkdir("configs", 0o700))
	for key, value := range map[string]string{
		"PI_TELEMETRY": "false", "METRICS_PORT": "0", "JWT_SECRET": "test-user-boundary-secret-at-least-32-bytes",
		"DB_HOST": "", "DB_DIALECT": "", "REDIS_HOST": "", "PUBSUB_BACKEND": "",
		"TRACE_EXPORTER": "", "TRACER_URL": "", "TRACER_HOST": "", "REMOTE_LOG_URL": "",
		"CERT_FILE": "", "KEY_FILE": "", "LOG_LEVEL": "ERROR", "HTTP_ADDR": "127.0.0.1:0", "USER_GRPC_ENABLED": "false",
		"GRPC_PORT":            strconv.Itoa(grpcPort.Addr().(*net.TCPAddr).Port),
		"CORS_ALLOWED_ORIGINS": "https://web.example", "CORS_ALLOW_CREDENTIALS": "false",
	} {
		t.Setenv(key, value)
	}

	app := pi.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Stop(ctx))
	})
	logger := log.NewLogger(app.Logger())
	tokens, err := jwt.NewJwt(app)
	require.NoError(t, err)
	token, err := tokens.GenToken("user-a", time.Now().Add(time.Hour))
	require.NoError(t, err)

	users := mockservice.NewMockUserService(gomock.NewController(t))
	users.EXPECT().Register(gomock.Any(), &types.RegisterInput{
		Email: "a@example.com", Password: "test-password",
	}).Return(nil)
	users.EXPECT().Login(gomock.Any(), &types.LoginInput{
		Email: "a@example.com", Password: "test-password",
	}).Return(&types.LoginOutput{AccessToken: token}, nil)
	users.EXPECT().GetProfile(gomock.Any(), "user-a").Return(&types.UserOutput{
		UserId: "user-a", Nickname: "Alice",
	}, nil)
	users.EXPECT().UpdateProfile(gomock.Any(), "user-a", &types.UpdateProfileInput{
		Email: "a@example.com", Nickname: "Alice updated",
	}).Return(nil)

	require.NoError(t, NewHTTPServer(router.RouterDeps{
		App: app, Logger: logger, JWT: tokens,
		UserHandler: handler.NewUserHandler(handler.NewHandler(logger), users),
	}))
	require.NoError(t, app.Start(t.Context()), "default HTTP setup must not start gRPC")

	cases := []struct {
		name, method, path, token, body string
		status                          int
	}{
		{"register", http.MethodPost, "/api/v1/register", "",
			`{"email":"a@example.com","password":"test-password"}`, http.StatusCreated},
		{"login", http.MethodPost, "/api/v1/login", "",
			`{"email":"a@example.com","password":"test-password"}`, http.StatusOK},
		{"anonymous profile", http.MethodGet, "/api/v1/user/", "", "", http.StatusUnauthorized},
		{"invalid token", http.MethodGet, "/api/v1/user/", "invalid", "", http.StatusUnauthorized},
		{"query token rejected", http.MethodGet, "/api/v1/user/?accessToken=" + token, "", "", http.StatusUnauthorized},
		{"anonymous update", http.MethodPut, "/api/v1/user/", "",
			`{"email":"b@example.com","nickname":"Bob"}`, http.StatusUnauthorized},
		{"cross profile", http.MethodGet, "/api/v1/user/?userId=user-b", token, "", http.StatusForbidden},
		{"own profile", http.MethodGet, "/api/v1/user/", token, "", http.StatusOK},
		{"own update", http.MethodPut, "/api/v1/user/", token,
			`{"email":"a@example.com","nickname":"Alice updated"}`, http.StatusNoContent},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	baseURL := "http://" + app.HTTPAddress()
	for _, origin := range []string{"https://web.example", "https://evil.example"} {
		req, err := http.NewRequestWithContext(t.Context(), "OPTIONS", baseURL+"/api/v1/user/", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "PUT")
		req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		if origin == "https://web.example" {
			require.Equal(t, 204, resp.StatusCode)
			require.Equal(t, []string{origin}, resp.Header.Values("Access-Control-Allow-Origin"))
		} else {
			require.Equal(t, 403, resp.StatusCode)
			require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, reqErr := http.NewRequestWithContext(t.Context(), tc.method, baseURL+tc.path, strings.NewReader(tc.body))
			require.NoError(t, reqErr)
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			resp, doErr := client.Do(req)
			require.NoError(t, doErr)
			defer resp.Body.Close()
			require.Equal(t, tc.status, resp.StatusCode)
			if tc.status == http.StatusNoContent {
				data, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Empty(t, data)
				return
			}
			var body struct {
				Code int `json:"code"`
				Data struct {
					UserID      string `json:"userId"`
					AccessToken string `json:"accessToken"`
				} `json:"data"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			if tc.status >= 400 {
				require.NotZero(t, body.Code)
			} else {
				require.Zero(t, body.Code)
			}
			if tc.name == "own profile" {
				require.Equal(t, "user-a", body.Data.UserID)
			}
			if tc.name == "login" {
				require.Equal(t, token, body.Data.AccessToken)
			}
		})
	}
}
