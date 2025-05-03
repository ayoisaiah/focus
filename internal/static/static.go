// Package static embeds static files into the binary and copies them to the
// filesystem
package static

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/report"
)

const (
	filesDir = "files"
)

//go:embed files/*
var embeddedFiles embed.FS

func copyEmbeddedFilesToDataDir() error {
	return fs.WalkDir(
		embeddedFiles,
		filesDir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			b, err := embeddedFiles.ReadFile(path)
			if err != nil {
				return err
			}

			stripped := strings.TrimPrefix(
				path,
				filesDir+string(os.PathSeparator),
			)

			relPath := filepath.Join(config.Dir(), stripped)

			destPath, err := xdg.DataFile(relPath)
			if err != nil {
				return err
			}

			// Only write if file does not already exist
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
					return err
				}

				if err := os.WriteFile(destPath, b, 0o644); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func init() {
	err := copyEmbeddedFilesToDataDir()
	if err != nil {
		report.Quit(err)
	}
}
