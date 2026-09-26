package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	pb "github.com/sllt/pi-layout/internal/grpc/user"
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/model"
	"github.com/sllt/pi-layout/internal/repository"
	"github.com/sllt/pi-layout/internal/router"
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/types"
	"github.com/sllt/pi-layout/migrations"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi-layout/pkg/sid"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/apperror"
	"github.com/sllt/pi/pkg/pi/auth"
	piGRPC "github.com/sllt/pi/pkg/pi/grpc"
	"github.com/sllt/pi/pkg/pi/migration"
	"github.com/sllt/pi/pkg/pi/testkit"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type businessFixture struct {
	app      *pi.App
	users    service.UserService
	repo     *repository.Repository
	accounts repository.UserRepository
	profiles repository.UserProfileRepository
	tokens   *jwt.JWT
	client   pb.UserServiceClient
	logger   *log.Logger
}

func newBusinessFixture(t *testing.T) *businessFixture {
	t.Helper()
	a := testkit.New(t, pi.WithConfig(map[string]string{"DB_DIALECT": "sqlite", "DB_NAME": filepath.Join(t.TempDir(), "contract.db"), "DB_MAX_OPEN_CONNECTION": "1", "HTTP_ENABLED": "false", "GRPC_ADDR": "127.0.0.1:0", "GRPC_ENABLED": "true", "USER_GRPC_ENABLED": "true", "JWT_SECRET": "contract-test-key-at-least-thirty-two-bytes", "LOG_LEVEL": "ERROR"}), pi.WithManagedSQL(), pi.WithExplicitHTTPStatus())
	l := log.NewLogger(a.Logger())
	tokens, err := jwt.NewJwt(a)
	require.NoError(t, err)
	r := repository.NewRepository(l, a.Container().SQL)
	accounts := repository.NewUserRepository(r)
	profiles := repository.NewUserProfileRepository(r)
	users := service.NewUserService(service.NewService(r, l, sid.NewSid(), tokens), accounts, profiles)
	require.NoError(t, NewHTTPServer(router.RouterDeps{App: a, Logger: l, JWT: tokens, UserService: users, UserHandler: handler.NewUserHandler(handler.NewHandler(l), users)}))
	a.OnStart(func(ctx *pi.Context) error { _, err := migration.Run(ctx, migrations.All(), ctx.Container); return err })
	require.NoError(t, a.Start(t.Context()))
	conn, err := grpc.NewClient(a.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return &businessFixture{a, users, r, accounts, profiles, tokens, pb.NewUserServiceClient(conn), l}
}
func (f *businessFixture) http(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return testkit.Request(t, f.app, r)
}
func rpcContext(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
}
func seedUser(t *testing.T, f *businessFixture, email string) (string, string) {
	t.Helper()
	require.NoError(t, f.users.Register(t.Context(), &types.RegisterInput{Email: email, Password: "test-password"}))
	u, err := f.accounts.GetByEmail(t.Context(), email)
	require.NoError(t, err)
	token, err := f.tokens.Issue(u.UserId)
	require.NoError(t, err)
	return u.UserId, token
}
func requireKind(t *testing.T, err error, kind apperror.Kind) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, kind, apperror.Resolve(err).Kind())
}
func requireRPCKind(t *testing.T, err error, code codes.Code, kind apperror.Kind) {
	t.Helper()
	require.Equal(t, code, status.Code(err))
	for _, d := range status.Convert(err).Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			require.Equal(t, string(kind), info.Reason)
			return
		}
	}
	t.Fatal("missing typed gRPC error details")
}

