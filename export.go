package pocket

import (
	"context"
	"encoding/json"
)

// TaskInfo represents a task for export.
// This is the public type used by the export API for CI/CD integration.
type TaskInfo struct {
	Name   string   `json:"name"`             // CLI command name
	Usage  string   `json:"usage"`            // Description/help text
	Paths  []string `json:"paths,omitempty"`  // Directories this task runs in
	Hidden bool     `json:"hidden,omitempty"` // Whether task is hidden from help
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

// ExportJSON exports task information as JSON bytes.
// This is the underlying implementation for the export command.
func ExportJSON(r Runnable) ([]byte, error) {
	tasks := CollectTasks(r)
	return json.MarshalIndent(tasks, "", "  ")
}
