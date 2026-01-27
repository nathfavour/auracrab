package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type SystemSkill struct{}

func (s *SystemSkill) Name() string {
	return "system"
}

func (s *SystemSkill) Description() string {
	return "Perform system operations like security audits and resource checks"
}

func (s *SystemSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["audit", "df", "top"]
				}
			},
			"required": ["action"]
		}
	}`)
}

func (s *SystemSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	switch params.Action {
	case "audit":
		cmd = exec.CommandContext(ctx, "lynis", "audit", "system", "--quick")
	case "df":
		cmd = exec.CommandContext(ctx, "df", "-h")
	case "top":
		cmd = exec.CommandContext(ctx, "top", "-b", "-n", "1")
	default:
		return "", fmt.Errorf("unknown system action: %s", params.Action)
	}

	out, _ := cmd.CombinedOutput()
	return string(out), nil
}
