package server

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/sllt/pi-layout/internal/task"
	"github.com/sllt/pi-layout/pkg/log"
	"go.uber.org/fx"
)

func RegisterTaskServer(lc fx.Lifecycle, log *log.Logger, userTask task.UserTask) {
	var (
		scheduler *gocron.Scheduler
		runCtx    context.Context
		cancel    context.CancelFunc
	)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			gocron.SetPanicHandler(func(jobName string, recoverData interface{}) {
				log.Errorf("TaskServer Panic job=%s recover=%v", jobName, recoverData)
			})

			runCtx, cancel = context.WithCancel(context.Background())
			scheduler = gocron.NewScheduler(time.UTC)

			_, err := scheduler.CronWithSeconds("0/3 * * * * *").Do(func() {
				if err := userTask.CheckUser(runCtx); err != nil {
					log.Errorf("CheckUser error: %v", err)
				}
			})
			if err != nil {
				return err
			}

			scheduler.StartAsync()
			return nil
		},
		OnStop: func(context.Context) error {
			if cancel != nil {
				cancel()
			}
			if scheduler != nil {
				scheduler.Stop()
			}
			log.Info("TaskServer stop...")
			return nil
		},
	})
}
