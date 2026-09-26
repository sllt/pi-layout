package service

import (
	"context"
	"strings"
	"time"

	"github.com/sllt/pi-layout/internal/model"
	"github.com/sllt/pi-layout/internal/repository"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi-layout/pkg/errcode"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Register(ctx context.Context, input *types.RegisterInput) error
	Login(ctx context.Context, input *types.LoginInput) (*types.LoginOutput, error)
	GetProfile(ctx context.Context, userId string) (*types.UserOutput, error)
	UpdateProfile(ctx context.Context, userId string, input *types.UpdateProfileInput) error
}

func NewUserService(
	service *Service,
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
) UserService {
	return &userService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		Service:     service,
	}
}

type userService struct {
	userRepo    repository.UserRepository
	profileRepo repository.UserProfileRepository
	*Service
}

func (s *userService) Register(ctx context.Context, input *types.RegisterInput) error {
	// check if email already exists
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return errcode.ErrInternalServerError
	}
	if user != nil {
		return errcode.ErrEmailAlreadyUse
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// Generate user ID
	userId, err := s.sid.GenString()
	if err != nil {
		return err
	}
	user = &model.User{
		UserId:   userId,
		Email:    input.Email,
		Password: string(hashedPassword),
	}
	profile := &model.UserProfile{
		UserId:   userId,
		Nickname: defaultNickname(input.Email),
	}

	// Transaction demo: keep account and profile creation atomic.
	err = s.tm.Transaction(ctx, func(ctx context.Context) error {
		if err = s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		return s.profileRepo.Create(ctx, profile)
	})
	return err
}

func (s *userService) Login(ctx context.Context, input *types.LoginInput) (*types.LoginOutput, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil || user == nil {
		return nil, errcode.ErrUnauthorized
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		return nil, errcode.ErrUnauthorized
	}
	token, err := s.jwt.GenToken(user.UserId, time.Now().Add(time.Hour*24*90))
	if err != nil {
		return nil, err
	}

	return &types.LoginOutput{AccessToken: token}, nil
}

func (s *userService) GetProfile(ctx context.Context, userId string) (*types.UserOutput, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &types.UserOutput{
		UserId:   profile.UserId,
		Nickname: profile.Nickname,
	}, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userId string, input *types.UpdateProfileInput) error {
	user, err := s.userRepo.GetByID(ctx, userId)
	if err != nil {
		return err
	}
	profile, err := s.profileRepo.GetByUserID(ctx, userId)
	if err != nil {
		return err
	}

	if input.Email != user.Email {
		existing, err := s.userRepo.GetByEmail(ctx, input.Email)
		if err != nil {
			return errcode.ErrInternalServerError
		}
		if existing != nil && existing.UserId != userId {
			return errcode.ErrEmailAlreadyUse
		}
	}

	user.Email = input.Email
	profile.Nickname = input.Nickname

	return s.tm.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}
		return s.profileRepo.Update(ctx, profile)
	})
}

func defaultNickname(email string) string {
	local, _, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return "New User"
	}
	return local
}
