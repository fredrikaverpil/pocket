package pk

import (
	"context"
	"io"
	"testing"
)

func TestGitDiffTask_Disabled(t *testing.T) {
	ctx := context.Background()
	ctx = contextWithGitDiffEnabled(ctx, false)
	ctx = context.WithValue(ctx, outputKey{}, &Output{Stdout: io.Discard, Stderr: io.Discard})

	// Should return nil immediately when git diff is disabled
	if err := gitDiffTask.run(ctx); err != nil {
		t.Errorf("gitDiffTask.run() with disabled flag returned error: %v", err)
	}
}

func TestGitDiffEnabledFromContext_Default(t *testing.T) {
	ctx := context.Background()

	// Default should be false (git diff disabled)
	if gitDiffEnabledFromContext(ctx) {
		t.Error("expected gitDiffEnabled to be false by default")
	}
}

func TestGitDiffEnabledFromContext_Enabled(t *testing.T) {
	ctx := context.Background()
	ctx = contextWithGitDiffEnabled(ctx, true)

	if !gitDiffEnabledFromContext(ctx) {
		t.Error("expected gitDiffEnabled to be true after setting")
	}
}
