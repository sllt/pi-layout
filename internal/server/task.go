package server

import (
	"time"

	"github.com/go-co-op/gocron"
	"github.com/sllt/pi-layout/internal/task"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
)

// The scheduler is a managed worker: jobs receive runtime cancellation and Stop
// waits for the scheduler before closing SQL, including fatal runtime shutdown.
func RegisterTaskServer(app *pi.App, logger *log.Logger, userTask task.UserTask) {
	app.Go("task-scheduler", func(ctx *pi.Context) error {
		scheduler := gocron.NewScheduler(time.UTC)
		defer scheduler.Stop()
		_, err := scheduler.CronWithSeconds("0/3 * * * * *").Do(func() {
			defer func() {
				if p := recover(); p != nil {
					logger.Errorf("CheckUser panic: %v", p)
				}
			}()
			if err := userTask.CheckUser(ctx.Context); err != nil {
				logger.Errorf("CheckUser error: %v", err)
			}
		})
		if err != nil {
			return err
		}
		scheduler.StartAsync()
		<-ctx.Done()
		return ctx.Err()
	})
}
