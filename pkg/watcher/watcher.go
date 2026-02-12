package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher *fsnotify.Watcher
	onEvent func(string)
}

func NewWatcher(onEvent func(string)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher: w,
		onEvent: onEvent,
	}, nil
}

func (w *Watcher) Start(ctx context.Context, dir string) error {
	// Recursive watch
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		// Debounce timer to avoid multiple heartbeats for simultaneous changes
		var timer *time.Timer
		const debounceDuration = 2 * time.Second

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				
				// Ignore non-write/create/remove/rename events
				if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					continue
				}

				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceDuration, func() {
					w.onEvent(event.Name)
				})

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}
