package v1

type RegisterRequest struct {
	Email    string `json:"email" validate:"required" label:"邮箱"`
	Password string `json:"password" validate:"required" label:"密码"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required" label:"邮箱"`
	Password string `json:"password" validate:"required" label:"密码"`
}

type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname" label:"昵称"`
	Email    string `json:"email" validate:"required" label:"邮箱"`
}

type GetProfileResponseData struct {
	UserId   string `json:"userId"`
	Nickname string `json:"nickname"`
}
