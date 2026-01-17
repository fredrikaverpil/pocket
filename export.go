package pocket

import (
	"context"
)

// TaskInfo represents a task for export.
// This is the public type used by the export API for CI/CD integration.
type TaskInfo struct {
	Name   string   `json:"name"`             // CLI command name
	Usage  string   `json:"usage"`            // Description/help text
	Paths  []string `json:"paths,omitempty"`  // Directories this task runs in
	Hidden bool     `json:"hidden,omitempty"` // Whether task is hidden from help
}

// ExportPlan represents the full export structure.
// AutoRun shows the compositional tree structure (serial/parallel).
// ManualRun shows tasks as a flat list.
type ExportPlan struct {
	AutoRun   []*PlanStep `json:"autoRun,omitempty"`
	ManualRun []TaskInfo  `json:"manualRun,omitempty"`
}

// CollectTasks extracts task information from a Runnable tree.
// This uses Engine.Plan() internally to collect tasks without executing them,
// then combines with path mappings from the static tree structure.
// Tasks without RunIn() wrappers get ["."] (root only).
func CollectTasks(r Runnable) []TaskInfo {
	if r == nil {
		return nil
	}

	// Use Engine.Plan() to collect execution plan (this includes hidden tasks)
	engine := NewEngine(r)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		// Fall back to empty on error (shouldn't happen in practice)
		return nil
	}

	// Collect path mappings from static tree structure
	pathMappings := collectPathMappings(r)

	// Extract flattened task list from plan
	return plan.Tasks(pathMappings)
}

// BuildExportPlan builds the export plan structure from config.
// This captures the full tree structure for AutoRun and flat list for ManualRun.
// The caller is responsible for JSON marshaling.
func BuildExportPlan(cfg Config) (ExportPlan, error) {
	export := ExportPlan{}

	// Export AutoRun as tree structure
	if cfg.AutoRun != nil {
		engine := NewEngine(cfg.AutoRun)
		plan, err := engine.Plan(context.Background())
		if err != nil {
			return export, err
		}
		export.AutoRun = plan.Steps()
	}

	// Export ManualRun as flat list
	for _, r := range cfg.ManualRun {
		tasks := CollectTasks(r)
		export.ManualRun = append(export.ManualRun, tasks...)
	}

	return export, nil
}
