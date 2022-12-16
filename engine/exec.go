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

// Parameters for [RouteExecOnchanges].
type ExecOnchangesParam struct {
	RootDir      string   // root directory for monitoring files
	Command      []string // executing command
	IncludeRules []string // including file path rules for execution
	ExcludeRules []string // excluding file or directory path rules for execution
}

// Watches the specified directory and executes commands on file modified.
//
// Note that it is designed to be executed persistently.
func RouteExecOnchanges(ctx context.Context, logger *zap.Logger, param ExecOnchangesParam) error {
	absRootDir, err := filepath.Abs(param.RootDir)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	eventCh := make(chan fsnotify.Event)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fsnotify.RouteWatch(ctx, logger, absRootDir, fsnotify.GlobIncludeRule, param.IncludeRules, param.ExcludeRules, eventCh); err != nil {
			logger.Error("error in route watch", zap.Error(err))
			cancel()
		}
	}()
	defer wg.Wait()

	parallels := parallelProcesses{}

	for {
	selectBreak:
		select {
		case <-ctx.Done():
			return nil
		case event := <-eventCh:
			if parallels.numRunning() > 0 {
				logger.Info("file change detected, but it will be skipped because process is running", zap.String("path", event.Name))
				break selectBreak
			} else if !IsActionEvent(event) {
				break selectBreak
			}

			events := map[string]struct{}{event.Name: {}}
			timer := time.NewTimer(time.Second)

		timeoutLoop:
			for {
			selectBreakInTimer:
				select {
				case <-timer.C:
					break timeoutLoop
				case event := <-eventCh:
					if !IsActionEvent(event) {
						break selectBreakInTimer
					}

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

// Check to see if event requires to execute command.
//
// This rule should be determined regardless
// of whether or not it is included
// in the runtime conditions, and is intended to prevent execution
// on deleted files or directories.
func IsActionEvent(event fsnotify.Event) bool {
	if fsnotify.IsRemoveEvent(event) {
		return false
	}

	info, err := os.Stat(event.Name)
	if err != nil {
		return false
	}
	return !info.IsDir()
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
