package fsnotify

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
)

// Types for file path glob rule (include/exclude).
type GlobRuleType bool

// Types result of checking path with glob rules in [GlobRuleManager].
type GlobRuleResult int

const (
	GlobIncludeRule GlobRuleType = false
	GlobExcludeRule GlobRuleType = true
)

const (
	GlobRuleDefault GlobRuleResult = iota
	GlobRuleInclude
	GlobRuleExclude
)

type GlobRuleManager struct {
	rootDir       string
	prefferedRule GlobRuleType
	includeRules  []glob.Glob
	excludeRules  []glob.Glob
}

func newGlobRuleManager(rootDir string, prefferedRule GlobRuleType, includeGlobRules, excludeGlobRules []string) (*GlobRuleManager, error) {
	manager := &GlobRuleManager{
		rootDir:       rootDir,
		prefferedRule: prefferedRule,
		includeRules:  []glob.Glob{},
		excludeRules:  []glob.Glob{},
	}

	for _, rule := range includeGlobRules {
		glob, err := compileRule(rootDir, rule)
		if err != nil {
			return nil, err
		}
		manager.includeRules = append(manager.includeRules, glob)
	}
	for _, rule := range excludeGlobRules {
		glob, err := compileRule(rootDir, rule)
		if err != nil {
			return nil, err
		}
		manager.excludeRules = append(manager.excludeRules, glob)
	}

	return manager, nil
}

func compileRule(rootDir, rule string) (glob.Glob, error) {
	toslash := filepath.ToSlash(rule)
	if strings.Contains(toslash, "/../") || strings.HasPrefix(toslash, "../") {
		return nil, fmt.Errorf("found '/../' in path rule")
	}

	if strings.HasPrefix(toslash, "./") {
		return glob.Compile(filepath.Join(rootDir, filepath.FromSlash(rule)), filepath.Separator)
	} else if filepath.IsAbs(rule) || strings.HasPrefix(toslash, "**/") {
		return glob.Compile(rule, filepath.Separator)
	} else {
		return glob.Compile(filepath.Join("**", filepath.FromSlash(rule)), filepath.Separator)
	}
}

// Check if a file path meets the include/exclude conditions.
// If it does not satisfy either conditions, will return [GlobRuleDefault].
func (m *GlobRuleManager) IsInclude(path string) (GlobRuleResult, error) {

	abspath := func() string {
		if filepath.IsAbs(path) {
			return path
		} else {
			return filepath.Join(m.rootDir, path)
		}
	}()

	switch m.prefferedRule {
	case GlobIncludeRule:
		for _, glob := range m.includeRules {
			if glob.Match(abspath) {
				return GlobRuleInclude, nil
			}
		}
		for _, glob := range m.excludeRules {
			if glob.Match(abspath) {
				return GlobRuleExclude, nil
			}
		}
	case GlobExcludeRule:
		for _, glob := range m.excludeRules {
			if glob.Match(abspath) {
				return GlobRuleExclude, nil
			}
		}
		for _, glob := range m.includeRules {
			if glob.Match(abspath) {
				return GlobRuleInclude, nil
			}
		}
	}
	return GlobRuleDefault, nil
}
