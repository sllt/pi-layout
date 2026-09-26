package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sllt/pi-layout/internal/model"
	"github.com/sllt/pi-layout/pkg/errcode"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
)

var errNilUser = errors.New("user is nil")
var errNilUserProfile = errors.New("user profile is nil")

const userColumns = "id, user_id, password, email, created_at, updated_at"

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	UpdateEmail(ctx context.Context, userID, email string) error
	UpdatePassword(ctx context.Context, userID, hash string) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type UserProfileRepository interface {
	Create(ctx context.Context, profile *model.UserProfile) error
	Update(ctx context.Context, profile *model.UserProfile) error
	GetByUserID(ctx context.Context, userId string) (*model.UserProfile, error)
}

func NewUserRepository(
	r *Repository,
) UserRepository {
	return &userRepository{
		Repository: r,
	}
}

type userRepository struct {
	*Repository
}

func NewUserProfileRepository(
	r *Repository,
) UserProfileRepository {
	return &userProfileRepository{
		Repository: r,
	}
}

type userProfileRepository struct {
	*Repository
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if user == nil {
		return errNilUser
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	q := r.GetQuerier(ctx)
	result, err := q.ExecContext(ctx,
		"INSERT INTO users (user_id, password, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		user.UserId, user.Password, user.Email, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return uniqueError(err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		user.Id = uint(id)
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if user == nil {
		return errNilUser
	}

	user.UpdatedAt = time.Now()

	q := r.GetQuerier(ctx)
	_, err := q.ExecContext(ctx,
		"UPDATE users SET password = ?, email = ?, updated_at = ? WHERE id = ?",
		user.Password, user.Email, user.UpdatedAt, user.Id,
	)
	return err
}

// Profile/password use separate statements so a stale profile snapshot cannot
// overwrite a newer password. Update remains for legacy whole-record callers.
func (r *userRepository) UpdateEmail(ctx context.Context, userID, email string) error {
	q := r.GetQuerier(ctx)
	result, err := q.ExecContext(ctx, "UPDATE users SET email = ?, updated_at = ? WHERE user_id = ?", email, time.Now(), userID)
	return checkUpdate(ctx, q, result, err, "users", "user_id", userID)
}
func (r *userRepository) UpdatePassword(ctx context.Context, userID, hash string) error {
	q := r.GetQuerier(ctx)
	result, err := q.ExecContext(ctx, "UPDATE users SET password = ?, updated_at = ? WHERE user_id = ?", hash, time.Now(), userID)
	return checkUpdate(ctx, q, result, err, "users", "user_id", userID)
}
func uniqueError(err error) error {
	if piSQL.IsUniqueViolation(err) {
		return errcode.ErrEmailAlreadyUse.WithCause(err)
	}
	return err
}
func checkUpdate(ctx context.Context, q piSQL.Executor, result sql.Result, err error, table, column string, id any) error {
	if err != nil {
		return uniqueError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	// MySQL may report changed rows rather than matched rows. Zero is not enough
	// to classify NotFound; check existence in the same transaction/context.
	var exists int
	err = q.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE "+column+" = ?", id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return errcode.ErrNotFound
	}
	return err
}

func (r *userRepository) GetByID(ctx context.Context, userId string) (*model.User, error) {
	return r.queryOne(ctx, "SELECT "+userColumns+" FROM users WHERE user_id = ?", userId, errcode.ErrNotFound)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.queryOne(ctx, "SELECT "+userColumns+" FROM users WHERE email = ?", email, nil)
}

func (r *userRepository) queryOne(ctx context.Context, query string, arg any, notFoundErr error) (*model.User, error) {
	q := r.GetQuerier(ctx)
	var user model.User
	err := q.Select(ctx, &user, query, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFoundErr
		}
		return nil, err
	}

	return &user, nil
}

func (r *userProfileRepository) Create(ctx context.Context, profile *model.UserProfile) error {
	if profile == nil {
		return errNilUserProfile
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	q := r.GetQuerier(ctx)
	result, err := q.ExecContext(ctx,
		"INSERT INTO user_profiles (user_id, nickname, created_at, updated_at) VALUES (?, ?, ?, ?)",
		profile.UserId, profile.Nickname, profile.CreatedAt, profile.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		profile.Id = uint(id)
	}

	return nil
}

func (r *userProfileRepository) Update(ctx context.Context, profile *model.UserProfile) error {
	if profile == nil {
		return errNilUserProfile
	}

	profile.UpdatedAt = time.Now()

	q := r.GetQuerier(ctx)
	result, err := q.ExecContext(ctx,
		"UPDATE user_profiles SET nickname = ?, updated_at = ? WHERE id = ?",
		profile.Nickname, profile.UpdatedAt, profile.Id,
	)
	return checkUpdate(ctx, q, result, err, "user_profiles", "id", profile.Id)
}

func (r *userProfileRepository) GetByUserID(ctx context.Context, userId string) (*model.UserProfile, error) {
	q := r.GetQuerier(ctx)
	var profile model.UserProfile
	err := q.Select(ctx, &profile, "SELECT id, user_id, nickname, created_at, updated_at FROM user_profiles WHERE user_id = ?", userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcode.ErrNotFound
		}
		return nil, err
	}

	return &profile, nil
}
