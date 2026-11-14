package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sllt/pi-layout/internal/bootstrap"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "openapi: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	outDir := fs.String("out", "docs", "directory for generated openapi.json and openapi.yaml")
	check := fs.Bool("check", false, "verify existing OpenAPI files match generated output without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}

	app, err := bootstrap.BuildOpenAPIApp()
	if err != nil {
		return fmt.Errorf("build openapi app: %w", err)
	}

	jsonBytes, err := app.OpenAPIJSON()
	if err != nil {
		return fmt.Errorf("generate openapi.json: %w", err)
	}
	yamlBytes, err := app.OpenAPIYAML()
	if err != nil {
		return fmt.Errorf("generate openapi.yaml: %w", err)
	}

	outputs := []struct {
		name string
		data []byte
	}{
		{name: "openapi.json", data: ensureTrailingNewline(jsonBytes)},
		{name: "openapi.yaml", data: ensureTrailingNewline(yamlBytes)},
	}

	if *check {
		for _, out := range outputs {
			target := filepath.Join(*outDir, out.name)
			existing, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("check %s: %w", target, err)
			}
			if !bytes.Equal(existing, out.data) {
				return fmt.Errorf("openapi spec drift detected in %s (regenerate with `go run ./cmd/openapi --out %s`)", target, *outDir)
			}
		}
		fmt.Fprintf(stdout, "OpenAPI specs up to date in %s\n", *outDir)
		return nil
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", *outDir, err)
	}

	for _, out := range outputs {
		target := filepath.Join(*outDir, out.name)
		if err := writeFileAtomically(target, out.data); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", target)
	}

	return nil
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\n' {
		return data
	}
	out := make([]byte, len(data)+1)
	copy(out, data)
	out[len(out)-1] = '\n'
	return out
}

func writeFileAtomically(target string, data []byte) error {
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", target, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", target, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("rename %s: %w", target, err)
	}
	cleanup = false
	return nil
}
