package handler

import (
	v1 "github.com/sllt/pi-layout/api/v1"
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi/pkg/pi"
)

type UserHandler struct {
	*Handler
	userService service.UserService
}

func NewUserHandler(handler *Handler, userService service.UserService) *UserHandler {
	return &UserHandler{
		Handler:     handler,
		userService: userService,
	}
}

func (h *UserHandler) Register(ctx *pi.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	input := &types.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userService.Register(ctx, input); err != nil {
		if h.Handler != nil && h.logger != nil {
			h.logger.WithContext(ctx).Errorf("userService.Register error: %v", err)
		}
		return nil, err
	}

	return nil, nil
}

func (h *UserHandler) Login(ctx *pi.Context, req *v1.LoginRequest) (*v1.LoginResponseData, error) {
	input := &types.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	output, err := h.userService.Login(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.LoginResponseData{
		AccessToken: output.AccessToken,
	}, nil
}

func (h *UserHandler) GetProfile(ctx *pi.Context, req *v1.GetProfileRequest) (*v1.GetProfileResponseData, error) {
	targetID := req.UserId
	if targetID == "" {
		targetID = GetUserIdFromCtx(ctx)
	}

	output, err := h.userService.GetProfile(ctx, targetID)
	if err != nil {
		return nil, err
	}

	return &v1.GetProfileResponseData{
		UserId:   output.UserId,
		Nickname: output.Nickname,
	}, nil
}

func (h *UserHandler) UpdateProfile(ctx *pi.Context, req *v1.UpdateProfileRequest) (*v1.UpdateProfileResponse, error) {
	targetID := req.UserId
	if targetID == "" {
		targetID = GetUserIdFromCtx(ctx)
	}

	input := &types.UpdateProfileInput{
		Nickname: req.Nickname,
		Email:    req.Email,
	}

	if err := h.userService.UpdateProfile(ctx, targetID, input); err != nil {
		return nil, err
	}

	return nil, nil
}
