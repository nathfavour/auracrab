package anyisland

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"
)

// SocketPath returns the anyisland UDS path (aligned with anyisland/pkg/ipc).
func SocketPath() string {
	if v := os.Getenv("ANYISLAND_SOCKET"); v != "" {
		return v
	}
	if v := os.Getenv("ANYISLAND_IPC_SOCK"); v != "" {
		return v
	}
	if run := os.Getenv("AGENTIC_RUN_DIR"); run != "" {
		return filepath.Join(run, "anyisland.sock")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".anyisland", "anyisland.sock")
}

// IsManaged checks if the current process is being managed by Anyisland
func IsManaged() bool {
	socketPath := SocketPath()
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return false
	}

	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	req := map[string]string{"op": "HANDSHAKE"}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return false
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return false
	}

	return resp.Status == "MANAGED"
}

// VisualShot sends an ANSI string to the Anyisland visual service to capture a screenshot
func VisualShot(tool string, ansi string) (string, error) {
	conn, err := net.DialTimeout("unix", SocketPath(), 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	req := map[string]interface{}{
		"op":      "VISUAL_SHOT",
		"tool":    tool,
		"payload": ansi,
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return "", err
	}

	var resp struct {
		Status string `json:"status"`
		Path   string `json:"path"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return "", err
	}

	if resp.Status != "SUCCESS" {
		return "", nil
	}

	return resp.Path, nil
}
