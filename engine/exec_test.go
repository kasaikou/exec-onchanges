package engine_test

import (
	"context"
	"testing"

	"github.com/streamwest-1629/exec-onchanges/engine"
	"go.uber.org/zap"
)

func TestXxx(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	engine.RouteExecOnchanges(context.Background(), logger, engine.ExecOnchangesParam{
		RootDir:      ".",
		Command:      []string{"echo", "detected file changed: {{FILEPATH}}"},
		IncludeRules: []string{},
		ExcludeRules: []string{".github", ".devcontainer", "**.md"},
	})
}
