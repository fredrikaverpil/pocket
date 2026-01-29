package treesitter

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	treesitterCLI "github.com/fredrikaverpil/pocket/tools/treesitter"
	"github.com/fredrikaverpil/pocket/tools/tsqueryls"
)

var (
	queryFormatFlags   = flag.NewFlagSet("query-format", flag.ContinueOnError)
	queryFormatParsers = queryFormatFlags.String("parsers", "", "comma-separated parser names to compile")
	queryLintFlags     = flag.NewFlagSet("query-lint", flag.ContinueOnError)
	queryLintParsers   = queryLintFlags.String("parsers", "", "comma-separated parser names to compile")
	queryLintFix       = queryLintFlags.Bool("fix", false, "auto-fix lint issues")
)

// QueryFormat formats tree-sitter query files using ts_query_ls.
var QueryFormat = pk.NewTask("query-format", "format tree-sitter query files", queryFormatFlags,
	pk.Serial(treesitterCLI.Install, tsqueryls.Install, queryFormatCmd()),
)

// QueryLint lints tree-sitter query files using ts_query_ls.
var QueryLint = pk.NewTask("query-lint", "lint tree-sitter query files", queryLintFlags,
	pk.Serial(treesitterCLI.Install, tsqueryls.Install, queryLintCmd()),
)

func queryFormatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		parsers := parseParsers(*queryFormatParsers)
		parserDir, err := ensureParsers(ctx, parsers)
		if err != nil {
			return err
		}

		dirs := findQueryDirs(ctx)
		if len(dirs) == 0 {
			if pk.Verbose(ctx) {
				pk.Printf(ctx, "  no tree-sitter query directories found\n")
			}
			return nil
		}

		configArgs := tsQueryLsConfigArgs(parserDir)
		for _, dir := range dirs {
			args := []string{"format"}
			args = append(args, configArgs...)
			args = append(args, dir)
			if err := pk.Exec(ctx, tsqueryls.Name, args...); err != nil {
				return err
			}
		}
		return nil
	})
}

func queryLintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		parsers := parseParsers(*queryLintParsers)
		parserDir, err := ensureParsers(ctx, parsers)
		if err != nil {
			return err
		}

		dirs := findQueryDirs(ctx)
		if len(dirs) == 0 {
			if pk.Verbose(ctx) {
				pk.Printf(ctx, "  no tree-sitter query directories found\n")
			}
			return nil
		}

		configArgs := tsQueryLsConfigArgs(parserDir)
		for _, dir := range dirs {
			args := []string{"check"}
			if *queryLintFix {
				args = append(args, "--fix")
			}
			args = append(args, configArgs...)
			args = append(args, dir)
			if err := pk.Exec(ctx, tsqueryls.Name, args...); err != nil {
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

// parseParsers parses a comma-separated list of parser names.
func parseParsers(csv string) []string {
	if csv == "" {
		return nil
	}
	return strings.Split(csv, ",")
}
