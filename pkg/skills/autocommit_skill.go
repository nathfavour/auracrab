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
	return "Analyze staged git changes and generate/commit with an AI message using autocommiter."
}

func (s *AutoCommitSkill) Manifest() json.RawMessage {
	return json.RawMessage(`{
		"name": "autocommit",
		"description": "Analyze staged git changes and generate/commit with an AI message using autocommiter.",
		"parameters": {
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the git repository (default: current directory)"
				},
				"dry_run": {
					"type": "boolean",
					"description": "Only generate the message, do not commit"
				}
			}
		}
	}`)
}

func (s *AutoCommitSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path   string `json:"path"`
		DryRun bool   `json:"dry_run"`
	}
	_ = json.Unmarshal(args, &params)

	cmd := exec.Command("autocommiter")
	if params.Path != "" {
		cmd.Dir = params.Path
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("autocommiter failed: %w\nOutput: %s", err, string(out))
	}

	return string(out), nil
}

func init() {
	Register(&AutoCommitSkill{})
}
