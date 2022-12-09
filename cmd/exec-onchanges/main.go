package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattn/go-shellwords"
	"github.com/streamwest-1629/exec-onchanges/engine"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	wd, _ := os.Getwd()
	flagConfigFile := ""
	flagExcludeRules := []string{}
	flagIncludeRules := []string{}
	flagCmd := []string{}

	// Read CLI arguments
readBreak:
	for i := 1; i < len(os.Args); i++ {
		switch {
		case os.Args[i] == "-h", os.Args[i] == "--help":
			help()
			os.Exit(-1)
		case os.Args[i] == "--file", os.Args[i] == "-f":
			i++
			flagConfigFile = os.Args[i]
		case strings.HasPrefix(os.Args[i], "--file="), strings.HasPrefix(os.Args[i], "-f="):
			flagConfigFile = unquote(os.Args[i][strings.Index(os.Args[i], "=")+1:])
		case os.Args[i] == "--exclude", os.Args[i] == "-e":
			i++
			flagExcludeRules = append(flagExcludeRules, os.Args[i])
		case strings.HasPrefix(os.Args[i], "--exclude="), strings.HasPrefix(os.Args[i], "-e="):
			flagExcludeRules = append(flagExcludeRules, unquote(os.Args[i][strings.Index(os.Args[i], "=")+1:]))
		case os.Args[i] == "--include", os.Args[i] == "-i":
			i++
			flagIncludeRules = append(flagIncludeRules, os.Args[i])
		case strings.HasPrefix(os.Args[i], "--include="), strings.HasPrefix(os.Args[i], "-i="):
			flagIncludeRules = append(flagIncludeRules, unquote(os.Args[i][strings.Index(os.Args[i], "=")+1:]))
		case os.Args[i] == "--":
			i++
			flagCmd = os.Args[i:]
			break readBreak
		default:
			logger.Fatal(fmt.Sprintf("unknown expression: \"%s\"", os.Args[i]))
		}
	}

	// read config file
	confBody := struct {
		Command  string   `yaml:"command"`
		Includes []string `yaml:"includes"`
		Excludes []string `yaml:"excludes"`
	}{}
	if flagConfigFile != "" {
		func() {
			if b, err := os.ReadFile(flagConfigFile); err != nil {
				logger.Fatal("cannot open config file: " + err.Error())
			} else if err := json.Unmarshal(b, &confBody); err != nil {
				logger.Fatal("failed to parse yaml: " + err.Error())
			}
		}()
	}

	// set parameter
	param := engine.ExecOnchangesParam{}
	if flagConfigFile != "" {
		abspath, _ := filepath.Abs(flagConfigFile)
		param.RootDir = filepath.Dir(abspath)
	} else {
		param.RootDir = wd
	}

	if confBody.Command != "" {
		parsed, err := shellwords.Parse(confBody.Command)
		if err != nil {
			logger.Fatal("cannot parse command: " + err.Error())
		}
		param.Command = parsed
	} else if len(flagCmd) > 0 {
		param.Command = flagCmd
	} else {
		// set default command
		param.Command = []string{"echo", "detected file changed: {{FILEPATH}}"}
	}

	param.IncludeRules = append(flagIncludeRules, confBody.Includes...)
	param.ExcludeRules = append(flagExcludeRules, confBody.Excludes...)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := engine.RouteExecOnchanges(context.Background(), logger, param); err != nil {
			logger.Fatal(err.Error())
		}
	}()

	defer wg.Wait()
}

func unquote(str string) string {
	if len(str) < 2 {
		return str
	}
	if str[0] != str[len(str)-1] {
		return str
	}
	switch str[0] {
	case '\'', '"':
		return str[1 : len(str)-1]
	default:
		return str
	}
}

func help() {

}
