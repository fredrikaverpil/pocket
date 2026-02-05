package treesitter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	treesitterCLI "github.com/fredrikaverpil/pocket/tools/treesitter"
)

// ensureParsers compiles the specified parsers and returns
// the directory containing the compiled parser libraries.
// Returns ("", nil) if no parsers were specified.
func ensureParsers(ctx context.Context, parsers []string) (string, error) {
	if len(parsers) == 0 {
		return "", nil
	}

	dir := pk.FromToolsDir("treesitter-parsers", treesitterCLI.Version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create parser directory: %w", err)
	}

	for _, name := range parsers {
		if err := installParser(ctx, name, dir); err != nil {
			return "", fmt.Errorf("install parser %s: %w", name, err)
		}
	}

	return dir, nil
}

// installParser compiles a single tree-sitter parser from source.
func installParser(ctx context.Context, name, dir string) error {
	// Skip if already compiled.
	outFile := filepath.Join(dir, name+parserExt())
	if _, err := os.Stat(outFile); err == nil {
		return nil
	}

	// Clone the parser repository into a temp dir.
	tmpDir, err := os.MkdirTemp("", "tree-sitter-"+name+"-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoURL := fmt.Sprintf("https://github.com/tree-sitter/tree-sitter-%s", name)
	pk.Printf(ctx, "  Cloning %s\n", repoURL)
	if err := pk.Exec(ctx, "git", "clone", "--depth", "1", "--quiet", repoURL, tmpDir); err != nil {
		return fmt.Errorf("clone %s: %w", repoURL, err)
	}

	// Build the parser shared library.
	pk.Printf(ctx, "  Building tree-sitter-%s parser\n", name)
	if err := pk.Exec(ctx, "tree-sitter", "build", "-o", outFile, tmpDir); err != nil {
		return fmt.Errorf("build %s: %w", name, err)
	}

	return nil
}

// parserExt returns the platform-specific shared library extension.
func parserExt() string {
	switch pk.HostOS() {
	case pk.Darwin:
		return ".dylib"
	case pk.Windows:
		return ".dll"
	default:
		return ".so"
	}
}

// tsQueryLsConfigArgs returns the --config arguments for ts_query_ls
// to locate compiled parsers. Returns nil if parserDir is empty.
func tsQueryLsConfigArgs(parserDir string) []string {
	if parserDir == "" {
		return nil
	}
	config := fmt.Sprintf(`{"parser_install_directories":[%q]}`, parserDir)
	return []string{"--config", config}
}
