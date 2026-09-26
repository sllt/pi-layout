package migrationcmd

import (
	"github.com/sllt/pi/pkg/pi/config"
	"os"
)

func loadConfig(dir string) (*config.Snapshot, error) { return config.LoadSnapshot(dir, os.Environ()) }
