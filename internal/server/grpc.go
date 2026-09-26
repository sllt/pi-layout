package server

import (
	"context"
	"errors"
	"strconv"

	"github.com/sllt/pi-layout/internal/grpc/user"
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/auth"
	piGRPC "github.com/sllt/pi/pkg/pi/grpc"
	"google.golang.org/grpc"
)

// NewUserGRPCServer opts in explicitly. Authorization always remains in service,
// including when invoked by trusted tasks instead of a transport adapter.
func NewUserGRPCServer(app *pi.App, tokens *jwt.JWT, users service.UserService) error {
	enabled, err := strconv.ParseBool(app.Config.GetOrDefault("USER_GRPC_ENABLED", "false"))
	if err != nil {
		return errors.New("USER_GRPC_ENABLED must be a boolean")
	}
	if !enabled {
		return nil
	}
	listener, err := strconv.ParseBool(app.Config.GetOrDefault("GRPC_ENABLED", "true"))
	if err != nil || !listener {
		return errors.New("USER_GRPC_ENABLED requires GRPC_ENABLED=true")
	}
	if users == nil || tokens == nil {
		return errors.New("gRPC user service requires token verifier and business service")
	}
	verify := func(_ context.Context, encoded string) (auth.Principal, error) {
		claims, err := tokens.ParseToken(encoded)
		if err != nil {
			return auth.Principal{}, err
		}
		return auth.Principal{Subject: claims.Subject}, nil
	}
	prefix := "/" + user.UserService_ServiceDesc.ServiceName + "/"
	app.AddGRPCUnaryInterceptors(piGRPC.ErrorUnaryInterceptor, piGRPC.NewPrincipalUnaryInterceptor(verify,
		prefix+"Register", prefix+"Login", "/grpc.health.v1.Health/Check"))
	app.AddGRPCServerStreamInterceptors(piGRPC.ErrorStreamInterceptor, piGRPC.NewPrincipalStreamInterceptor(verify))
	app.AddGRPCServerOptions(grpc.MaxRecvMsgSize(1 << 20))
	user.RegisterUserServiceServerWithPi(app, user.NewUserServicePiServerWithService(users))
	return nil
}
