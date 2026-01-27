package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type BrowserSkill struct{}

func (s *BrowserSkill) Name() string {
	return "browser_action"
}

func (s *BrowserSkill) Description() string {
	return "Open a URL or perform web automation using the system browser."
}

func (s *BrowserSkill) Manifest() json.RawMessage {
	return json.RawMessage(`{
		"name": "browser_action",
		"description": "Open a URL or perform web automation using the system browser.",
		"parameters": {
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The URL to visit"
				},
				"action": {
					"type": "string",
					"enum": ["open", "screenshot", "scrape"],
					"description": "Action to perform"
				}
			},
			"required": ["url", "action"]
		}
	}`)
}

func (s *BrowserSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		URL    string `json:"url"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	// For now, implement 'open' as a simple platform-specific command
	// and 'scrape' using curl/lynx. PARITY with moltbot's Playwright will be 
	// handled by a future specialized crab using these primitives.
	switch params.Action {
	case "open":
		// Simple xdg-open on Linux
		cmd := exec.Command("xdg-open", params.URL)
		err := cmd.Start()
		if err != nil {
			return "", fmt.Errorf("failed to open browser: %v", err)
		}
		return fmt.Sprintf("Opened %s in system browser.", params.URL), nil
	case "scrape":
		cmd := exec.Command("curl", "-sL", params.URL)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("scrape failed: %v", err)
		}
		// Return snippet
		res := string(out)
		if len(res) > 2000 {
			res = res[:2000] + "... [truncated]"
		}
		return res, nil
	default:
		return "", fmt.Errorf("action %s not implemented natively yet", params.Action)
	}
}

func init() {
	Register(&BrowserSkill{})
}
