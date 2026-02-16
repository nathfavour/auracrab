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
					"enum": ["open", "scrape"]
				},
				"url": {
					"type": "string"
				}
			},
			"required": ["action", "url"]
		}
	}`)
}

func (s *BrowserSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action string `json:"action"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	switch params.Action {
	case "open":
		if params.URL == "" {
			return "", fmt.Errorf("missing url")
		}
		return s.open(ctx, params.URL)
	case "scrape":
		if params.URL == "" {
			return "", fmt.Errorf("missing url")
		}
		return s.scrape(ctx, params.URL)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (s *BrowserSkill) open(ctx context.Context, url string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc != nil && bc.IsActive() {
		_, err := bc.Request(ctx, "open "+url)
		if err == nil {
			return fmt.Sprintf("Opened %s in browser extension", url), nil
		}
	}

	var cmd *exec.Cmd
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

func (s *BrowserSkill) scrape(ctx context.Context, url string) (string, error) {
	bc := connect.GetBrowserChannel()
	if bc != nil && bc.IsActive() {
		// If we have a connected browser, we can use it to scrape (even JS-heavy sites)
		// First open the URL if it's not already open or just scrape active tab
		// For now, let's assume we want to scrape the provided URL
		_, _ = bc.Request(ctx, "open "+url)
		// Wait a bit for load? Extensions usually handle this better
		// For now just try to scrape
		return bc.Request(ctx, "scrape")
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

	// Basic truncation for now
	content := string(body)
	if len(content) > 5000 {
		content = content[:5000] + "... [truncated]"
	}

	return content, nil
}
