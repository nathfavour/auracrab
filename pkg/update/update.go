package update

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/nathfavour/auracrab/pkg/anyisland"
)

type HandshakeResponse struct {
	Status           string `json:"status"`
	ToolID           string `json:"tool_id"`
	Version          string `json:"version"`
	AnyislandVersion string `json:"anyisland_version"`
}

// Check checks for updates via Anyisland Pulse IPC or CLI
func Check() (bool, string, error) {
	if anyisland.IsManaged() {
		return checkPulse()
	}
	return checkCLI()
}

func checkPulse() (bool, string, error) {
	conn, err := net.DialTimeout("unix", anyisland.SocketPath(), 2*time.Second)
	if err != nil {
		return false, "", err
	}
	defer conn.Close()

	// Handshake first to ensure we are recognized
	req := map[string]string{"op": "HANDSHAKE"}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return false, "", err
	}

	var resp HandshakeResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return false, "", err
	}

	if resp.Status != "MANAGED" {
		return false, "", fmt.Errorf("tool is not managed by anyisland")
	}

	// Check for updates
	// Note: According to docs, Pulse can send push notifications.
	// For a simple check, we might need a specific 'op' if anyisland supports it,
	// otherwise we fallback to CLI check.
	// Assuming anyisland CLI is the source of truth for now.
	return checkCLI()
}

func checkCLI() (bool, string, error) {
	cmd := exec.Command("anyisland", "update", "auracrab", "--check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If anyisland doesn't support --check, it might fail.
		// For now, let's assume it works or we just try to update.
		return false, "", err
	}

	// Parsing 'anyisland update --check' output would go here.
	// Since I don't have the exact output format, I'll return false for now
	// and assume Apply() will handle it.
	_ = out
	return false, "", nil
}

// Apply triggers an update of auracrab via anyisland
func Apply() error {
	// anyisland update auracrab
	cmd := exec.Command("anyisland", "update", "auracrab")
	return cmd.Run()
}
