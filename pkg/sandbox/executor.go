package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nathfavour/auracrab/pkg/security"
)

// DefaultExecutor runs verification inside an isolated sandbox.
type DefaultExecutor struct{}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e *DefaultExecutor) Execute(ctx context.Context, req VerifyRequest) (*VerifyResult, error) {
	if req.WorkDir == "" {
		return nil, fmt.Errorf("workdir is required")
	}

	workDir, err := filepath.Abs(req.WorkDir)
	if err != nil {
		return nil, err
	}

	command := req.Command
	if command == "" {
		command = "go test ./..."
	}

	if hasDocker(ctx) {
		out, err := security.RunInSandbox(ctx, fmt.Sprintf("cd /work && %s", command), req.Image)
		return &VerifyResult{
			Success:  err == nil,
			ExitCode: exitCode(err),
			Output:   out,
		}, nil
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	out, runErr := cmd.CombinedOutput()
	return &VerifyResult{
		Success:  runErr == nil,
		ExitCode: exitCode(runErr),
		Output:   strings.TrimSpace(string(out)),
	}, nil
}

func hasDocker(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "version").Run() == nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}
