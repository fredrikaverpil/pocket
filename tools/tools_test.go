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
	install     *pocket.FuncDef
	binary      string
	versionArgs []string
}

var tools = []toolTest{
	{"golangci-lint", golangcilint.Install, golangcilint.Name, []string{"version"}},
	{"govulncheck", govulncheck.Install, govulncheck.Name, []string{"-version"}},
	{"uv", uv.Install, uv.Name, []string{"--version"}},
	{"mdformat", mdformat.Install, mdformat.Name, []string{"--version"}},
	{"ruff", ruff.Install, ruff.Name, []string{"--version"}},
	{"mypy", mypy.Install, mypy.Name, []string{"--version"}},
	{"basedpyright", basedpyright.Install, basedpyright.Name, []string{"--version"}},
	{"stylua", stylua.Install, stylua.Name, []string{"--version"}},
	{"bun", bun.Install, bun.Name, []string{"--version"}},
	{"prettier", prettier.Install, prettier.Name, []string{"--version"}},
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
			testFunc := pocket.Func("test:"+tool.name, "test "+tool.name, pocket.Serial(
				tool.install,
				func(fnCtx context.Context) error {
					return pocket.Exec(fnCtx, tool.binary, tool.versionArgs...)
				},
			))

			if err := testFunc.Run(ctx); err != nil {
				t.Fatalf("failed: %v", err)
			}
		})
	}
}
