package test

import (
	"testing"

	"github.com/sllt/pi-layout/internal/bootstrap"
	"github.com/sllt/pi-layout/internal/server"
	"go.uber.org/fx"
)

func TestServerDIGraph(t *testing.T) {
	err := fx.ValidateApp(
		bootstrap.CoreModule,
		bootstrap.InfraModule,
		bootstrap.RepositoryModule,
		bootstrap.ServiceModule,
		bootstrap.HandlerModule,
		fx.Invoke(server.NewHTTPServer),
	)
	if err != nil {
		t.Fatalf("server DI graph validation failed: %v", err)
	}
}

func TestMigrationDIGraph(t *testing.T) {
	err := fx.ValidateApp(
		bootstrap.CoreModule,
		fx.Invoke(server.RegisterMigrateServer),
	)
	if err != nil {
		t.Fatalf("migration DI graph validation failed: %v", err)
	}
}

func TestTaskDIGraph(t *testing.T) {
	err := fx.ValidateApp(
		bootstrap.CoreModule,
		bootstrap.InfraModule,
		bootstrap.RepositoryModule,
		bootstrap.TaskModule,
		fx.Invoke(server.RegisterTaskServer),
	)
	if err != nil {
		t.Fatalf("task DI graph validation failed: %v", err)
	}
}
