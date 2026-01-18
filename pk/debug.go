package pk

import (
	"context"
	"fmt"
	"strings"
)

// BuildAndShowPlan creates a plan and returns a string representation
// showing tasks and their resolved path mappings.
// This is a temporary debug function for development.
func BuildAndShowPlan(ctx context.Context, root Runnable) string {
	p, err := NewPlan(ctx, root)
	if err != nil {
		return fmt.Sprintf("Error building plan: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d tasks:\n", len(p.Tasks)))

	for i, task := range p.Tasks {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, task.Name()))

		// Show resolved path info if available
		if pathInfo, ok := p.PathMappings[task.Name()]; ok {
			if len(pathInfo.ResolvedPaths) > 0 {
				sb.WriteString(fmt.Sprintf(" → %v", pathInfo.ResolvedPaths))
			}
		} else {
			sb.WriteString(" → [.]")
		}
		sb.WriteString("\n")
	}

	// Show module directories
	if len(p.ModuleDirectories) > 0 {
		sb.WriteString(fmt.Sprintf("\nModule directories (for shim generation): %v\n", p.ModuleDirectories))
	}

	return sb.String()
}
