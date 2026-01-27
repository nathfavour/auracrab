package skills

import (
"context"
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

func (s *BrowserSkill) Execute(ctx context.Context, action string, args map[string]interface{}) (string, error) {
	switch action {
	case "open":
		url, ok := args["url"].(string)
		if !ok {
			return "", fmt.Errorf("missing url")
		}
		return s.open(url)
	case "scrape":
		url, ok := args["url"].(string)
		if !ok {
			return "", fmt.Errorf("missing url")
		}
		return s.scrape(url)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
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
