package social

import (
	"context"
	"fmt"
	"log"
)

type XPlatform struct {
	ApiKey string
}

func (x *XPlatform) Name() string {
	return "x"
}

func (x *XPlatform) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to X: %s", content)
	// Placeholder for real API call
	return "https://x.com/status/mock_id", nil
}

func (x *XPlatform) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

type LinkedInPlatform struct {
	AccessToken string
}

func (l *LinkedInPlatform) Name() string {
	return "linkedin"
}

func (l *LinkedInPlatform) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to LinkedIn: %s", content)
	return "https://www.linkedin.com/feed/update/mock_id", nil
}

func (l *LinkedInPlatform) GetFeed(ctx context.Context, limit int) ([]Post, error) {
	return []Post{}, nil
}

func init() {
	// These would ideally be registered via config
}
