package pocket

import (
	"context"
)

// TaskInfo represents a task for introspection.
// This is the public type used by the introspection API for CI/CD integration.
type TaskInfo struct {
	Name   string   `json:"name"`             // CLI command name
	Usage  string   `json:"usage"`            // Description/help text
	Paths  []string `json:"paths,omitempty"`  // Directories this task runs in
	Hidden bool     `json:"hidden,omitempty"` // Whether task is hidden from help
}

// IntrospectPlan represents the full introspection structure.
// AutoRun shows the compositional tree structure (serial/parallel).
// ManualRun shows tasks as a flat list.
type IntrospectPlan struct {
	AutoRun   []*PlanStep `json:"autoRun,omitempty"`
	ManualRun []TaskInfo  `json:"manualRun,omitempty"`
}

// CollectTasks extracts task information from a Runnable tree.
// This uses Engine.Plan() internally to collect tasks without executing them,
// then combines with path mappings from the static tree structure.
// Tasks without RunIn() wrappers get ["."] (root only).
func CollectTasks(r Runnable) ([]TaskInfo, error) {
	if r == nil {
		return nil, nil
	}

	// Use Engine.Plan() to collect execution plan (this includes hidden tasks)
	engine := NewEngine(r)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		return nil, err
	}

	// Collect path mappings from static tree structure
	pathMappings := collectPathMappings(r)

	// Extract flattened task list from plan
	return plan.Tasks(pathMappings), nil
}

// BuildIntrospectPlan builds the introspection plan structure from config.
// This captures the full tree structure for AutoRun and flat list for ManualRun.
// The caller is responsible for JSON marshaling.
func BuildIntrospectPlan(cfg Config) (IntrospectPlan, error) {
	result := IntrospectPlan{}

	// Collect AutoRun as tree structure
	if cfg.AutoRun != nil {
		engine := NewEngine(cfg.AutoRun)
		execPlan, err := engine.Plan(context.Background())
		if err != nil {
			return result, err
		}
		result.AutoRun = execPlan.Steps()
	}

	// Collect ManualRun as flat list
	for _, r := range cfg.ManualRun {
		tasks, err := CollectTasks(r)
		if err != nil {
			return result, err
		}
		result.ManualRun = append(result.ManualRun, tasks...)
	}

	return result, nil
}
