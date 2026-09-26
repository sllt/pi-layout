package handler

import (
	"fmt"
	"github.com/sllt/pi-layout/internal/handler"
	jwt2 "github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/logging"
	"os"
	"testing"
	"time"
)

var (
	userId = "xxx"
)
var logger *log.Logger
var hdl *handler.Handler
var jwt *jwt2.JWT

func TestMain(m *testing.M) {
	fmt.Println("begin")

	// Set JWT_SECRET for tests
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-testing-32-bytes")

	logger = log.NewLogger(logging.NewLogger(logging.INFO))
	hdl = handler.NewHandler(logger)
	var err error
	jwt, err = jwt2.NewJwt(nil)
	if err != nil {
		panic(err)
	}

	code := m.Run()
	fmt.Println("test end")

	os.Exit(code)
}

func genToken(t *testing.T) string {
	token, err := jwt.GenToken(userId, time.Now().Add(time.Hour))
	if err != nil {
		t.Error(err)
		return token
	}
	return token
}
