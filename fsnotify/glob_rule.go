package fsnotify

import (
	"fmt"
	"path/filepath"

	"github.com/gobwas/glob"
)

type GlobRuleType bool

type GlobRuleCmd int

type GlobRuleResult int

const (
	GlobRuleAdd GlobRuleCmd = iota
	GlobRuleDelete
	GlobIncludeRule GlobRuleType = false
	GlobExcludeRule GlobRuleType = true
)

const (
	GlobRuleDefault GlobRuleResult = iota
	GlobRuleInclude
	GlobRuleExclude
)

type globRuleManager struct {
	rootDir       string
	prefferedRule GlobRuleType
	includeRules  []glob.Glob
	excludeRules  []glob.Glob
}

func newGlobRuleManager(rootDir string, prefferedRule GlobRuleType, includeGlobRules, excludeGlobRules []string) *globRuleManager {
	manager := &globRuleManager{
		rootDir:       rootDir,
		prefferedRule: prefferedRule,
		includeRules:  []glob.Glob{},
		excludeRules:  []glob.Glob{},
	}

	for _, rule := range includeGlobRules {
		manager.includeRules = append(manager.includeRules, glob.MustCompile(rule))
	}
	for _, rule := range excludeGlobRules {
		manager.excludeRules = append(manager.excludeRules, glob.MustCompile(rule))
	}

	return manager
}

func (m *globRuleManager) IsInclude(path string) (GlobRuleResult, error) {

	relpath, err := func() (string, error) {
		if filepath.IsAbs(path) {
			return filepath.Rel(m.rootDir, path)
		} else {
			return path, nil
		}
	}()
	if err != nil {
		return GlobRuleDefault, fmt.Errorf("cannot convert to relative path: %s: %w", path, err)
	}

	relpath = filepath.ToSlash(relpath)
	switch m.prefferedRule {
	case GlobIncludeRule:
		for _, glob := range m.includeRules {
			if glob.Match(relpath) {
				return GlobRuleInclude, nil
			}
		}
		for _, glob := range m.excludeRules {
			if glob.Match(relpath) {
				return GlobRuleExclude, nil
			}
		}
	case GlobExcludeRule:
		for _, glob := range m.excludeRules {
			if glob.Match(relpath) {
				return GlobRuleExclude, nil
			}
		}
		for _, glob := range m.includeRules {
			if glob.Match(relpath) {
				return GlobRuleInclude, nil
			}
		}
	}
	return GlobRuleDefault, nil
}