func TestBusinessValidationParity(t *testing.T) {
	f := newBusinessFixture(t)
	for _, input := range []*types.RegisterInput{{Email: "bad", Password: "test-password"}, {Email: "a@example.com", Password: "short"}, {Email: "a@example.com", Password: strings.Repeat("界", 25)}} {
		err := f.users.Register(t.Context(), input)
		requireKind(t, err, apperror.InvalidArgument)
		body, _ := json.Marshal(map[string]string{"email": input.Email, "password": input.Password})
		w := f.http(t, "POST", "/api/v1/register", string(body), "")
		require.Equal(t, 400, w.Code)
		require.NotContains(t, w.Body.String(), input.Password)
		_, err = f.client.Register(t.Context(), &pb.RegisterRequest{Email: input.Email, Password: input.Password})
		requireRPCKind(t, err, codes.InvalidArgument, apperror.InvalidArgument)
		require.NotContains(t, status.Convert(err).Message(), input.Password)
	}
	w := f.http(t, "POST", "/api/v1/register", `{"email":"ok@example.com","password":"test-password"}`, "")
	require.Equal(t, 201, w.Code)
	w = f.http(t, "POST", "/api/v1/login", `{"email":"ok@example.com","password":"test-password"}`, "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "accessToken")
	_, err := f.client.Login(t.Context(), &pb.LoginRequest{Email: "ok@example.com", Password: "test-password"})
	require.NoError(t, err)
	_, err = f.client.Register(t.Context(), &pb.RegisterRequest{Email: "ok@example.com", Password: "test-password"})
	requireRPCKind(t, err, codes.AlreadyExists, apperror.Conflict)
	w = f.http(t, "POST", "/api/v1/register", `{"email":"ok@example.com","password":"test-password"}`, "")
	require.Equal(t, 409, w.Code)
	w = f.http(t, "POST", "/api/v1/register", `{"password":"`+strings.Repeat("s", (1<<20)+1)+`"}`, "")
	require.Equal(t, 413, w.Code)
}

type contextStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s contextStream) Context() context.Context { return s.ctx }

func TestBusinessAuthorizationParity(t *testing.T) {
	f := newBusinessFixture(t)
	a, token := seedUser(t, f, "a@example.com")
	b, _ := seedUser(t, f, "b@example.com")
	_, err := f.users.GetProfile(t.Context(), a)
	requireKind(t, err, apperror.Unauthenticated)
	for _, bad := range []string{"", "invalid"} {
		w := f.http(t, "GET", "/api/v1/user", "", bad)
		require.Equal(t, 401, w.Code)
		_, err = f.client.GetProfile(rpcContext(t.Context(), bad), &pb.GetProfileRequest{UserId: a})
		requireRPCKind(t, err, codes.Unauthenticated, apperror.Unauthenticated)
	}
	caller := auth.WithPrincipal(t.Context(), auth.Principal{Subject: a})
	profile, err := f.users.GetProfile(caller, a)
	require.NoError(t, err)
	require.Equal(t, a, profile.UserId)
	_, err = f.users.GetProfile(caller, b)
	requireKind(t, err, apperror.Forbidden)
	w := f.http(t, "GET", "/api/v1/user?userId="+b, "", token)
	require.Equal(t, 403, w.Code)
	_, err = f.client.GetProfile(rpcContext(t.Context(), token), &pb.GetProfileRequest{UserId: b})
	requireRPCKind(t, err, codes.PermissionDenied, apperror.Forbidden)
	got, err := f.client.GetProfile(rpcContext(t.Context(), token), &pb.GetProfileRequest{})
	require.NoError(t, err)
	require.Equal(t, a, got.UserId)
	_, err = f.client.UpdateProfile(rpcContext(t.Context(), token), &pb.UpdateProfileRequest{UserId: b, Email: "b@example.com", Nickname: "stolen"})
	requireRPCKind(t, err, codes.PermissionDenied, apperror.Forbidden)
	_, err = f.client.UpdateProfile(rpcContext(t.Context(), token), &pb.UpdateProfileRequest{Email: "a@example.com", Nickname: "grpc nickname"})
	require.NoError(t, err)
	w = f.http(t, "GET", "/api/v1/user", "", token)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "grpc nickname")
	w = f.http(t, "PUT", "/api/v1/user", `{"email":"a@example.com","nickname":"http nickname"}`, token)
	require.Equal(t, 204, w.Code)
	require.Empty(t, w.Body.String())
	admin := auth.WithPrincipal(t.Context(), auth.Principal{Subject: "trusted-task", Permissions: []string{"users:read:any"}})
	profile, err = f.users.GetProfile(admin, b)
	require.NoError(t, err)
	require.Equal(t, b, profile.UserId)
	err = f.users.UpdateProfile(admin, b, &types.UpdateProfileInput{Email: "b@example.com"})
	requireKind(t, err, apperror.Forbidden)
	verify := func(_ context.Context, encoded string) (auth.Principal, error) {
		claims, err := f.tokens.ParseToken(encoded)
		if err != nil {
			return auth.Principal{}, err
		}
		return auth.Principal{Subject: claims.Subject}, nil
	}
	interceptor := piGRPC.NewPrincipalStreamInterceptor(verify)
	for _, tc := range []struct {
		token, target string
		code          codes.Code
	}{{"", a, codes.Unauthenticated}, {"invalid", a, codes.Unauthenticated}, {token, a, codes.OK}, {token, b, codes.PermissionDenied}} {
		ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+tc.token))
		err := interceptor(nil, contextStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/fixture.Profile/Read"}, func(_ any, s grpc.ServerStream) error {
			_, err := f.users.GetProfile(s.Context(), tc.target)
			return piGRPC.MapError(err)
		})
		require.Equal(t, tc.code, status.Code(err))
	}
}

