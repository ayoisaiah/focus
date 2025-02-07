// Package pathutil manages application file paths and locations
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
)

// Paths holds all application path configurations.
type Paths struct {
	configDir      string
	configFileName string
	dbFileName     string
	statusFileName string
	logFileName    string

	// Computed absolute paths
	configFilePath string
	dbFilePath     string
	statusFilePath string
	logFilePath    string
}

var (
	paths *Paths
	once  sync.Once
)

// Initialize must be called once at program startup.
func Initialize() error {
	var initErr error

	once.Do(func() {
		paths = &Paths{
			configDir:      "focus",
			configFileName: "config.yml",
			dbFileName:     "focus.db",
			statusFileName: "status.json",
			logFileName:    "focus.log",
		}

		paths.applyEnvironmentOverrides()
		initErr = paths.computePaths()
	})

	return initErr
}

// Must panics if paths haven't been initialized.
func Must() *Paths {
	if paths == nil {
		panic("pathutil.Initialize() must be called before accessing paths")
	}
	return paths
}

func Dir() string {
	return paths.configDir
}

func DBFilePath() string {
	return paths.dbFilePath
}

func StatusFilePath() string {
	return paths.statusFilePath
}

func LogFilePath() string {
	return paths.logFilePath
}

func (p *Paths) applyEnvironmentOverrides() {
	focusEnv := strings.TrimSpace(os.Getenv("FOCUS_ENV"))
	if focusEnv != "" {
		p.configFileName = fmt.Sprintf("config_%s.yml", focusEnv)
		p.dbFileName = fmt.Sprintf("focus_%s.db", focusEnv)
		p.statusFileName = fmt.Sprintf("status_%s.json", focusEnv)
		p.logFileName = fmt.Sprintf("focus_%s.log", focusEnv)
	}
}

func (p *Paths) computePaths() error {
	var err error

	relPath := filepath.Join(p.configDir, p.configFileName)

	p.configFilePath, err = xdg.ConfigFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	dataDir, err := xdg.DataFile(p.configDir)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	p.dbFilePath = filepath.Join(dataDir, p.dbFileName)

	p.statusFilePath = filepath.Join(dataDir, p.statusFileName)

	p.logFilePath = filepath.Join(dataDir, "log", p.logFileName)

	return nil
}

// StripExtension returns the input file name without its extension.
func StripExtension(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}
