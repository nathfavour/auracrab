package skills

import (
"context"
"fmt"
"os/exec"
)

type AutoCommitSkill struct{}

func (s *AutoCommitSkill) Name() string {
	return "autocommit"
}

func (s *AutoCommitSkill) Description() string {
	return "Automatically stages and commits changes with AI-generated messages via autocommiter.go"
}

func (s *AutoCommitSkill) Execute(ctx context.Context, action string, args map[string]interface{}) (string, error) {
	// Action is ignored for now as autocommit is one task
	cmd := exec.CommandContext(ctx, "autocommiter", "-y")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("autocommiter failed: %w", err)
	}
	return string(out), nil
}
