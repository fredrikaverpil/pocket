package python

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// FormatOptions configures the py-format task.
type FormatOptions struct {
	RuffConfig string `arg:"ruff-config" usage:"path to ruff config file"`
}

// Format formats Python files using ruff format.
var Format = pocket.Func("py-format", "format Python files", format).
	With(FormatOptions{})

func format(ctx context.Context) error {
	pocket.Serial(ctx, ruff.Install)

	opts := pocket.Options[FormatOptions](ctx)
	configPath := opts.RuffConfig
	if configPath == "" {
		var err error
		configPath, err = pocket.ConfigPath("ruff", ruff.Config)
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}

	args := []string{"format"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, pocket.Path(ctx))

	return pocket.Exec(ctx, ruff.Name, args...)
}
