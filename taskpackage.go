package pocket

import (
	"context"
	"slices"
)

// TaskDef defines a task within a TaskGroupDef.
type TaskDef[O any] struct {
	// Name is the full task name (e.g., "go-format", "go-lint").
	Name string
	// Create returns a Task for the given modules.
	Create func(modules map[string]O) *Task
}

// TaskGroupDef defines a collection of related tasks for a language/technology.
// O is the options type (e.g., golang.Options) which must implement ModuleConfig.
type TaskGroupDef[O ModuleConfig] struct {
	// Name is the task group identifier (e.g., "go", "python").
	Name string
	// Detect returns paths where this task group applies (for Auto mode).
	Detect func() []string
	// Tasks defines the tasks in this group.
	Tasks []TaskDef[O]
}

// Auto creates a TaskGroup that auto-detects modules using the Detect function.
// The defaults parameter specifies default options for all detected modules.
// Skip patterns can be passed to exclude paths or specific tasks.
func (g *TaskGroupDef[O]) Auto(defaults O, opts ...SkipOption) TaskGroup {
	cfg := newSkipConfig(opts...)
	return &autoTaskGroup[O]{
		def:      g,
		skipCfg:  cfg,
		defaults: defaults,
		detected: nil, // lazily populated
	}
}

// New creates a TaskGroup with explicit module configuration.
func (g *TaskGroupDef[O]) New(modules map[string]O) TaskGroup {
	return &explicitTaskGroup[O]{
		def:     g,
		modules: modules,
	}
}

// autoTaskGroup implements TaskGroup for auto-detected modules.
type autoTaskGroup[O ModuleConfig] struct {
	def      *TaskGroupDef[O]
	skipCfg  *skipConfig
	defaults O            // default options for all detected modules
	detected map[string]O // lazily populated
}

func (tg *autoTaskGroup[O]) Name() string            { return tg.def.Name }
func (tg *autoTaskGroup[O]) DetectModules() []string { return tg.def.Detect() }

func (tg *autoTaskGroup[O]) doDetect() map[string]O {
	if tg.detected != nil {
		return tg.detected
	}

	paths := tg.def.Detect()
	modules := make(map[string]O, len(paths))
	for _, p := range paths {
		// Skip paths that match skip patterns
		if tg.skipCfg.shouldSkipPath(p) {
			continue
		}
		modules[p] = tg.defaults
	}

	tg.detected = modules
	return modules
}

func (tg *autoTaskGroup[O]) ModuleConfigs() map[string]ModuleConfig {
	detected := tg.doDetect()
	modules := make(map[string]ModuleConfig, len(detected))
	for path, opts := range detected {
		modules[path] = opts
	}
	return modules
}

func (tg *autoTaskGroup[O]) ForContext(ctx string) TaskGroup {
	detected := tg.doDetect()
	if ctx == "." {
		return &explicitTaskGroup[O]{
			def:     tg.def,
			modules: detected,
			skipCfg: tg.skipCfg,
		}
	}
	if opts, ok := detected[ctx]; ok {
		return &explicitTaskGroup[O]{
			def:     tg.def,
			modules: map[string]O{ctx: opts},
			skipCfg: tg.skipCfg,
		}
	}
	return nil
}

func (tg *autoTaskGroup[O]) Tasks(cfg Config) []*Task {
	detected := tg.doDetect()
	return (&explicitTaskGroup[O]{
		def:     tg.def,
		modules: detected,
		skipCfg: tg.skipCfg,
	}).Tasks(cfg)
}

// explicitTaskGroup implements TaskGroup for explicitly configured modules.
type explicitTaskGroup[O ModuleConfig] struct {
	def     *TaskGroupDef[O]
	modules map[string]O
	skipCfg *skipConfig // may be nil for New() without skip options
}

func (tg *explicitTaskGroup[O]) Name() string            { return tg.def.Name }
func (tg *explicitTaskGroup[O]) DetectModules() []string { return tg.def.Detect() }

func (tg *explicitTaskGroup[O]) ModuleConfigs() map[string]ModuleConfig {
	modules := make(map[string]ModuleConfig, len(tg.modules))
	for path, opts := range tg.modules {
		modules[path] = opts
	}
	return modules
}

func (tg *explicitTaskGroup[O]) ForContext(ctx string) TaskGroup {
	if ctx == "." {
		return tg
	}
	if opts, ok := tg.modules[ctx]; ok {
		return &explicitTaskGroup[O]{
			def:     tg.def,
			modules: map[string]O{ctx: opts},
			skipCfg: tg.skipCfg,
		}
	}
	return nil
}

func (tg *explicitTaskGroup[O]) Tasks(_ Config) []*Task {
	tasks := make([]*Task, 0, len(tg.def.Tasks)+1) // +1 for orchestrator
	taskPtrs := make([]*Task, 0, len(tg.def.Tasks))

	for _, def := range tg.def.Tasks {
		mods := tg.modulesFor(def.Name)
		if len(mods) == 0 {
			continue
		}
		task := def.Create(mods)
		tasks = append(tasks, task)
		taskPtrs = append(taskPtrs, task)
	}

	// Create orchestrator task that runs all tasks serially.
	if len(taskPtrs) > 0 {
		hidden := true
		if tg.skipCfg != nil && tg.skipCfg.showAll {
			hidden = false
		}
		allTask := &Task{
			Name:   tg.def.Name + "-all",
			Usage:  "run all " + tg.def.Name + " tasks",
			Hidden: hidden,
			Action: func(ctx context.Context, _ map[string]string) error {
				return SerialDeps(ctx, taskPtrs...)
			},
		}
		tasks = append(tasks, allTask)
	}

	return tasks
}

// modulesFor returns modules where the given task should run.
func (tg *explicitTaskGroup[O]) modulesFor(taskName string) map[string]O {
	result := make(map[string]O)
	for path, opts := range tg.modules {
		// Check Options.ShouldRun (respects Skip field in Options)
		if !opts.ShouldRun(taskName) {
			continue
		}
		// Check skip config patterns (from Auto mode)
		if tg.skipCfg != nil && tg.skipCfg.shouldSkipTask(taskName, path) {
			continue
		}
		result[path] = opts
	}
	return result
}

// BaseOptions provides a default ShouldRun implementation for Options structs.
// Embed this in your Options struct to get skip functionality for free.
type BaseOptions struct {
	// Skip lists full task names to skip (e.g., ["go-lint", "go-vulncheck"]).
	Skip []string
}

// ShouldRun returns true if the given task should run based on the Skip list.
func (o BaseOptions) ShouldRun(taskName string) bool {
	return !slices.Contains(o.Skip, taskName)
}
