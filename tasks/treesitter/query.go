package treesitter

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	treesitterCLI "github.com/fredrikaverpil/pocket/tools/treesitter"
	"github.com/fredrikaverpil/pocket/tools/tsqueryls"
)

// Flag names for the QueryFormat and QueryLint tasks.
const (
	FlagParsers = "parsers"
	FlagFix     = "fix"
)

// QueryFormat formats tree-sitter query files using ts_query_ls.
var QueryFormat = &pk.Task{
	Name:  "query-format",
	Usage: "format tree-sitter query files",
	Flags: map[string]pk.FlagDef{
		FlagParsers: {Default: "", Usage: "comma-separated parser names to compile"},
	},
	Body: pk.Serial(treesitterCLI.Install, tsqueryls.Install, queryFormatCmd()),
}

// QueryLint lints tree-sitter query files using ts_query_ls.
var QueryLint = &pk.Task{
	Name:  "query-lint",
	Usage: "lint tree-sitter query files",
	Flags: map[string]pk.FlagDef{
		FlagFix:     {Default: false, Usage: "auto-fix lint issues"},
		FlagParsers: {Default: "", Usage: "comma-separated parser names to compile"},
	},
	Body: pk.Serial(treesitterCLI.Install, tsqueryls.Install, queryLintCmd()),
}

func queryFormatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		parsers := parseParsers(pk.GetFlag[string](ctx, FlagParsers))
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
		parsers := parseParsers(pk.GetFlag[string](ctx, FlagParsers))
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
			if pk.GetFlag[bool](ctx, FlagFix) {
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
