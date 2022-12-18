package fsnotify

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gopkg.in/fsnotify.v1"
)

type Event = fsnotify.Event

func abs(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		return filepath.Join(root, path)
	}
}

// Watches the specified directory or files changed.
//
// Not that it is designed to be executed persistently.
func RouteWatch(ctx context.Context, logger *zap.Logger, rootAbsDir string, globManager FilepathChecker, eventReciever chan<- Event) error {

	notifier, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer notifier.Close()

	logger.Info("search directories as initialize")
	addRecursive(rootAbsDir, globManager, notifier)

	for {
	selectBreak:
		select {
		case <-ctx.Done():
			return nil

		case event := <-notifier.Events:

			include, err := globManager.IsInclude(event.Name)
			if err != nil {
				logger.Error("error in checking", zap.Error(err))
				break selectBreak
			} else if include == GlobRuleExclude {
				break selectBreak
			}

			abspath := abs(rootAbsDir, event.Name)
			if s, err := os.Stat(abspath); err == nil && s != nil && s.IsDir() {
				if event.Op&fsnotify.Create != 0 {
					if err := addRecursive(abspath, globManager, notifier); err != nil {
						logger.Error("cannot add fsnotify monitoring", zap.Error(err), zap.String("path", abspath))
					}
				}
			}
			if event.Op&fsnotify.Remove != 0 {
				notifier.Remove(event.Name)
			}

			if include == GlobRuleInclude {
				eventReciever <- event
			}
		}
	}
}

// Check if event is a deletion event using the [fsnotify.Remove] flag.
func IsRemoveEvent(event fsnotify.Event) bool {
	return event.Op&fsnotify.Remove != 0
}

func addRecursive(abspath string, manager FilepathChecker, watcher *fsnotify.Watcher) error {
	ignoredDirs := map[string]struct{}{}

	return filepath.WalkDir(abspath, func(path string, d fs.DirEntry, err error) error {
		absItempath := abs(abspath, path)

		isdir := d.IsDir()
		if err != nil {
			if included, err := manager.IsInclude(absItempath); err == nil {
				if isdir {
					if _, exist := ignoredDirs[filepath.Dir(absItempath)]; exist || included == GlobRuleExclude {
						ignoredDirs[absItempath] = struct{}{}
						return nil
					}
				} else {
					if included == GlobRuleDefault || included == GlobRuleExclude {
						return nil
					}
				}
			}
			return err
		}
		if isdir {
			included, err := manager.IsInclude(absItempath)
			if err != nil {
				return err
			} else if _, exist := ignoredDirs[filepath.Dir(absItempath)]; exist || included == GlobRuleExclude {
				ignoredDirs[absItempath] = struct{}{}
			} else {
				if err := watcher.Add(absItempath); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
