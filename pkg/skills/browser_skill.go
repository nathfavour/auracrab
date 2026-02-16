package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/nathfavour/auracrab/pkg/connect"
)

type BrowserSkill struct{}

func (s *BrowserSkill) Name() string {
	return "browser"
}

func (s *BrowserSkill) Description() string {
	return "Open websites and scrape content"
}

func (s *BrowserSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["open", "scrape", "click", "type", "hover", "wait", "screenshot"]
				},
				"url": {
					"type": "string"
				},
				"selector": {
					"type": "string"
				},
				"text": {
					"type": "string"
				},
				"condition": {
					"type": "string",
					"description": "For 'wait' action: can be a timeout in ms or 'selector:.css-selector'"
				},
				"context": {
					"type": "string",
					"description": "Optional keyword to identify the right browser profile/window (e.g., 'twitter', 'gmail')"
				}
			},
			"required": ["action"]
		}
	}`)
}

func (s *BrowserSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action    string `json:"action"`
		URL       string `json:"url"`
		Selector  string `json:"selector"`
		Text      string `json:"text"`
		Condition string `json:"condition"`
		Context   string `json:"context"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	switch params.Action {
	case "open":
		if params.URL == "" {
			return "", fmt.Errorf("missing url")
		}
		return s.open(ctx, params.URL, params.Context)
	case "scrape":
		return s.scrape(ctx, params.URL, params.Context)
	case "click":
		if params.Selector == "" {
			return "", fmt.Errorf("missing selector")
		}
		return s.click(ctx, params.Selector, params.Context)
	case "type":
		if params.Selector == "" || params.Text == "" {
			return "", fmt.Errorf("missing selector or text")
		}
		return s.typeText(ctx, params.Selector, params.Text, params.Context)
	case "hover":
		if params.Selector == "" {
			return "", fmt.Errorf("missing selector")
		}
		return s.hover(ctx, params.Selector, params.Context)
	case "wait":
		cond := params.Condition
		if cond == "" {
			cond = params.Text // Fallback to text
		}
		return s.wait(ctx, cond, params.Context)
	case "screenshot":
		return s.screenshot(ctx, params.Context)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (s *BrowserSkill) screenshot(ctx context.Context, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}
	targetInstance := ""
	if contextStr != "" {
		client := bc.FindClientByTab(contextStr)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}
	return bc.Request(ctx, targetInstance, "screenshot")
}

func (s *BrowserSkill) hover(ctx context.Context, selector string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}
	targetInstance := ""
	if contextStr != "" {
		client := bc.FindClientByTab(contextStr)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}
	return bc.Request(ctx, targetInstance, "hover "+selector)
}

func (s *BrowserSkill) wait(ctx context.Context, condition string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}
	targetInstance := ""
	if contextStr != "" {
		client := bc.FindClientByTab(contextStr)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}
	return bc.Request(ctx, targetInstance, "wait "+condition)
}

func (s *BrowserSkill) open(ctx context.Context, url string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc != nil && bc.IsActive() {
		targetInstance := ""
		if contextStr != "" {
			client := bc.FindClientByTab(contextStr)
			if client != nil {
				targetInstance = client.InstanceID
			}
		}
		_, err := bc.Request(ctx, targetInstance, "open "+url)
		if err == nil {
			return fmt.Sprintf("Opened %s in browser extension (context: %s)", url, contextStr), nil
		}
	}

	var cmd *exec.Cmd
	// ... existing local open logic ...
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return "", fmt.Errorf("unsupported platform")
	}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Opened %s", url), nil
}

func (s *BrowserSkill) scrape(ctx context.Context, url string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc != nil && bc.IsActive() {
		targetInstance := ""
		if contextStr != "" {
			client := bc.FindClientByTab(contextStr)
			if client != nil {
				targetInstance = client.InstanceID
			}
		}
		if url != "" {
			_, _ = bc.Request(ctx, targetInstance, "open "+url)
		}
		return bc.Request(ctx, targetInstance, "scrape")
	}

	if url == "" {
		return "", fmt.Errorf("missing url for non-extension scraping")
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content := string(body)
	if len(content) > 5000 {
		content = content[:5000] + "... [truncated]"
	}

	return content, nil
}

func (s *BrowserSkill) click(ctx context.Context, selector string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}
	targetInstance := ""
	if contextStr != "" {
		client := bc.FindClientByTab(contextStr)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}
	return bc.Request(ctx, targetInstance, "click "+selector)
}

func (s *BrowserSkill) typeText(ctx context.Context, selector, text string, contextStr string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}
	targetInstance := ""
	if contextStr != "" {
		client := bc.FindClientByTab(contextStr)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}
	return bc.Request(ctx, targetInstance, fmt.Sprintf("type %s %s", selector, text))
}
