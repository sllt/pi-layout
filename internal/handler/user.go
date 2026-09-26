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

// Register godoc
// @Summary 用户注册
// @Schemes
// @Description 目前只支持邮箱登录
// @Tags 用户模块
// @Accept json
// @Produce json
// @Param request body v1.RegisterRequest true "params"
// @Success 200 {object} errcode.Response
// @Router /register [post]
func (h *UserHandler) Register(ctx *pi.Context) (any, error) {
	req := new(v1.RegisterRequest)
	if err := ctx.Bind(req); err != nil {
		return nil, err
	}

	input := &types.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userService.Register(ctx, input); err != nil {
		h.logger.Errorf("userService.Register error: %v", err)
		return nil, err
	}

	return nil, nil
}

// Login godoc
// @Summary 账号登录
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Param request body v1.LoginRequest true "params"
// @Success 200 {object} errcode.Response{data=v1.LoginResponseData}
// @Router /login [post]
func (h *UserHandler) Login(ctx *pi.Context) (any, error) {
	var req v1.LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return nil, err
	}

	input := &types.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	output, err := h.userService.Login(ctx, input)
	if err != nil {
		return nil, err
	}

	return v1.LoginResponseData{
		AccessToken: output.AccessToken,
	}, nil
}

// GetProfile godoc
// @Summary 获取用户信息
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} errcode.Response{data=v1.GetProfileResponseData}
// @Router /user [get]
func (h *UserHandler) GetProfile(ctx *pi.Context) (any, error) {
	userId := GetUserIdFromCtx(ctx)

	output, err := h.userService.GetProfile(ctx, userId)
	if err != nil {
		return nil, err
	}

	return v1.GetProfileResponseData{
		UserId:   output.UserId,
		Nickname: output.Nickname,
	}, nil
}

// UpdateProfile godoc
// @Summary 修改用户信息
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.UpdateProfileRequest true "params"
// @Success 200 {object} errcode.Response
// @Router /user [put]
func (h *UserHandler) UpdateProfile(ctx *pi.Context) (any, error) {
	userId := GetUserIdFromCtx(ctx)

	var req v1.UpdateProfileRequest
	if err := ctx.Bind(&req); err != nil {
		return nil, err
	}

	input := &types.UpdateProfileInput{
		Nickname: req.Nickname,
		Email:    req.Email,
	}

	if err := h.userService.UpdateProfile(ctx, userId, input); err != nil {
		return nil, err
	}

	return nil, nil
}
