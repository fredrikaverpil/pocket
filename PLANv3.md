See @README.md and we have just implemented `./pok plan`. The idea with that was
to see the composition of  
 functions from the config's AutoRun without actually running it.

However, it showed a flaw in the architecture. We should have an engine which
can:

- Calculate where shims are to be created (path resolution).
- Calculate the serial/parallel execution order of functions (and nested
  pocket.Serial|Parallel within those  
  functions) <-- this we don't have today
- Leverage a pattern where we have minimal and consistent API UX, like always
  resort to using  
  pocket.Serial|pocket.Parallel without introducing a new concept like
  pocket.Deps.

Do you think it's doable?  
 Currently pocket.Serial|Parallel can optionally take ctx. Maybe we should
simply do away with that capability, so that it does not take ctx ever. Instead
the engine takes care of collecting the functions into the tree of  
 pocket.Serial|Parallel calls, and executes them (with ctx) in the correct
order.

What do you think about that?  
 Then `./pok plan` would also just print the plan from the engine and not have
to sprinkle code all over the place in order to figure out what the plan is
like.

Ultrathink

I want to stress; please try NOT to introduce a new pocket.Deps because it seems
like a pit which is easy to fall into.

## The Solution: Context-Based Mode Switching

Keep pocket.Serial(ctx, ...) but make it behave differently based on a mode in
the context:

func lint(ctx context.Context) error { pocket.Serial(ctx, Install) // Works in
BOTH modes return pocket.Exec(ctx, "tool", "lint", "./...") }

The engine controls the mode:
┌─────────┬────────────────────────────────────┬──────────────────────────────────┐
│ Mode │ Serial(ctx, ...) behavior │ Exec(ctx, ...) behavior │
├─────────┼────────────────────────────────────┼──────────────────────────────────┤
│ Collect │ Registers deps to ctx, returns nil │ Registers as action, returns
nil │
├─────────┼────────────────────────────────────┼──────────────────────────────────┤
│ Execute │ Runs deps (with dedup) │ Actually runs command │
└─────────┴────────────────────────────────────┴──────────────────────────────────┘
type Engine struct { root Runnable }

func (e *Engine) Plan() *Plan { ctx := newContext(modeCollect) e.root.run(ctx)
// Calls all functions, they register deps return buildPlanFromRegistry(ctx) }

func (e \*Engine) Execute(ctx context.Context) error { ctx = withMode(ctx,
modeExecute) return e.root.run(ctx) }

---

How It Works

Config (top-level composition): var Config = pocket.Config{ AutoRun:
pocket.Serial( Format, pocket.Parallel(Lint, Test), Build, ), }

Function with nested deps: func lint(ctx context.Context) error {
pocket.Serial(ctx, Install) // Nested dep return pocket.Exec(ctx, ruff.Name,
"check", ".") }

Planning phase:

1. Engine creates ctx with modeCollect
2. Walks Serial(Format, Parallel(Lint, Test), Build)
3. Calls each function—they register their internal Serial(ctx, Install) calls
4. Engine now sees the complete tree including nested deps

Execution phase:

1. Engine creates ctx with modeExecute
2. Runs the collected plan
3. Deduplicates—each FuncDef runs at most once

---

API Stays the Same

- pocket.Serial(ctx, Install) - still works, behavior depends on ctx mode
- pocket.Parallel(ctx, A, B) - same
- pocket.Exec(ctx, ...) - same
- No new Deps concept

The only new concept is the Engine which owns context creation and mode
switching.

---

★ Insight ───────────────────────────────────── The key realization: ctx already
carries execution state. Adding a "mode" field lets the same API serve both
planning and execution—functions don't need to know which mode they're in.
─────────────────────────────────────────────────

---

Current Architecture Summary

What Works Well
┌───────────────────────┬───────────────────┬───────────────────────────────────┐
│ Component │ Location │ Description │
├───────────────────────┼───────────────────┼───────────────────────────────────┤
│ execContext │ context.go:12-19 │ Already carries state through ctx │
├───────────────────────┼───────────────────┼───────────────────────────────────┤
│ dedupState │ context.go:23-45 │ Thread-safe deduplication │
├───────────────────────┼───────────────────┼───────────────────────────────────┤
│ serial/parallel types │ group.go:17-92 │ Composition mode already exists │
├───────────────────────┼───────────────────┼───────────────────────────────────┤
│ printTree │ runner.go:237-300 │ Walks static tree structure │
└───────────────────────┴───────────────────┴───────────────────────────────────┘
The Gap

