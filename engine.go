package pocket

import (
	"context"
	"io"
	"strings"
	"sync"
)

// execMode controls how Serial/Parallel/Exec behave.
type execMode int

const (
	// modeExecute is the default mode - actually run commands.
	modeExecute execMode = iota
	// modeCollect registers deps without executing, for plan generation.
	modeCollect
)

// PlanStep represents a single step in the execution plan.
type PlanStep struct {
	Type     string      `json:"type"`               // "serial", "parallel", "func"
	Name     string      `json:"name,omitempty"`     // Function name
	Usage    string      `json:"usage,omitempty"`    // Function usage/description
	Hidden   bool        `json:"hidden,omitempty"`   // Whether this is a hidden function
	Deduped  bool        `json:"deduped,omitempty"`  // Would be skipped due to deduplication
	Path     string      `json:"path,omitempty"`     // Path context for path-filtered execution
	Children []*PlanStep `json:"children,omitempty"` // Nested steps (for serial/parallel groups)
}

// ExecutionPlan holds the complete plan collected during modeCollect.
type ExecutionPlan struct {
	mu    sync.Mutex
	steps []*PlanStep
	stack []*PlanStep // Current nesting stack during collection
}

// newExecutionPlan creates a new empty execution plan.
func newExecutionPlan() *ExecutionPlan {
	return &ExecutionPlan{
		steps: make([]*PlanStep, 0),
		stack: make([]*PlanStep, 0),
	}
}

// AddFunc adds a function call to the plan.
func (p *ExecutionPlan) AddFunc(name, usage string, hidden, deduped bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	step := &PlanStep{
		Type:    "func",
		Name:    name,
		Usage:   usage,
		Hidden:  hidden,
		Deduped: deduped,
	}
	p.appendStep(step)
	// Push onto stack so nested deps become children
	p.stack = append(p.stack, step)
}

// PopFunc ends the current function's scope.
func (p *ExecutionPlan) PopFunc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.stack) > 0 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}

// PushSerial starts a serial group.
func (p *ExecutionPlan) PushSerial() {
	p.mu.Lock()
	defer p.mu.Unlock()
	step := &PlanStep{Type: "serial"}
	p.appendStep(step)
	p.stack = append(p.stack, step)
}

// PushParallel starts a parallel group.
func (p *ExecutionPlan) PushParallel() {
	p.mu.Lock()
	defer p.mu.Unlock()
	step := &PlanStep{Type: "parallel"}
	p.appendStep(step)
	p.stack = append(p.stack, step)
}

// Pop ends the current group.
func (p *ExecutionPlan) Pop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.stack) > 0 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}

// appendStep adds a step to the current context (root or nested).
// Must be called with lock held.
func (p *ExecutionPlan) appendStep(step *PlanStep) {
	if len(p.stack) == 0 {
		p.steps = append(p.steps, step)
	} else {
		parent := p.stack[len(p.stack)-1]
		parent.Children = append(parent.Children, step)
	}
}

// Steps returns the top-level steps in the plan.
func (p *ExecutionPlan) Steps() []*PlanStep {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.steps
}

// Tasks flattens the execution plan into a list of TaskInfo.
// This extracts all func steps from the tree, combining with path information
// from the provided pathMappings. Tasks without path mappings get ["."].
func (p *ExecutionPlan) Tasks(pathMappings map[string]*PathFilter) []TaskInfo {
	p.mu.Lock()
	steps := p.steps
	p.mu.Unlock()

	var result []TaskInfo
	seen := make(map[string]bool) // deduplicate by name
	collectTasksFromSteps(steps, pathMappings, &result, seen)
	return result
}

// collectTasksFromSteps recursively extracts TaskInfo from plan steps.
func collectTasksFromSteps(
	steps []*PlanStep,
	pathMappings map[string]*PathFilter,
	result *[]TaskInfo,
	seen map[string]bool,
) {
	for _, step := range steps {
		switch step.Type {
		case "func":
			// Skip if already seen (deduplication across plan tree)
			if seen[step.Name] {
				continue
			}
			seen[step.Name] = true

			info := TaskInfo{
				Name:   step.Name,
				Usage:  step.Usage,
				Hidden: step.Hidden,
			}

			// Get paths from mapping, default to ["."] for root-only tasks
			if pf, ok := pathMappings[step.Name]; ok {
				info.Paths = pf.Resolve()
			} else {
				info.Paths = []string{"."}
			}

			*result = append(*result, info)

			// Recurse into nested deps
			if len(step.Children) > 0 {
				collectTasksFromSteps(step.Children, pathMappings, result, seen)
			}

		case "serial", "parallel":
			// Recurse into children
			if len(step.Children) > 0 {
				collectTasksFromSteps(step.Children, pathMappings, result, seen)
			}
		}
	}
}

