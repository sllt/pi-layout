package main

import (
	"github.com/sllt/pi-layout/internal/bootstrap"
	"github.com/sllt/pi-layout/internal/server"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		bootstrap.CoreModule,
		bootstrap.InfraModule,
		bootstrap.RepositoryModule,
		bootstrap.TaskModule,
		fx.Invoke(server.RegisterTaskServer),
	).Run()
}
