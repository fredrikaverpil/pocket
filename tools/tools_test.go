package tools_test

import (
	"context"
	"os"
	"testing"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/basedpyright"
	"github.com/fredrikaverpil/pocket/tools/bun"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
	"github.com/fredrikaverpil/pocket/tools/mypy"
	"github.com/fredrikaverpil/pocket/tools/prettier"
	"github.com/fredrikaverpil/pocket/tools/ruff"
	"github.com/fredrikaverpil/pocket/tools/stylua"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// toolTest defines a tool to test.
type toolTest struct {
	name        string
	install     *pocket.TaskDef
	binary      string
	versionArgs []string
	// customExec is used for tools that need special execution (e.g., prettier via bun).
	customExec func(ctx context.Context, args ...string) error
}

var tools = []toolTest{
	{"golangci-lint", golangcilint.Install, golangcilint.Name, []string{"version"}, nil},
	{"govulncheck", govulncheck.Install, govulncheck.Name, []string{"-version"}, nil},
	{"uv", uv.Install, uv.Name, []string{"--version"}, nil},
	{"mdformat", mdformat.Install, mdformat.Name, []string{"--version"}, nil},
	{"ruff", ruff.Install, ruff.Name, []string{"--version"}, nil},
	{"mypy", mypy.Install, mypy.Name, []string{"--version"}, nil},
	{"basedpyright", basedpyright.Install, basedpyright.Name, []string{"--version"}, nil},
	{"stylua", stylua.Install, stylua.Name, []string{"--version"}, nil},
	{"bun", bun.Install, bun.Name, []string{"--version"}, nil},
	{"prettier", prettier.Install, prettier.Name, []string{"--version"}, prettier.Exec},
}

func TestTools(t *testing.T) {
	// Create execution context for testing.
	out := pocket.StdOutput()
	out.Stdout = os.Stdout
	out.Stderr = os.Stderr

	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			ctx := pocket.TestContext(out)

			// Create a test function that installs and runs the tool.
			// Compose the install dependency with the version check.
			testFunc := pocket.Task("test:"+tool.name, "test "+tool.name, pocket.Serial(
				tool.install,
				func(fnCtx context.Context) error {
					if tool.customExec != nil {
						return tool.customExec(fnCtx, tool.versionArgs...)
					}
					return pocket.Exec(fnCtx, tool.binary, tool.versionArgs...)
				},
			))

			if err := testFunc.Run(ctx); err != nil {
				t.Fatalf("failed: %v", err)
			}
		})
	}
}
