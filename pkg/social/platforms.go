package social

import (
	"context"
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

type FacebookPlatform struct{}

func (f *FacebookPlatform) Name() string { return "facebook" }
func (f *FacebookPlatform) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Facebook: %s", content)
	return "https://facebook.com/mock_post", nil
}
func (f *FacebookPlatform) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

type InstagramPlatform struct{}

func (i *InstagramPlatform) Name() string { return "instagram" }
func (i *InstagramPlatform) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Instagram: %s", content)
	return "https://instagram.com/mock_post", nil
}
func (i *InstagramPlatform) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

type ThreadsPlatform struct{}

func (t *ThreadsPlatform) Name() string { return "threads" }
func (t *ThreadsPlatform) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Threads: %s", content)
	return "https://threads.net/mock_post", nil
}
func (t *ThreadsPlatform) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

func init() {
	// These would ideally be registered via config
}
