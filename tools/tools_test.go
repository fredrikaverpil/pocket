package tools_test

import (
	"context"
	"testing"

	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
	"github.com/fredrikaverpil/pocket/tools/stylua"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var tools = []struct {
	name        string
	prepare     func(context.Context) error
	run         func(context.Context, ...string) error
	versionArgs []string
}{
	{"golangci-lint", golangcilint.Prepare, golangcilint.Run, []string{"version"}},
	{"govulncheck", govulncheck.Prepare, govulncheck.Run, []string{"-version"}},
	{"uv", uv.Prepare, uv.Run, []string{"--version"}},
	{"mdformat", mdformat.Prepare, mdformat.Run, []string{"--version"}},
	{"stylua", stylua.Prepare, stylua.Run, []string{"--version"}},
}

func TestTools(t *testing.T) {
	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			ctx := context.Background()
			if err := tool.prepare(ctx); err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			if err := tool.run(ctx, tool.versionArgs...); err != nil {
				t.Fatalf("Run %v: %v", tool.versionArgs, err)
			}
		})
	}
}
