package pk

import (
	"context"
	"fmt"
	"strings"
)

// BuildAndShowPlan creates a plan and returns a string representation
// showing tasks and their path mappings.
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

		// Show path info if available
		if pathInfo, ok := p.PathMappings[task.Name()]; ok {
			if len(pathInfo.Paths) > 0 || len(pathInfo.ExcludePaths) > 0 {
				sb.WriteString(" →")
				if len(pathInfo.Paths) > 0 {
					sb.WriteString(fmt.Sprintf(" include:%v", pathInfo.Paths))
				}
				if len(pathInfo.ExcludePaths) > 0 {
					sb.WriteString(fmt.Sprintf(" exclude:%v", pathInfo.ExcludePaths))
				}
			}
		} else {
			sb.WriteString(" → (root)")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