func TestConcurrentRegistrationMapsUniqueConflict(t *testing.T) {
	f := newBusinessFixture(t)
	var successes atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	start := make(chan struct{})
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := f.users.Register(t.Context(), &types.RegisterInput{Email: "race@example.com", Password: "test-password"})
			if err == nil {
				successes.Add(1)
			} else {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	require.EqualValues(t, 1, successes.Load())
	for err := range errs {
		requireKind(t, err, apperror.Conflict)
	}
}

type pausedAccounts struct {
	repository.UserRepository
	loaded, release chan struct{}
}

func (p *pausedAccounts) GetByID(ctx context.Context, id string) (*model.User, error) {
	u, e := p.UserRepository.GetByID(ctx, id)
	close(p.loaded)
	<-p.release
	return u, e
}

func TestProfileUpdateDoesNotOverwriteConcurrentPassword(t *testing.T) {
	f := newBusinessFixture(t)
	id, _ := seedUser(t, f, "before@example.com")
	paused := &pausedAccounts{UserRepository: f.accounts, loaded: make(chan struct{}), release: make(chan struct{})}
	unblock := sync.OnceFunc(func() { close(paused.release) })
	defer unblock()
	users := service.NewUserService(service.NewService(f.repo, f.logger, sid.NewSid(), f.tokens), paused, f.profiles)
	done := make(chan error, 1)
	go func() {
		done <- users.UpdateProfile(auth.WithPrincipal(t.Context(), auth.Principal{Subject: id}), id, &types.UpdateProfileInput{Email: "after@example.com", Nickname: "updated"})
	}()
	<-paused.loaded
	require.NoError(t, f.accounts.UpdatePassword(t.Context(), id, "new-password-hash"))
	unblock()
	require.NoError(t, <-done)
	u, err := f.accounts.GetByID(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, "new-password-hash", u.Password)
	require.Equal(t, "after@example.com", u.Email)
	_, err = f.app.Container().SQL.ExecContext(t.Context(), "DELETE FROM users WHERE user_id = ?", id)
	require.NoError(t, err)
	err = f.accounts.UpdateEmail(t.Context(), id, "deleted@example.com")
	requireKind(t, err, apperror.NotFound)
}

type pausedProfiles struct {
	repository.UserProfileRepository
	loaded, release chan struct{}
}

func (p *pausedProfiles) GetByUserID(ctx context.Context, id string) (*model.UserProfile, error) {
	profile, err := p.UserProfileRepository.GetByUserID(ctx, id)
	close(p.loaded)
	<-p.release
	return profile, err
}

func TestDeletionDuringProfileUpdateRollsBackEmail(t *testing.T) {
	f := newBusinessFixture(t)
	id, _ := seedUser(t, f, "keep@example.com")
	paused := &pausedProfiles{UserProfileRepository: f.profiles, loaded: make(chan struct{}), release: make(chan struct{})}
	unblock := sync.OnceFunc(func() { close(paused.release) })
	defer unblock()
	users := service.NewUserService(service.NewService(f.repo, f.logger, sid.NewSid(), f.tokens), f.accounts, paused)
	done := make(chan error, 1)
	go func() {
		done <- users.UpdateProfile(auth.WithPrincipal(t.Context(), auth.Principal{Subject: id}), id, &types.UpdateProfileInput{Email: "rollback@example.com", Nickname: "late update"})
	}()
	<-paused.loaded
	_, err := f.app.Container().SQL.ExecContext(t.Context(), "DELETE FROM user_profiles WHERE user_id = ?", id)
	require.NoError(t, err)
	unblock()
	requireKind(t, <-done, apperror.NotFound)
	u, err := f.accounts.GetByID(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, "keep@example.com", u.Email)
}
