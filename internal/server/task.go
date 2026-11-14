package server

import (
	"time"

	"github.com/sllt/pi-layout/internal/task"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
)

// RegisterTaskServer registers scheduled tasks under Pi's supervised cron runtime.
// Compared to the prior gocron v1.37.0 wrapper:
//   - No goroutine starts during Fx construction; scheduling starts in App.Start;
//   - Schedule ("0/3 * * * * *") and timezone (time.UTC) match gocron's CronWithSeconds;
//   - Overlap uses CronSkipIfRunning so slow CheckUser runs cannot pile up on SQLite;
//   - App.Stop cancels in-flight task contexts and waits for task exit before closing Owned SQL.
func RegisterTaskServer(app *pi.App, logger *log.Logger, userTask task.UserTask) error {
	return app.AddCron("0/3 * * * * *", "check-user", func(ctx *pi.Context) error {
		if err := userTask.CheckUser(ctx.Context); err != nil {
			logger.Errorf("CheckUser error: %v", err)
			return err
		}
		return nil
	}, pi.CronJobOptions{
		Overlap:  pi.CronSkipIfRunning,
		Location: time.UTC,
	})
}
