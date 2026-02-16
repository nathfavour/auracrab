package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type AutoCommitSkill struct{}

func (s *AutoCommitSkill) Name() string {
	return "autocommit"
}

func (s *AutoCommitSkill) Description() string {
	return "Automatically stages and commits changes with AI-generated messages"
}

func (s *AutoCommitSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {}
		}
	}`)
}

func (s *AutoCommitSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	cmd := exec.CommandContext(ctx, "autocommiter", "-y")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("autocommiter failed: %w", err)
	}
	return string(out), nil
}
