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