// Engine orchestrates plan collection and execution.
type Engine struct {
	root Runnable
}

// NewEngine creates an engine for the given runnable tree.
func NewEngine(root Runnable) *Engine {
	return &Engine{root: root}
}

// Plan collects the complete execution plan without running anything.
func (e *Engine) Plan(ctx context.Context) (*ExecutionPlan, error) {
	if e.root == nil {
		return newExecutionPlan(), nil
	}

	plan := newExecutionPlan()
	ec := &execContext{
		mode:    modeCollect,
		out:     discardOutput(),
		cwd:     ".",
		verbose: false,
		dedup:   newDedupState(),
		plan:    plan,
	}
	ctx = withExecContext(ctx, ec)

	// Walk the tree - functions run but register instead of execute
	if err := e.root.run(ctx); err != nil {
		return nil, err
	}

	return plan, nil
}

// Execute runs the tree with normal execution.
func (e *Engine) Execute(ctx context.Context, out *Output, cwd string, verbose bool) error {
	return runWithContext(ctx, e.root, out, cwd, verbose)
}

// discardOutput returns an output that discards all writes.
func discardOutput() *Output {
	return &Output{Stdout: io.Discard, Stderr: io.Discard}
}

// Print outputs the execution plan tree.
func (p *ExecutionPlan) Print(ctx context.Context, showHidden, showDedup bool) {
	p.mu.Lock()
	steps := p.steps
	p.mu.Unlock()

	printPlanSteps(ctx, steps, "  ", showHidden, showDedup)
}

// printPlanSteps recursively prints plan steps with indentation.
func printPlanSteps(ctx context.Context, steps []*PlanStep, indent string, showHidden, showDedup bool) {
	// Filter steps based on visibility options
	visible := filterPlanSteps(steps, showHidden, showDedup)

	for i, step := range visible {
		last := i == len(visible)-1
		connector := "├── "
		if last {
			connector = "└── "
		}
		childIndent := indent + "│   "
		if last {
			childIndent = indent + "    "
		}

		switch step.Type {
		case "func":
			label := step.Name
			var annotations []string
			if step.Hidden {
				annotations = append(annotations, "hidden")
			}
			if step.Deduped {
				annotations = append(annotations, "skipped")
			}
			if len(annotations) > 0 {
				label += " (" + strings.Join(annotations, ", ") + ")"
			}
			if step.Usage != "" && !step.Hidden {
				label += " - " + step.Usage
			}
			Printf(ctx, "%s%s%s\n", indent, connector, label)

			// Print children (nested deps)
			if len(step.Children) > 0 {
				printPlanSteps(ctx, step.Children, childIndent, showHidden, showDedup)
			}

		case "serial":
			filtered := filterPlanSteps(step.Children, showHidden, showDedup)
			if len(filtered) == 0 {
				continue
			}
			Printf(ctx, "%s%sSerial:\n", indent, connector)
			printPlanSteps(ctx, step.Children, childIndent, showHidden, showDedup)

		case "parallel":
			filtered := filterPlanSteps(step.Children, showHidden, showDedup)
			if len(filtered) == 0 {
				continue
			}
			Printf(ctx, "%s%sParallel:\n", indent, connector)
			printPlanSteps(ctx, step.Children, childIndent, showHidden, showDedup)
		}
	}
}

// filterPlanSteps filters steps based on visibility options.
func filterPlanSteps(steps []*PlanStep, showHidden, showDedup bool) []*PlanStep {
	result := make([]*PlanStep, 0, len(steps))
	for _, step := range steps {
		// Skip hidden unless showHidden is true
		if step.Hidden && !showHidden {
			continue
		}
		// Skip deduped unless showDedup is true
		if step.Deduped && !showDedup {
			continue
		}
		result = append(result, step)
	}
	return result
}
