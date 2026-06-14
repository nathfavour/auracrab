package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nathfavour/auracrab/pkg/vault"
)

type XDriver struct {
	ApiKey string
}

func (x *XDriver) Name() string {
	return "x"
}

func (x *XDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to X: %s", content)
	// Placeholder for real API call
	return "https://x.com/status/mock_id", nil
}

func (x *XDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

type LinkedInDriver struct {
	AccessToken string
}

func (l *LinkedInDriver) Name() string {
	return "linkedin"
}

func (l *LinkedInDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to LinkedIn: %s", content)
	return "https://www.linkedin.com/feed/update/mock_id", nil
}

func (l *LinkedInDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

type FacebookDriver struct{}

func (f *FacebookDriver) Name() string { return "facebook" }
func (f *FacebookDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Facebook: %s", content)
	return "https://facebook.com/mock_post", nil
}
func (f *FacebookDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

type InstagramDriver struct{}

func (i *InstagramDriver) Name() string { return "instagram" }
func (i *InstagramDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Instagram: %s", content)
	return "https://instagram.com/mock_post", nil
}
func (i *InstagramDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

type ThreadsDriver struct {
	httpClient *http.Client
}

func (t *ThreadsDriver) Name() string { return "threads" }

func (t *ThreadsDriver) Post(ctx context.Context, content string) (string, error) {
	// 1. Get credentials from env or Vault
	accessToken := os.Getenv("THREADS_ACCESS_TOKEN")
	userID := os.Getenv("THREADS_USER_ID")

	if accessToken == "" || userID == "" {
		v := vault.GetVault()
		if tok, err := v.Get("THREADS_ACCESS_TOKEN"); err == nil && tok != "" {
			accessToken = tok
		}
		if uid, err := v.Get("THREADS_USER_ID"); err == nil && uid != "" {
			userID = uid
		}
	}

	// Fallback to placeholder/mock if credentials are not configured
	if accessToken == "" {
		log.Printf("[Threads] THREADS_ACCESS_TOKEN not set. Running in mock/dry-run mode.")
		return "https://threads.net/mock_post_dryrun", nil
	}

	if userID == "" {
		userID = "me"
	}

	if t.httpClient == nil {
		t.httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	apiURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads", userID)

	reqPayload := map[string]interface{}{
		"media_type":        "TEXT",
		"text":              content,
		"auto_publish_text": true,
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal threads payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("threads api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse response json: %w", err)
	}

	postURL := fmt.Sprintf("https://www.threads.net/post/%s", result.ID)
	log.Printf("[Threads] Successfully published post: %s", postURL)

	return postURL, nil
}

func (t *ThreadsDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

func init() {
	// These would ideally be registered via config
}
