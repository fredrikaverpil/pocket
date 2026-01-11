package tool

import (
	"context"
	"os/exec"
	"sync"

	"github.com/fredrikaverpil/pocket"
)

// Tool represents a tool that can be prepared (installed) and executed.
// It provides a standard Command and Run pattern that all tools share.
//
// Preparation is thread-safe: concurrent calls to Command or Run will only
// trigger one Prepare call, with subsequent calls waiting for and reusing
// the result.
type Tool struct {
	// Name is the binary name (without .exe extension).
	Name string
	// Prepare ensures the tool is installed. It is called before Command.
	Prepare func(ctx context.Context) error

	// once ensures Prepare is only called once per Tool instance.
	once sync.Once
	// prepareErr caches the result of Prepare.
	prepareErr error
}

// Command prepares the tool and returns an exec.Cmd for running it.
// Preparation is thread-safe and only runs once per Tool instance.
func (t *Tool) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	t.once.Do(func() {
		t.prepareErr = t.Prepare(ctx)
	})
	if t.prepareErr != nil {
		return nil, t.prepareErr
	}
	return pocket.Command(ctx, pocket.FromBinDir(pocket.BinaryName(t.Name)), args...), nil
}

// Run prepares and executes the tool.
func (t *Tool) Run(ctx context.Context, args ...string) error {
	cmd, err := t.Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}
