package pocket

import (
	"context"
	"os"
	"os/exec"
)

// Command creates an exec.Cmd with the working directory set to the git root
// and PATH prepended with the .pocket/bin directory.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = GitRoot()
	cmd.Env = PrependPath(os.Environ(), FromBinDir())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// PrependPath prepends a directory to the PATH in the given environment.
func PrependPath(env []string, dir string) []string {
	result := make([]string, 0, len(env)+1)
	pathSet := false
	for _, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			result = append(result, "PATH="+dir+string(os.PathListSeparator)+e[5:])
			pathSet = true
		} else {
			result = append(result, e)
		}
	}
	if !pathSet {
		result = append(result, "PATH="+dir)
	}
	return result
}
