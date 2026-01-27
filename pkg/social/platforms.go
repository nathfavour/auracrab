package social

import (
	"context"
	"log"
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
func (f *FacebookDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

type InstagramDriver struct{}

func (i *InstagramDriver) Name() string { return "instagram" }
func (i *InstagramDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Instagram: %s", content)
	return "https://instagram.com/mock_post", nil
}
func (i *InstagramDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

type ThreadsDriver struct{}

func (t *ThreadsDriver) Name() string { return "threads" }
func (t *ThreadsDriver) Post(ctx context.Context, content string) (string, error) {
	log.Printf("Posting to Threads: %s", content)
	return "https://threads.net/mock_post", nil
}
func (t *ThreadsDriver) GetFeed(ctx context.Context, limit int) ([]Post, error) { return []Post{}, nil }

func init() {
	// These would ideally be registered via config
}
