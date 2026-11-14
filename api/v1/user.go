package v1

import "github.com/sllt/pi/pkg/pi"

type RegisterRequest struct {
	pi.Meta  `path:"/register" method:"POST" status:"201" tags:"用户模块" summary:"用户注册" description:"目前只支持邮箱登录" errors:"409"`
	Email    string `json:"email" validate:"required" label:"邮箱"`
	Password string `json:"password" validate:"required" label:"密码"`
}

type RegisterResponse struct{}

type LoginRequest struct {
	pi.Meta  `path:"/login" method:"POST" tags:"用户模块" summary:"账号登录" errors:"401"`
	Email    string `json:"email" validate:"required" label:"邮箱"`
	Password string `json:"password" validate:"required" label:"密码"`
}

type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}

type GetProfileRequest struct {
	pi.Meta `path:"" method:"GET" tags:"用户模块" summary:"获取用户信息" errors:"401,403,404"`
	UserId  string `query:"userId" dc:"目标用户 ID（留空表示当前登录用户）"`
}

type UpdateProfileRequest struct {
	pi.Meta  `path:"" method:"PUT" status:"204" tags:"用户模块" summary:"修改用户信息" errors:"401,403,404,409"`
	UserId   string `query:"userId" dc:"目标用户 ID（留空表示当前登录用户）"`
	Nickname string `json:"nickname" label:"昵称"`
	Email    string `json:"email" validate:"required" label:"邮箱"`
}

type UpdateProfileResponse struct{}

type GetProfileResponseData struct {
	UserId   string `json:"userId"`
	Nickname string `json:"nickname"`
}
