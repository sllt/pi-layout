// versions:
// 	pi-cli v0.1.0
// 	pi v0.1.0
// 	source: user.proto

package user

import (
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi/pkg/pi"
)

// UserServicePiServer defines the gRPC server implementation.
// This example is deliberately unregistered: profile methods currently trust
// request user IDs. See README.md for the requirements before enabling it.
type UserServicePiServer struct {
	health      *healthServer
	userService service.UserService
}

// NewUserServicePiServerWithService creates a new instance with service dependency
func NewUserServicePiServerWithService(userService service.UserService) *UserServicePiServer {
	return &UserServicePiServer{
		health:      getOrCreateHealthServer(),
		userService: userService,
	}
}

func (s *UserServicePiServer) Register(ctx *pi.Context) (any, error) {
	// 获取 protobuf 请求
	reqWrapper := ctx.Request.(*RegisterRequestWrapper)
	req := reqWrapper.RegisterRequest

	// pb → types 转换
	input := &types.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	}

	if err := s.userService.Register(ctx, input); err != nil {
		return nil, err
	}

	return &RegisterResponse{}, nil
}

func (s *UserServicePiServer) Login(ctx *pi.Context) (any, error) {
	reqWrapper := ctx.Request.(*LoginRequestWrapper)
	req := reqWrapper.LoginRequest

	// pb → types 转换
	input := &types.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	output, err := s.userService.Login(ctx, input)
	if err != nil {
		return nil, err
	}

	// types → pb 转换
	return &LoginResponse{
		AccessToken: output.AccessToken,
	}, nil
}

func (s *UserServicePiServer) GetProfile(ctx *pi.Context) (any, error) {
	reqWrapper := ctx.Request.(*GetProfileRequestWrapper)
	req := reqWrapper.GetProfileRequest

	output, err := s.userService.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// types → pb 转换
	return &GetProfileResponse{
		UserId:   output.UserId,
		Nickname: output.Nickname,
	}, nil
}

func (s *UserServicePiServer) UpdateProfile(ctx *pi.Context) (any, error) {
	reqWrapper := ctx.Request.(*UpdateProfileRequestWrapper)
	req := reqWrapper.UpdateProfileRequest

	// pb → types 转换
	input := &types.UpdateProfileInput{
		Nickname: req.Nickname,
		Email:    req.Email,
	}

	if err := s.userService.UpdateProfile(ctx, req.UserId, input); err != nil {
		return nil, err
	}

	return &UpdateProfileResponse{}, nil
}
