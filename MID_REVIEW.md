# Phase 7: Mid-Review Analysis

★ Insight ─────────────────────────────────────  
 This review compares the v2 rewrite against v1 across three dimensions: DX
(developer experience), project layout, and Go idioms. The key insight is that
v2 simplifies many areas but has a few places where v1's approach was more  
 ergonomic.  
 ─────────────────────────────────────────────────

1. DX Perspective: Is the API Surface Easy to Understand?

✅ Improvements in v2:  
 ┌──────────────┬──────────────────────────────────────┬────────────────────────────────────────┬─────────────────────┐
│ Area │ v1 │ v2 │ Verdict │
├──────────────┼──────────────────────────────────────┼────────────────────────────────────────┼─────────────────────┤
│ Task │ pocket.Task(name, usage, body, │ pk.NewTask(name, usage, flags, body) │
v2 wins - explicit │ │ creation │ opts...) with any body │ with typed Runnable │
types │
├──────────────┼──────────────────────────────────────┼────────────────────────────────────────┼─────────────────────┤
│ Composition │ Serial/Parallel accepting ...any │ Serial/Parallel accepting
...Runnable │ v2 wins - type │ │ │ │ │ safety │
├──────────────┼──────────────────────────────────────┼────────────────────────────────────────┼─────────────────────┤
│ Path │ RunIn(r, ...opts) │ WithOptions(r, ...opts) │ Neutral - both work │ │
filtering │ │ │ │
├──────────────┼──────────────────────────────────────┼────────────────────────────────────────┼─────────────────────┤
│ Flags │ Custom Opts() with struct reflection │ Standard flag.FlagSet │ v2
wins - familiar │ │ │ │ │ to Go devs │
├──────────────┼──────────────────────────────────────┼────────────────────────────────────────┼─────────────────────┤
│ Manual tasks │ Config.ManualRun: []Runnable{} │ Task.Manual() + Config.Manual
│ v2 wins - clearer │
└──────────────┴──────────────────────────────────────┴────────────────────────────────────────┴─────────────────────┘
⚠️ Areas where v1 was simpler:  
 ┌────────────────┬─────────────────────────────┬─────────────────────────────────────┬───────────────────────────────┐
│ Area │ v1 │ v2 │ Concern │
├────────────────┼─────────────────────────────┼─────────────────────────────────────┼───────────────────────────────┤
│ Static │ pocket.Run("go", "fmt") │ Must use pk.Do(func(ctx) { │ v1 simpler │ │
commands │ │ pk.Exec(...) }) │ │
├────────────────┼─────────────────────────────┼─────────────────────────────────────┼───────────────────────────────┤
│ Detection │ pocket.Detect(fn) where │ pk.WithDetect(fn) where fn(dirs, │ v1
simpler │ │ │ fn() []string │ gitRoot) []string │ │
├────────────────┼─────────────────────────────┼─────────────────────────────────────┼───────────────────────────────┤
│ Path options │ Include(), Exclude(), │ WithIncludePath(), │ v1 more concise │
│ naming │ Detect() │ WithExcludePath(), WithDetect() │ │
├────────────────┼─────────────────────────────┼─────────────────────────────────────┼───────────────────────────────┤
│ Task options │ Named(), AsHidden() as │ Task.Hidden() returns new task │ v1
more flexible │ │ │ TaskOpt │ │ │
├────────────────┼─────────────────────────────┼─────────────────────────────────────┼───────────────────────────────┤
│ Config │ Has Shim, SkipGenerate, │ Minimal - only Auto and Manual │
Trade-off - v2 has less │ │ │ SkipGitDiff │ │ config but fewer knobs │
└────────────────┴─────────────────────────────┴─────────────────────────────────────┴───────────────────────────────┘
Key API Observations:

1. v2's WithOptions() vs v1's RunIn() - Both work, but "RunIn" more clearly
   expresses intent ("run this in these  
   paths"). Consider renaming.
2. No Run() primitive in v2 - v1 has pocket.Run("cmd", args...) for static
   commands. v2 requires wrapping in Do().  
   This is extra boilerplate for the common case.
3. Detection signature changed - v1's simpler func() []string is more ergonomic.
   v2's func(dirs []string, gitRoot  
   string) []string forces users to understand the pre-walked directory
   mechanism.

---

2. Files/Packages Perspective: Is the Git Project Laid Out Well?

Project Structure Comparison:

v1 (flat): v2 (structured):  
 pocket-v1/ pocket/  
 ├── \*.go (30 files in root) ├── pk/ ✅ Public API package  
 ├── internal/ ├── internal/  
 │ ├── shim/ │ ├── shim/  
 │ └── scaffold/ │ └── scaffold/  
 ├── cmd/pocket/ ├── cmd/pocket/  
 ├── tools/ ├── tools/  
 └── tasks/ └── tasks/

✅ v2 Improvements:

- All public API in pk/ package - cleaner import path (github.com/.../pocket/pk)
- Root is clean, not cluttered with 30 Go files
- Clear separation: pk/ (public) vs internal/ (private)

