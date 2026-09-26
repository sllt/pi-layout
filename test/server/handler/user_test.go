package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/types"
	mockservice "github.com/sllt/pi-layout/test/mocks/service"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/testkit"
	"github.com/stretchr/testify/require"
)

func TestLoginThroughPublicTestkit(t *testing.T) {
	app := testkit.New(t, pi.WithConfig(map[string]string{"VALIDATION_LOCALE": "en"}))
	users := mockservice.NewMockUserService(gomock.NewController(t))
	users.EXPECT().Login(gomock.Any(), &types.LoginInput{Email: "a@example.com", Password: "test-password"}).Return(&types.LoginOutput{AccessToken: "test-token"}, nil)
	h := handler.NewUserHandler(hdl, users)
	app.POST("/login", h.Login)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"a@example.com","password":"test-password"}`))
	req.Header.Set("Content-Type", "application/json")
	w := testkit.Request(t, app, req)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"code":0,"data":{"accessToken":"test-token"},"message":"ok"}`, w.Body.String())
	bad := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":`))
	bad.Header.Set("Content-Type", "application/json")
	w = testkit.Request(t, app, bad)
	require.Equal(t, 400, w.Code)
	require.NotContains(t, w.Body.String(), "test-token")
}
