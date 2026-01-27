package skills

import (
"context"
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

func (s *SystemSkill) Execute(ctx context.Context, action string, args map[string]interface{}) (string, error) {
	switch action {
	case "audit":
		cmd := exec.CommandContext(ctx, "lynis", "audit", "system", "--quick")
		out, _ := cmd.CombinedOutput()
		return string(out), nil
	case "df":
		cmd := exec.CommandContext(ctx, "df", "-h")
		out, _ := cmd.CombinedOutput()
		return string(out), nil
	case "top":
		cmd := exec.CommandContext(ctx, "top", "-b", "-n", "1")
		out, _ := cmd.CombinedOutput()
		return string(out), nil
	default:
		return "", fmt.Errorf("unknown system action: %s", action)
	}
}