printTree can see the static tree (what's in Config.AutoRun), but cannot see
nested Serial(ctx, ...) calls inside func tion bodies:

func lint(ctx context.Context) error { pocket.Serial(ctx, Install) // ←
INVISIBLE to printTree return pocket.Exec(ctx, "tool", "lint", "./...") }

Line 264-266 in runner.go only sees f.body if it was set via composition, not
runtime calls.

---

Proposed Solution: Mode-Based Execution

1. Add mode to execContext

// context.go type execMode int

const ( modeExecute execMode = iota // Default: actually run commands
modeCollect // Collect deps without executing )

type execContext struct { mode execMode // NEW out *Output path string cwd
string verbose bool opts map[string]any dedup *dedupState plan \*ExecutionPlan
// NEW: collects during modeCollect }

2. Modify Serial/Parallel execution mode

// group.go - modify executeSerial func executeSerial(ctx context.Context, items
[]any) error { ec := getExecContext(ctx)

      if ec.mode == modeCollect {
          // Just register, don't execute
          for _, item := range items {
              r := toRunnable(item)
              ec.plan.AddSerial(r)
          }
          return nil
      }

      // Existing execution logic...
      for _, item := range items {
          r := toRunnable(item)
          if !shouldRun(ec, r) {
              continue
          }
          if err := r.run(ctx); err != nil {
              return err
          }
      }
      return nil

}

3. Modify Exec to be a no-op in collect mode

// context.go func Exec(ctx context.Context, name string, args ...string) error
{ ec := getExecContext(ctx)

      if ec.mode == modeCollect {
          // Register as action, don't execute
          ec.plan.AddAction(name, args...)
          return nil
      }

      // Existing execution logic...

}

4. Create Engine type

// engine.go (new file) type Engine struct { root Runnable }

type ExecutionPlan struct { steps []PlanStep // Flattened, ordered steps }

type PlanStep struct { Type string // "serial", "parallel", "func", "action"
Name string // Function name or command Children []\*PlanStep }

func (e *Engine) Plan(ctx context.Context) (*ExecutionPlan, error) { plan :=
&ExecutionPlan{} ec := newExecContext(nil, ".", false) ec.mode = modeCollect
ec.plan = plan

      ctx = withExecContext(ctx, ec)

      // This walks the tree and calls all functions
      // They register their deps to ec.plan instead of executing
      if err := e.root.run(ctx); err != nil {
          return nil, err
      }

      return plan, nil

}

func (e \*Engine) Execute(ctx context.Context) error { // Could use the plan, or
just run directly with modeExecute return Run(ctx, e.root, StdOutput(), ".",
false) }

5. Update ./pok plan to use Engine

// runner.go Func("plan", "show the execution tree", func(ctx context.Context)
error { engine := &Engine{root: cfg.AutoRun} plan, err :=
engine.Plan(context.Background()) if err != nil { return err } plan.Print(ctx)
// Shows full tree including nested deps return nil })

---

Key Benefits
┌──────────────────────────────────────────┬────────────────────────────────────────────┐
│ Before │ After │
├──────────────────────────────────────────┼────────────────────────────────────────────┤
│ Serial(ctx, ...) executes immediately │ Respects mode: collect or execute │
├──────────────────────────────────────────┼────────────────────────────────────────────┤
│ Plan can't see nested deps │ Plan sees everything │
├──────────────────────────────────────────┼────────────────────────────────────────────┤
│ Two code paths for planning vs execution │ Single tree walk, mode determines
behavior │
├──────────────────────────────────────────┼────────────────────────────────────────────┤
│ No explicit plan object │ ExecutionPlan is first-class │
└──────────────────────────────────────────┴────────────────────────────────────────────┘

---

★ Insight ─────────────────────────────────────

- The API stays the same: pocket.Serial(ctx, Install) still works
- The behavior changes based on ctx mode, which is invisible to user code
- This is similar to how database transactions work: same API, different
  behavior based on context ─────────────────────────────────────────────────

---

Implementation Order

1. Add mode and plan fields to execContext
2. Modify executeSerial and executeParallel to check mode
3. Modify Exec, Printf, etc. to be no-ops in collect mode
4. Create Engine and ExecutionPlan types
5. Update ./pok plan to use Engine
6. Add tests for collect mode
