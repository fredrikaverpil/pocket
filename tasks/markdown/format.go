package markdown

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

// FormatOptions configures markdown formatting.
type FormatOptions struct {
	Check bool `arg:"check" usage:"check only, don't write"`
}

// Format formats Markdown files using prettier.
var Format = pocket.Func("md-format", "format Markdown files", pocket.Serial(
	prettier.Install,
	format,
)).With(FormatOptions{})

func format(ctx context.Context) error {
	opts := pocket.Options[FormatOptions](ctx)

	args := []string{}
	if opts.Check {
		args = append(args, "--check")
	} else {
		args = append(args, "--write")
	}

	// Add config if available
	if configPath, err := pocket.ConfigPath("prettier", prettier.Config); err == nil && configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Add ignore file if available
	if ignorePath, err := prettier.EnsureIgnoreFile(); err == nil {
		args = append(args, "--ignore-path", ignorePath)
	}

	args = append(args, "**/*.md")

	if err := prettier.Exec(ctx, args...); err != nil {
		return fmt.Errorf("prettier failed: %w", err)
	}
	return nil
}
