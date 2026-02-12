package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nathfavour/auracrab/pkg/config"
)

// PIDFile returns the path to the daemon PID file
func PIDFile() string {
	return filepath.Join(config.DataDir(), "auracrab.pid")
}

// IsRunning checks if the daemon is already running
func IsRunning() (int, bool) {
	data, err := os.ReadFile(PIDFile())
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}

	// On Unix, FindProcess always succeeds even if the process is dead.
	// We need to send a signal 0 to check if it's actually alive.
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return 0, false
	}

	return pid, true
}

// WritePID writes the current process ID to the PID file
func WritePID() error {
	pid := os.Getpid()
	return os.WriteFile(PIDFile(), []byte(strconv.Itoa(pid)), 0644)
}

// RemovePID removes the PID file
func RemovePID() error {
	return os.Remove(PIDFile())
}

// Terminate sends a SIGTERM to the running daemon
func Terminate() error {
	pid, running := IsRunning()
	if !running {
		return fmt.Errorf("daemon is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Signal(syscall.SIGTERM)
}
