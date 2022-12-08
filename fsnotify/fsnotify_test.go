package fsnotify_test

import (
	"testing"

	"github.com/streamwest-1629/exec-onchanges/fsnotify"
	"go.uber.org/zap"
)

func TestFSNotifaction(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	w, _ := fsnotify.NewWatcher(logger, "/workspace", fsnotify.GlobIncludeRule, []string{"**.go"}, []string{".git", "**/.git"})
	defer w.Stop()

	for {
		e := <-w.Event
		logger.Info("fsnotify recieved: " + e.String())
	}
}
