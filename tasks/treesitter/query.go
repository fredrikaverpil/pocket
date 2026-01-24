package treesitter

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/tsqueryls"
)

var (
	queryFormatFlags = flag.NewFlagSet("query-format", flag.ContinueOnError)
	queryLintFlags   = flag.NewFlagSet("query-lint", flag.ContinueOnError)
	queryLintFix     = queryLintFlags.Bool("fix", false, "auto-fix lint issues")
)

// QueryFormat formats tree-sitter query files using ts_query_ls.
var QueryFormat = pk.NewTask("query-format", "format tree-sitter query files", queryFormatFlags,
	pk.Serial(tsqueryls.Install, queryFormatCmd()),
)

// QueryLint lints tree-sitter query files using ts_query_ls.
var QueryLint = pk.NewTask("query-lint", "lint tree-sitter query files", queryLintFlags,
	pk.Serial(tsqueryls.Install, queryLintCmd()),
)

func queryFormatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		dirs := findQueryDirs(ctx)
		if len(dirs) == 0 {
			if pk.Verbose(ctx) {
				pk.Printf(ctx, "  no tree-sitter query directories found\n")
			}
			return nil
		}

		for _, dir := range dirs {
			if err := pk.Exec(ctx, tsqueryls.Name, "format", dir); err != nil {
				return err
			}
		}
		return nil
	})
}

func queryLintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		dirs := findQueryDirs(ctx)
		if len(dirs) == 0 {
			if pk.Verbose(ctx) {
				pk.Printf(ctx, "  no tree-sitter query directories found\n")
			}
			return nil
		}

		args := []string{"check"}
		if *queryLintFix {
			args = append(args, "--fix")
		}

		for _, dir := range dirs {
			cmdArgs := append([]string{}, args...)
			cmdArgs = append(cmdArgs, dir)
			if err := pk.Exec(ctx, tsqueryls.Name, cmdArgs...); err != nil {
				return err
			}
		}
		return nil
	})
}

// findQueryDirs finds directories containing tree-sitter query files (.scm).
// Looks in common locations: queries/, lua/*/queries/, etc.
func findQueryDirs(ctx context.Context) []string {
	root := pk.FromGitRoot(pk.PathFromContext(ctx))
	seen := make(map[string]bool)
	var dirs []string

	// Walk the directory looking for .scm files.
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip common directories that shouldn't contain query files.
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".pocket" ||
				name == "vendor" || name == ".tests" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for .scm files.
		if strings.HasSuffix(path, ".scm") {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		return nil
	})

	return dirs
}
