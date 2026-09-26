// Package migrationcmd runs one-shot migrations without starting application servers.
package migrationcmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sllt/pi-layout/migrations"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/logging"
	"github.com/sllt/pi/pkg/pi/migration"
)

type options struct {
	command, configDir string
	target             int64
	timeout, lockTTL   time.Duration
	dryRun, lock       bool
}

type versionSummary struct {
	Version  int64                `json:"version"`
	Name     string               `json:"name"`
	Reason   migration.SkipReason `json:"reason,omitempty"`
	Duration string               `json:"duration"`
	Error    string               `json:"error,omitempty"`
}

type planItem struct {
	Version int64                `json:"version"`
	Name    string               `json:"name"`
	Action  migration.PlanAction `json:"action"`
	Reason  string               `json:"reason"`
}

type summary struct {
	Command      string                `json:"command"`
	OK           bool                  `json:"ok"`
	DryRun       bool                  `json:"dry_run"`
	LockEnabled  bool                  `json:"lock_enabled"`
	StateSource  migration.StateSource `json:"state_source,omitempty"`
	StatePrecise bool                  `json:"state_precise"`
	Applied      []versionSummary      `json:"applied"`
	Skipped      []versionSummary      `json:"skipped"`
	Pending      []versionSummary      `json:"pending"`
	Failed       *versionSummary       `json:"failed,omitempty"`
	Plan         []planItem            `json:"plan"`
	Gaps         []int64               `json:"gaps"`
	Error        string                `json:"error,omitempty"`
}

// Run writes exactly one JSON summary after execution and resource cleanup.
// Help and argument errors do not open resources. Diagnostic logs use stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	o, err := parseOptions(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	s := summary{Command: o.command, DryRun: o.dryRun, LockEnabled: o.command == "up" && o.lock,
		Applied: []versionSummary{}, Skipped: []versionSummary{}, Pending: []versionSummary{}, Plan: []planItem{}, Gaps: []int64{}}
	err = execute(ctx, o, &s, stderr)
	s.OK = err == nil
	if err != nil {
		s.Error = err.Error()
	}
	if writeErr := json.NewEncoder(stdout).Encode(s); writeErr != nil {
		err = errors.Join(err, fmt.Errorf("write migration summary: %w", writeErr))
	}
	return err
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	o := options{command: "up"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.command, args = args[0], args[1:]
	}
	if o.command != "up" && o.command != "plan" && o.command != "status" {
		return o, fmt.Errorf("unknown migration command %q (use up, plan, status)", o.command)
	}
	fs := flag.NewFlagSet("migration "+o.command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.configDir, "config-dir", "configs", "directory containing .env and environment overrides")
	fs.DurationVar(&o.timeout, "timeout", 10*time.Minute, "overall command timeout")
	fs.Int64Var(&o.target, "target", 0, "highest migration version; 0 means all")
	fs.BoolVar(&o.dryRun, "dry-run", false, "up: inspect pending versions without running user migrations")
	fs.BoolVar(&o.lock, "lock", true, "up: acquire a migration lease; explicitly use --lock=false to disable")
	fs.DurationVar(&o.lockTTL, "lock-ttl", 15*time.Minute, "up: lease lifetime (no renewal); must exceed timeout")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 {
		return o, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if o.timeout <= 0 || o.target < 0 {
		return o, fmt.Errorf("timeout must be positive and target must be nonnegative")
	}
	if o.command != "up" {
		var invalid string
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "dry-run" || f.Name == "lock" || f.Name == "lock-ttl" {
				invalid = f.Name
			}
		})
		if invalid != "" {
			return o, fmt.Errorf("--%s only applies to up; plan/status do not acquire an execution lock", invalid)
		}
	} else if o.lock && o.lockTTL <= o.timeout {
		return o, fmt.Errorf("lock-ttl must exceed timeout because the migration lease is not renewed")
	}
	return o, nil
}

func execute(ctx context.Context, o options, s *summary, stderr io.Writer) (err error) {
	cfg, err := loadConfig(o.configDir)
	if err != nil {
		return err
	}
	// The supplied business migrations are SQLite-specific. Other SQL backends
	// are supported by Pi, but require project-specific DDL before enabling here.
	if cfg.Get("DB_DIALECT") != "sqlite" {
		return fmt.Errorf("layout migrations require DB_DIALECT=sqlite; adapt the business DDL before using another dialect")
	}
	logger := logging.NewWriterLogger(logging.INFO, stderr, stderr)
	values := cfg.Values()
	values["HTTP_ENABLED"] = "false"
	values["GRPC_ENABLED"] = "false"
	values["METRICS_ENABLED"] = "false"
	app, err := pi.Build(pi.WithConfig(values), pi.WithLogger(logger), pi.WithManagedSQL())
	if err != nil {
		return err
	}
	if err = app.Start(ctx); err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if closeErr := app.Stop(stopCtx); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration database: %w", closeErr))
		}
	}()
	c := app.Container()
	opts := []migration.Option{migration.WithTarget(o.target)}
	switch o.command {
	case "up":
		if o.lock {
			opts = append(opts, migration.WithLock(), migration.WithLockTTL(o.lockTTL))
		} else {
			opts = append(opts, migration.WithoutLock())
		}
		if o.dryRun {
			opts = append(opts, migration.WithDryRun())
		}
		result, runErr := migration.Run(ctx, migrations.All(), c, opts...)
		s.StateSource, s.StatePrecise = result.StateSource, result.StatePrecise
		s.Applied, s.Skipped = versions(result.Applied), versions(result.Skipped)
		if result.Failed != nil {
			failed := versions([]migration.VersionResult{*result.Failed})
			s.Failed = &failed[0]
		}
		return runErr
	case "plan":
		result, planErr := migration.Plan(ctx, migrations.All(), c, opts...)
		s.StateSource, s.StatePrecise = result.StateSource, result.StatePrecise
		for _, item := range result.Items {
			s.Plan = append(s.Plan, planItem{item.Version, item.Name, item.Action, item.Reason})
			if item.Action == migration.PlanError {
				s.Gaps = append(s.Gaps, item.Version)
			}
		}
		if len(s.Gaps) > 0 {
			planErr = errors.Join(planErr, migration.ErrMigrationGap)
		}
		return planErr
	case "status":
		result, statusErr := migration.Status(ctx, migrations.All(), c, opts...)
		s.StateSource, s.StatePrecise = result.StateSource, result.StatePrecise
		s.Applied, s.Pending = versions(result.Applied), versions(result.Pending)
		s.Gaps = append(s.Gaps, result.Gaps...)
		if len(s.Gaps) > 0 {
			statusErr = errors.Join(statusErr, migration.ErrMigrationGap)
		}
		return statusErr
	}
	return nil
}

func versions(items []migration.VersionResult) []versionSummary {
	result := make([]versionSummary, 0, len(items))
	for _, item := range items {
		v := versionSummary{Version: item.Version, Name: item.Name, Reason: item.Reason, Duration: item.Duration.String()}
		if item.Error != nil {
			v.Error = item.Error.Error()
		}
		result = append(result, v)
	}
	return result
}
