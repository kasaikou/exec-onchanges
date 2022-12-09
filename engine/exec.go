package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/streamwest-1629/exec-onchanges/fsnotify"
	"go.uber.org/zap"
)

type ExecOnchangesParam struct {
	RootDir      string
	Command      []string
	IncludeRules []string
	ExcludeRules []string
}

func RouteExecOnchanges(ctx context.Context, logger *zap.Logger, param ExecOnchangesParam) error {
	absRootDir, err := filepath.Abs(param.RootDir)
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher(logger, absRootDir, fsnotify.GlobIncludeRule, param.IncludeRules, param.ExcludeRules)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	parallels := parallelProcesses{}

	for {
	selectBreak:
		select {
		case <-ctx.Done():
			return nil
		case event := <-watcher.Event:
			if parallels.numRunning() > 0 {
				logger.Info("file change detected, but it will be skipped because process is running", zap.String("path", event.Name))
				break selectBreak
			}

			events := map[string]struct{}{event.Name: {}}
			timer := time.NewTimer(time.Second)

		timeoutLoop:
			for {
				select {
				case <-timer.C:
					break timeoutLoop
				case event := <-watcher.Event:
					events[event.Name] = struct{}{}
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}

					timer.Reset(time.Second)
				}
			}

			command := []string{}
			for path := range events {
				func(path string) {
					parallels.add(func() {
						cmd := func() *exec.Cmd {
							command = command[:0]
							for _, arg := range param.Command {
								command = append(command, strings.ReplaceAll(arg, "{{FILEPATH}}", path))
							}
							if len(command) > 1 {
								return exec.Command(command[0], command[1:]...)
							} else {
								return exec.Command(command[0])
							}
						}()

						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr

						logger.Info("execute command", zap.String("command", cmd.String()))
						if err := cmd.Run(); err != nil {
							logger.Error("process error", zap.Error(err))
						}
					})
				}(path)
			}
		}
	}
}

type parallelProcesses struct {
	lock       sync.RWMutex
	numProcess int
}

func (pp *parallelProcesses) add(fn func()) {
	func() {
		pp.lock.Lock()
		defer pp.lock.Unlock()
		pp.numProcess++
	}()

	go func() {
		fn()
		pp.lock.Lock()
		defer pp.lock.Unlock()
		pp.numProcess--
	}()
}

func (pp *parallelProcesses) numRunning() int {
	pp.lock.RLock()
	defer pp.lock.RUnlock()
	return pp.numProcess
}
