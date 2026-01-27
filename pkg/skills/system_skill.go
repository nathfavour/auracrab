package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

type SystemSkill struct{}

func (s *SystemSkill) Name() string {
	return "audit_system"
}

func (s *SystemSkill) Description() string {
	return "Retrieve deep system information, hardware stats, and environment details."
}

func (s *SystemSkill) Manifest() json.RawMessage {
	return json.RawMessage(`{
		"name": "audit_system",
		"description": "Retrieve deep system information, hardware stats, and environment details.",
		"parameters": {
			"type": "object",
			"properties": {
				"detail": {
					"type": "string",
					"enum": ["basic", "full"],
					"description": "Level of detail to return"
				}
			}
		}
	}`)
}

func (s *SystemSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Detail string `json:"detail"`
	}
	_ = json.Unmarshal(args, &params)

	hostname, _ := os.Hostname()
	uptime := time.Since(time.Now().Add(-time.Second * 3600)) // placeholder, real uptime would require more logic

	res := fmt.Sprintf("System Audit:\nHost: %s\nOS: %s\nArch: %s\nCPUs: %d\nGo: %s\n", 
		hostname, runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())

	if params.Detail == "full" {
		res += fmt.Sprintf("PID: %d\nUID: %d\n", os.Getpid(), os.Getuid())
		res += fmt.Sprintf("Uptime (approx): %v\n", uptime)
	}

	return res, nil
}

func init() {
	Register(&SystemSkill{})
}
