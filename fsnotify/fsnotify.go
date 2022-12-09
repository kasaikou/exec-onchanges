package fsnotify

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/fsnotify.v1"
)

type Event = fsnotify.Event

type Watcher struct {
	lock   sync.Mutex
	closer chan<- *sync.WaitGroup
	Event  <-chan Event
}

func abs(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		return filepath.Join(root, path)
	}
}

func NewWatcher(logger *zap.Logger, rootAbsDir string, preferredRule GlobRuleType, includeGlobRules, excludeGlobRules []string) (*Watcher, error) {
	closer := make(chan *sync.WaitGroup, 1)
	events := make(chan fsnotify.Event)
	notifier, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := &Watcher{
		closer: closer,
		Event:  events,
	}

	go func() {
		logger.Info("search directories as initialize")
		globManager := newGlobRuleManager(rootAbsDir, preferredRule, includeGlobRules, excludeGlobRules)
		addRecursive(rootAbsDir, globManager, notifier)

		for {

		selectBreak:
			select {
			case wg := <-closer:
				defer wg.Done()
				notifier.Close()
				return

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
					events <- event
				}
			}
		}
	}()

	return watcher, nil
}

func (w *Watcher) Stop() (alreadyStopped bool) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.closer == nil {
		return true
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	w.closer <- &wg
	wg.Wait()
	close(w.closer)
	w.closer = nil
	return false
}

func IsRemoveEvent(event fsnotify.Event) bool {
	return event.Op&fsnotify.Remove != 0
}

func addRecursive(abspath string, manager *globRuleManager, watcher *fsnotify.Watcher) error {
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
