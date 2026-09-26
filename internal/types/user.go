package types

import (
	"github.com/sllt/pi/pkg/pi/apperror"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// RegisterInput 用户注册输入
type RegisterInput struct {
	Email    string
	Password string
}

// LoginInput 用户登录输入
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput 登录输出
type LoginOutput struct {
	AccessToken string
}

// UserOutput 用户信息输出
type UserOutput struct {
	UserId   string
	Nickname string
}

// UpdateProfileInput 更新用户资料输入
type UpdateProfileInput struct {
	Nickname string
	Email    string
}

func invalid(field, rule string) error {
	return apperror.New(apperror.InvalidArgument, 400, "invalid input").WithDetails(apperror.Detail{Field: field, Rule: rule})
}
func validateEmail(email string) error {
	if len(email) == 0 || len(email) > 254 || email != strings.TrimSpace(email) {
		return invalid("email", "valid email required")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !strings.Contains(email, "@") {
		return invalid("email", "valid email required")
	}
	return nil
}
func validatePassword(password string) error {
	if !utf8.ValidString(password) || len(password) < 8 || len(password) > 72 {
		return invalid("password", "must contain 8 to 72 bytes")
	}
	return nil
}
func (i *RegisterInput) Validate() error {
	if i == nil {
		return invalid("input", "required")
	}
	if err := validateEmail(i.Email); err != nil {
		return err
	}
	return validatePassword(i.Password)
}
func (i *LoginInput) Validate() error {
	if i == nil {
		return invalid("input", "required")
	}
	if err := validateEmail(i.Email); err != nil {
		return err
	}
	return validatePassword(i.Password)
}
func (i *UpdateProfileInput) Validate() error {
	if i == nil {
		return invalid("input", "required")
	}
	if err := validateEmail(i.Email); err != nil {
		return err
	}
	if !utf8.ValidString(i.Nickname) || utf8.RuneCountInString(i.Nickname) > 64 || len(i.Nickname) > 256 || strings.ContainsAny(i.Nickname, "\r\n\x00") {
		return invalid("nickname", "must be valid text up to 64 characters")
	}
	return nil
}
