package security

import (
	"context"
	"os/exec"
)

// RunInSandbox executes a command inside a Docker container for safety.
func RunInSandbox(ctx context.Context, command string, image string) (string, error) {
	if image == "" {
		image = "alpine" // Default lightweight image
	}

	// Basic sandbox: no network, limited memory, auto-remove
	args := []string{
		"run", "--rm",
		"--network", "none",
		"--memory", "128m",
		image,
		"sh", "-c", command,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
