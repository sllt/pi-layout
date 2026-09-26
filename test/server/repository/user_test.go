package repository

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sllt/pi-layout/internal/model"
	"github.com/sllt/pi-layout/internal/repository"
	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi/datasource"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
	"github.com/stretchr/testify/assert"
)

var logger *log.Logger

// testDB wraps *sql.DB to satisfy infra.DB interface for testing.
type testDB struct {
	*sql.DB
}

func (t *testDB) Begin() (*piSQL.Tx, error) {
	return nil, errors.New("not implemented in test")
}

func (t *testDB) Select(ctx context.Context, data any, query string, args ...any) error {
	rows, err := t.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("data must be a non-nil pointer")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return nil
	}

	if !rows.Next() {
		return sql.ErrNoRows
	}

	cols, _ := rows.Columns()
	fields := make([]any, len(cols))
	fieldMap := make(map[string]int)

	for i := 0; i < elem.Type().NumField(); i++ {
		tag := elem.Type().Field(i).Tag.Get("db")
		if tag != "" {
			fieldMap[tag] = i
		}
	}

	for i, col := range cols {
		if idx, ok := fieldMap[col]; ok {
			fields[i] = elem.Field(idx).Addr().Interface()
		} else {
			var dummy any
			fields[i] = &dummy
		}
	}

	return rows.Scan(fields...)
}

func (t *testDB) HealthCheck() *datasource.Health { return nil }

func (t *testDB) Dialect() string { return "mysql" }

func setupRepository(t *testing.T) (*repository.Repository, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	db := &testDB{DB: mockDB}
	repo := repository.NewRepository(logger, db)

	return repo, mock
}

func setupUserRepository(t *testing.T) (repository.UserRepository, sqlmock.Sqlmock) {
	repo, mock := setupRepository(t)
	return repository.NewUserRepository(repo), mock
}

func setupUserProfileRepository(t *testing.T) (repository.UserProfileRepository, sqlmock.Sqlmock) {
	repo, mock := setupRepository(t)
	return repository.NewUserProfileRepository(repo), mock
}

func TestUserRepository_Create(t *testing.T) {
	userRepo, mock := setupUserRepository(t)

	ctx := context.Background()
	user := &model.User{
		UserId:   "123",
		Password: "password",
		Email:    "test@example.com",
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.UserId, user.Password, user.Email, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := userRepo.Create(ctx, user)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.Id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	userRepo, mock := setupUserRepository(t)

	ctx := context.Background()
	user := &model.User{
		Id:       1,
		UserId:   "123",
		Password: "password",
		Email:    "test@example.com",
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs(user.Password, user.Email, sqlmock.AnyArg(), user.Id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := userRepo.Update(ctx, user)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateEmailZeroRowsChecksExistence(t *testing.T) {
	for _, exists := range []bool{true, false} {
		r, mock := setupUserRepository(t)
		mock.ExpectExec("UPDATE users SET email").WithArgs("new@example.com", sqlmock.AnyArg(), "id").WillReturnResult(sqlmock.NewResult(0, 0))
		rows := sqlmock.NewRows([]string{"exists"})
		if exists {
			rows.AddRow(1)
		}
		mock.ExpectQuery("SELECT 1 FROM users WHERE user_id").WithArgs("id").WillReturnRows(rows)
		err := r.UpdateEmail(t.Context(), "id", "new@example.com")
		if exists {
			assert.NoError(t, err)
		} else {
			assert.ErrorIs(t, err, errcode.ErrNotFound)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestUserRepository_GetById(t *testing.T) {
	userRepo, mock := setupUserRepository(t)

	ctx := context.Background()
	userId := "123"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "password", "email", "created_at", "updated_at"}).
		AddRow(1, "123", "password", "test@example.com", now, now)
	mock.ExpectQuery("SELECT id, user_id, password, email, created_at, updated_at FROM users WHERE user_id").
		WithArgs(userId).
		WillReturnRows(rows)

	user, err := userRepo.GetByID(ctx, userId)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "123", user.UserId)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail(t *testing.T) {
	userRepo, mock := setupUserRepository(t)

	ctx := context.Background()
	email := "test@example.com"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "password", "email", "created_at", "updated_at"}).
		AddRow(1, "123", "password", "test@example.com", now, now)
	mock.ExpectQuery("SELECT id, user_id, password, email, created_at, updated_at FROM users WHERE email").
		WithArgs(email).
		WillReturnRows(rows)

	user, err := userRepo.GetByEmail(ctx, email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProfileRepository_Create(t *testing.T) {
	profileRepo, mock := setupUserProfileRepository(t)

	ctx := context.Background()
	profile := &model.UserProfile{
		UserId:   "123",
		Nickname: "Test",
	}

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs(profile.UserId, profile.Nickname, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := profileRepo.Create(ctx, profile)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), profile.Id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProfileRepository_Update(t *testing.T) {
	profileRepo, mock := setupUserProfileRepository(t)

	ctx := context.Background()
	profile := &model.UserProfile{
		Id:       1,
		UserId:   "123",
		Nickname: "Test",
	}

	mock.ExpectExec("UPDATE user_profiles SET").
		WithArgs(profile.Nickname, sqlmock.AnyArg(), profile.Id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := profileRepo.Update(ctx, profile)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProfileRepository_GetByUserID(t *testing.T) {
	profileRepo, mock := setupUserProfileRepository(t)

	ctx := context.Background()
	userId := "123"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "nickname", "created_at", "updated_at"}).
		AddRow(1, "123", "Test", now, now)
	mock.ExpectQuery("SELECT id, user_id, nickname, created_at, updated_at FROM user_profiles WHERE user_id").
		WithArgs(userId).
		WillReturnRows(rows)

	profile, err := profileRepo.GetByUserID(ctx, userId)
	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "123", profile.UserId)
	assert.Equal(t, "Test", profile.Nickname)

	assert.NoError(t, mock.ExpectationsWereMet())
}
