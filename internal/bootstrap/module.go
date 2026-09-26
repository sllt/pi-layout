package bootstrap

import (
	"github.com/sllt/pi-layout/internal/handler"
	"github.com/sllt/pi-layout/internal/repository"
	"github.com/sllt/pi-layout/internal/service"
	"github.com/sllt/pi-layout/internal/task"
	"github.com/sllt/pi-layout/pkg/jwt"
	"github.com/sllt/pi-layout/pkg/sid"
	"go.uber.org/fx"
)

// CoreModule provides app/container level dependencies shared by all entry points.
var CoreModule = fx.Module("core",
	fx.Provide(
		NewPiApp,
		NewLogger,
		NewDB,
	),
)

// RepositoryModule provides repository-layer dependencies.
var RepositoryModule = fx.Module("repository",
	fx.Provide(
		repository.NewRepository,
		repository.NewTransaction,
		repository.NewUserRepository,
		repository.NewUserProfileRepository,
	),
)

// ServiceModule provides service-layer dependencies used by the HTTP server.
var ServiceModule = fx.Module("service",
	fx.Provide(
		service.NewService,
		service.NewUserService,
	),
)

// HandlerModule provides HTTP handlers.
var HandlerModule = fx.Module("handler",
	fx.Provide(
		handler.NewHandler,
		handler.NewUserHandler,
	),
)

// TaskModule provides task-layer dependencies used by the scheduler entry point.
var TaskModule = fx.Module("task",
	fx.Provide(
		task.NewTask,
		task.NewUserTask,
	),
)

// InfraModule provides cross-cutting utilities.
var InfraModule = fx.Module("infra",
	fx.Provide(
		sid.NewSid,
		jwt.NewJwt,
	),
)
