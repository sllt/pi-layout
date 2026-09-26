package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sllt/pi-layout/internal/model"
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi-layout/pkg/sid"
	"github.com/sllt/pi-layout/test/mocks/repository"
	"github.com/sllt/pi/pkg/pi/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

var (
	logger *log.Logger
	j      *jwt.JWT
	sf     *sid.Sid
)

func TestMain(m *testing.M) {
	fmt.Println("begin")

	// Set JWT_SECRET for tests
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-testing-32-bytes")

	logger = log.NewLogger(logging.NewLogger(logging.INFO))
	var err error
	j, err = jwt.NewJwt(nil)
	if err != nil {
		panic(err)
	}
	sf = sid.NewSid()

	code := m.Run()
	fmt.Println("test end")

	os.Exit(code)
}

func newUserServiceForTest(ctrl *gomock.Controller) (
	*mock_repository.MockUserRepository,
	*mock_repository.MockUserProfileRepository,
	*mock_repository.MockTransaction,
	service.UserService,
) {
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	mockProfileRepo := mock_repository.NewMockUserProfileRepository(ctrl)
	mockTm := mock_repository.NewMockTransaction(ctrl)
	srv := service.NewService(mockTm, logger, sf, j)

	return mockUserRepo, mockProfileRepo, mockTm, service.NewUserService(srv, mockUserRepo, mockProfileRepo)
}

func TestUserService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, mockProfileRepo, mockTm, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	req := &types.RegisterInput{
		Password: "password",
		Email:    "test@example.com",
	}

	mockUserRepo.EXPECT().GetByEmail(ctx, req.Email).Return(nil, nil)
	mockTm.EXPECT().
		Transaction(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	mockUserRepo.EXPECT().
		Create(ctx, gomock.AssignableToTypeOf(&model.User{})).
		DoAndReturn(func(_ context.Context, user *model.User) error {
			assert.Equal(t, req.Email, user.Email)
			assert.NotEmpty(t, user.UserId)
			assert.NotEqual(t, req.Password, user.Password)
			return nil
		})
	mockProfileRepo.EXPECT().
		Create(ctx, gomock.AssignableToTypeOf(&model.UserProfile{})).
		DoAndReturn(func(_ context.Context, profile *model.UserProfile) error {
			assert.NotEmpty(t, profile.UserId)
			assert.Equal(t, "test", profile.Nickname)
			return nil
		})

	err := userService.Register(ctx, req)

	assert.NoError(t, err)
}

func TestUserService_Register_UsernameExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, _, _, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	req := &types.RegisterInput{
		Password: "password",
		Email:    "test@example.com",
	}

	mockUserRepo.EXPECT().GetByEmail(ctx, req.Email).Return(&model.User{}, nil)

	err := userService.Register(ctx, req)

	assert.Error(t, err)
}

func TestUserService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, _, _, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	req := &types.LoginInput{
		Email:    "xxx@gmail.com",
		Password: "password",
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		t.Error("failed to hash password")
	}

	mockUserRepo.EXPECT().GetByEmail(ctx, req.Email).Return(&model.User{
		UserId:   "user-a",
		Password: string(hashedPassword),
	}, nil)

	token, err := userService.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, _, _, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	req := &types.LoginInput{
		Email:    "xxx@gmail.com",
		Password: "password",
	}

	mockUserRepo.EXPECT().GetByEmail(ctx, req.Email).Return(nil, errors.New("user not found"))

	_, err := userService.Login(ctx, req)

	assert.Error(t, err)
}

func TestUserService_GetProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, mockProfileRepo, _, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	userId := "123"

	mockProfileRepo.EXPECT().GetByUserID(ctx, userId).Return(&model.UserProfile{
		UserId:   userId,
		Nickname: "testuser",
	}, nil)

	user, err := userService.GetProfile(ctx, userId)

	assert.NoError(t, err)
	assert.Equal(t, userId, user.UserId)
	assert.Equal(t, "testuser", user.Nickname)
}

func TestUserService_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, mockProfileRepo, mockTm, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	userId := "123"
	req := &types.UpdateProfileInput{
		Nickname: "testuser",
		Email:    "test@example.com",
	}

	mockUserRepo.EXPECT().GetByID(ctx, userId).Return(&model.User{
		UserId: userId,
		Email:  "old@example.com",
	}, nil)
	mockProfileRepo.EXPECT().GetByUserID(ctx, userId).Return(&model.UserProfile{
		Id:       1,
		UserId:   userId,
		Nickname: "old",
	}, nil)
	mockUserRepo.EXPECT().GetByEmail(ctx, req.Email).Return(nil, nil)
	mockTm.EXPECT().
		Transaction(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	mockUserRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockProfileRepo.EXPECT().
		Update(ctx, gomock.AssignableToTypeOf(&model.UserProfile{})).
		DoAndReturn(func(_ context.Context, profile *model.UserProfile) error {
			assert.Equal(t, req.Nickname, profile.Nickname)
			return nil
		})

	err := userService.UpdateProfile(ctx, userId, req)

	assert.NoError(t, err)
}

func TestUserService_UpdateProfile_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo, _, _, userService := newUserServiceForTest(ctrl)

	ctx := context.Background()
	userId := "123"
	req := &types.UpdateProfileInput{
		Nickname: "testuser",
		Email:    "test@example.com",
	}

	mockUserRepo.EXPECT().GetByID(ctx, userId).Return(nil, errors.New("user not found"))

	err := userService.UpdateProfile(ctx, userId, req)

	assert.Error(t, err)
}
