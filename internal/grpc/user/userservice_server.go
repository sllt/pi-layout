// Handwritten protobuf-to-business adapter; wrapper generation preserves this file.

package user

import (
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi/pkg/pi"
)

// UserServicePiServer defines the gRPC server implementation.
// Identity and use-case authorization are enforced by interceptors and service.
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
	req := new(RegisterRequest)
	if err := ctx.Bind(req); err != nil {
		return nil, err
	}

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
	req := new(LoginRequest)
	if err := ctx.Bind(req); err != nil {
		return nil, err
	}

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
	req := new(GetProfileRequest)
	if err := ctx.Bind(req); err != nil {
		return nil, err
	}

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
	req := new(UpdateProfileRequest)
	if err := ctx.Bind(req); err != nil {
		return nil, err
	}

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
