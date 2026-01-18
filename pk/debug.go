package pk

import (
	"context"
	"fmt"
	"sort"
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

	// Sort tasks by name for deterministic output
	sortedTasks := make([]*Task, len(p.Tasks))
	copy(sortedTasks, p.Tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		return sortedTasks[i].Name() < sortedTasks[j].Name()
	})

	for i, task := range sortedTasks {
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