✅ v2 File Organization in pk/:  
 ┌────────────────┬──────────────────────────────┬──────────────────────────┐  
 │ File │ Responsibility │ Assessment │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ task.go │ Task type │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ composition.go │ Serial/Parallel │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ paths.go │ PathOption + pathFilter │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ plan.go │ Plan building + tree walking │ ⚠️ Large (250+ lines) │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ context.go │ Context keys + utilities │ ⚠️ Many responsibilities │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ exec.go │ Command execution │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ cli.go │ CLI entry + dispatch │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ output.go │ Output buffering │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ install.go │ Tool installation │ ✅ Focused │  
 ├────────────────┼──────────────────────────────┼──────────────────────────┤  
 │ builtins.go │ Built-in tasks │ ✅ Focused │  
 └────────────────┴──────────────────────────────┴──────────────────────────┘  
 Potential Issue: context.go handles path context, plan context, execution
tracking, verbose mode, output context, flag overrides, and force-run. Consider
splitting by concern.

---

3. Go Idiomatic View: Is the Project Following Go Idioms?

✅ Strong Go Idioms in v2:  
 ┌──────────────────────────────┬─────────────────────────────────────────────────────────┐

│ Pattern │ Implementation │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Functional options │ PathOption func(\*pathFilter) │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Context propagation │ Output, path, plan all via context.Context │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Unexported interface methods │ Runnable.run() - prevents external impl │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Error wrapping │ Uses fmt.Errorf("...: %w", err) │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Sync primitives │ sync.Mutex for thread-safe dedup, errgroup for parallel │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Standard flag package │ Instead of custom reflection-based parsing │  
 ├──────────────────────────────┼─────────────────────────────────────────────────────────┤

│ Build tags │ exec_unix.go, exec_other.go for platform specifics │  
 └──────────────────────────────┴─────────────────────────────────────────────────────────┘

⚠️ Minor Concerns:

1. Pointer identity for deduplication - v2 uses map[*Task]bool for dedup. This
   works but means two \*Task with same  
   name are different. v1 uses reflect.ValueOf(r).Pointer() which is similar but
   more explicit about the identity model.
2. Global git root caching - cachedGitRoot uses a package-level variable with
   sync.Once. Idiomatic, but makes testing harder.
3. any in WithExcludeTask/WithSkipTask - Accept any for flexibility, but this
   means less compile-time safety.

---

4. Comparison Summary: v1 vs v2  
   ┌────────────────────┬─────────────────────────────────────────┬──────────────────────────────────────┬──────────────┐ │
   Dimension │ v1 │ v2 │ Winner │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Type safety │ Uses any in Task, Serial, Parallel │ Uses typed Runnable
   everywhere │ v2 │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Package layout │ Flat (30 files in root) │ Structured (pk/ package) │ v2 │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Flags │ Custom struct-based with reflection │ Standard flag.FlagSet │ v2 │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Static commands │ Run("cmd", args...) │ Must wrap in Do() │ v1 │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Path option naming │ Include(), Exclude() │ WithIncludePath(),
   WithExcludePath() │ v1 (shorter) │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Detection │ Simple func() []string │ Complex func(dirs, gitRoot) []string │
   v1 │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Config options │ Shim, SkipGenerate, SkipGitDiff │ Minimal │ Trade-off │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Task cloning │ Clone(task, opts...) │ Task.Hidden(), Task.Manual() │ v2
   (methods) │
   ├────────────────────┼─────────────────────────────────────────┼──────────────────────────────────────┼──────────────┤
   │ Dual-mode walk │ modeCollect vs modeExecute in same code │
   Plan-then-execute separation │ v2 (clearer) │
   └────────────────────┴─────────────────────────────────────────┴──────────────────────────────────────┴──────────────┘

---

5. Specific Recommendations

High Impact (Consider Implementing):

1. Add pk.Run() for static commands - Reduces boilerplate for the common case:  
   // Current v2:  
   pk.Do(func(ctx context.Context) error { return pk.Exec(ctx, "go", "fmt",
   "./...") })

// With Run():  
 pk.Run("go", "fmt", "./...")  
 2. Simplify detection signature - Make it match v1's ergonomics:  
 // Current v2:  
 pk.WithDetect(func(dirs []string, gitRoot string) []string { ... })

// Simpler:  
 pk.WithDetect(golang.Detect) // where Detect returns func() []string  
 3. Rename WithOptions to RunIn - More intuitive name that expresses the "run in
these paths" intent.

Medium Impact:

4. Consider shorter path option names - Include() vs WithIncludePath(). The
   "With" prefix is Go convention for  
   options, but these aren't task options.
5. Add SkipGenerate/SkipGitDiff to Config - v1 had useful CI workflow controls.

Low Impact/Defer:

6. Split context.go - When it grows further.
7. Version from git tag - Already noted in PLAN.md.

---

6. Questions to Discuss

1. Do we want a Run() primitive? It would mean another Runnable type, but
   significantly improves ergonomics for the  
   common case.
1. Is the detection function signature worth the complexity? The pre-walked dirs
   enable cumulative filtering, but the API is harder to use.
1. Should WithOptions be renamed to RunIn? It matches v1 and is more intuitive.
