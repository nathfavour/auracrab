package watcher

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

type RemoteResource struct {
	URL      string
	Interval time.Duration
	LastETag string
	LastBody string
}

type RemoteWatcher struct {
	resources map[string]*RemoteResource
	mu        sync.RWMutex
	onEvent   func(url string, body string)
}

func NewRemoteWatcher(onEvent func(string, string)) *RemoteWatcher {
	return &RemoteWatcher{
		resources: make(map[string]*RemoteResource),
		onEvent:   onEvent,
	}
}

func (w *RemoteWatcher) Watch(url string, interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.resources[url] = &RemoteResource{
		URL:      url,
		Interval: interval,
	}
}

func (w *RemoteWatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkResources(ctx)
		}
	}
}

func (w *RemoteWatcher) checkResources(ctx context.Context) {
	w.mu.RLock()
	var toCheck []*RemoteResource
	for _, r := range w.resources {
		toCheck = append(toCheck, r)
	}
	w.mu.RUnlock()

	for _, r := range toCheck {
		go func(res *RemoteResource) {
			resp, err := http.Get(res.URL)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			content := string(body)

			if content != res.LastBody {
				res.LastBody = content
				w.onEvent(res.URL, content)
			}
		}(r)
	}
}
