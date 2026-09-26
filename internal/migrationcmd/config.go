package migrationcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type configValues map[string]string

func (c configValues) Get(key string) string { return c[key] }
func (c configValues) GetOrDefault(key, fallback string) string {
	if value := c[key]; value != "" {
		return value
	}
	return fallback
}

// Read files without modifying the process environment or calling log.Fatal.
// Precedence matches the app: OS > .APP_ENV.env (or .local.env) > .env.
func loadConfig(dir string) (configValues, error) {
	values := configValues{}
	read := func(name string) error {
		path := filepath.Join(dir, name)
		data, err := godotenv.Read(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read migration config %s: %w", path, err)
		}
		for k, v := range data {
			values[k] = v
		}
		return nil
	}
	if err := read(".env"); err != nil {
		return nil, err
	}
	env, exists := os.LookupEnv("APP_ENV")
	if !exists {
		env = values["APP_ENV"]
	}
	override := ".local.env"
	if env != "" {
		if strings.ContainsAny(env, "/\\") || env == ".." {
			return nil, fmt.Errorf("invalid APP_ENV")
		}
		override = "." + env + ".env"
	}
	if err := read(override); err != nil {
		return nil, err
	}
	for _, entry := range os.Environ() {
		k, v, ok := strings.Cut(entry, "=")
		if ok {
			values[k] = v
		}
	}
	return values, nil
}
