package fsnotify_test

import (
	"context"
	"sync"
	"testing"

	"github.com/streamwest-1629/exec-onchanges/fsnotify"
	"go.uber.org/zap"
)

func TestFSNotifaction(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	globManager, err := fsnotify.NewGlobRuleManager("/workspace", fsnotify.GlobIncludeRule, []string{"**.go"}, []string{".git", "**/.git"})
	if err != nil {
		logger.Fatal("initialize globManager error", zap.Error(err))
	}

	wg := sync.WaitGroup{}
	events := make(chan fsnotify.Event)
	defer close(events)
	ctx, stop := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		wg.Done()
		if err := fsnotify.RouteWatch(ctx, logger, "/workspace", globManager, events); err != nil {
			stop()
		}
	}()
	defer wg.Wait()
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-events:
			logger.Info("fsnotify recieved: " + e.String())
		}
	}
}
