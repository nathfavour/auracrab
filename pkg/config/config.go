package config

import (
	"os"
	"path/filepath"
)

// DataDir returns the path to the auracrab data directory (~/.auracrab)
func DataDir() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".auracrab")
	_ = os.MkdirAll(path, 0755)
	return path
}

// ConfigDir returns the same as DataDir, as requested by the user
func ConfigDir() string {
	return DataDir()
}

// SecretsPath returns the path to the fallback secrets file
func SecretsPath() string {
	return filepath.Join(DataDir(), "secrets.json")
}

// TasksPath returns the path to the tasks persistence file
func TasksPath() string {
	return filepath.Join(DataDir(), "tasks.json")
}

// CrabsDir returns the path to the specialized agents directory
func CrabsDir() string {
	path := filepath.Join(DataDir(), "crabs")
	_ = os.MkdirAll(path, 0755)
	return path
}

// SourceDir returns the path to the source code directory inside data directory
func SourceDir() string {
	path := filepath.Join(DataDir(), "source")
	_ = os.MkdirAll(path, 0755)
	return path
}

// ScreenshotDir returns the path to the screenshots directory in system downloads
func ScreenshotDir() string {
	home, _ := os.UserHomeDir()
	var defaultShotDir string
	if _, err := os.Stat("/data/data/com.termux/files/usr/bin/bash"); err == nil {
		defaultShotDir = filepath.Join(home, "downloads", "auracrab")
	} else {
		defaultShotDir = filepath.Join(home, "Downloads", "auracrab")
	}
	_ = os.MkdirAll(defaultShotDir, 0755)
	return defaultShotDir
}
