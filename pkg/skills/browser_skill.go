package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
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
					"enum": ["open", "scrape", "scrape_headless"]
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
		return s.open(params.URL)
	case "scrape":
		if params.URL == "" {
			return "", fmt.Errorf("missing url")
		}
		return s.scrape(params.URL)
	case "scrape_headless":
		if params.URL == "" {
			return "", fmt.Errorf("missing url")
		}
		return s.scrapeHeadless(params.URL)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (s *BrowserSkill) scrapeHeadless(url string) (string, error) {
	cmd := exec.Command("google-chrome", "--headless", "--disable-gpu", "--dump-dom", "--no-sandbox", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func (s *BrowserSkill) open(url string) (string, error) {
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

func (s *BrowserSkill) scrape(url string) (string, error) {
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
