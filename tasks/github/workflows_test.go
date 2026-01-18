package github

import (
	"path"
	"testing"
)

// TestWorkflowTemplates_EmbedReadFile verifies that all workflow templates
// can be read from the embedded filesystem. This test catches path separator
// issues on Windows - embed.FS requires forward slashes (path.Join), not
// OS-specific separators (filepath.Join).
func TestWorkflowTemplates_EmbedReadFile(t *testing.T) {
	templates := []string{
		"pocket.yml.tmpl",
		"pocket-matrix.yml.tmpl",
		"pr.yml.tmpl",
		"release.yml.tmpl",
		"stale.yml.tmpl",
		"sync.yml.tmpl",
	}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			// Use path.Join (forward slashes) - embed.FS always uses POSIX paths.
			// Using filepath.Join would break on Windows.
			embedPath := path.Join("workflows", tmpl)
			content, err := workflowTemplates.ReadFile(embedPath)
			if err != nil {
				t.Fatalf("ReadFile(%q) failed: %v", embedPath, err)
			}
			if len(content) == 0 {
				t.Errorf("ReadFile(%q) returned empty content", embedPath)
			}
		})
	}
}

func TestDefaultPocketConfig(t *testing.T) {
	cfg := DefaultPocketConfig()
	if cfg.Platforms == "" {
		t.Error("expected non-empty Platforms")
	}
}

func TestDefaultStaleConfig(t *testing.T) {
	cfg := DefaultStaleConfig()
	if cfg.DaysBeforeStale <= 0 {
		t.Errorf("expected positive DaysBeforeStale, got %d", cfg.DaysBeforeStale)
	}
	if cfg.DaysBeforeClose <= 0 {
		t.Errorf("expected positive DaysBeforeClose, got %d", cfg.DaysBeforeClose)
	}
	if cfg.ExemptLabels == "" {
		t.Error("expected non-empty ExemptLabels")
	}
}
