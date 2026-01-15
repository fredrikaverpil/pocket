The new model:

1. Remove Serial(ctx, ...) entirely - no ctx parameter ever
2. Dependencies via composition:

   var Lint = pocket.Func("go-lint", "run linter", pocket.Serial(
   golangcilint.Install, lintImpl, ))

3. Collection phase: Walk the static tree structure recursively. Function bodies
   never run.
4. Execution phase: Run the tree. When a FuncDef is reached, call its function
   body.

So the function body is pure execution logic but can also contain additional
dependencies:

    func lintFn(ctx context.Context) error {
      pocket.Serial(golangcilint.InstallSomethingElse)
      return pocket.Exec(ctx, golangcilint.Name, "run", "./...")
    }

And ./pok plan just walks the Runnable tree (Serial/Parallel/FuncDef nodes)
without ever calling user function bodies.

But maybe it has to be:

    func lintFn(ctx context.Context) error {
      pocket.Serial(golangcilint.InstallSomethingElse)
      return pocket.Exec(ctx, golangcilint.Name, "run", "./...")
    }

Or:

    func lintFn(ctx context.Context) error {
      return pocket.Serial(
          golangcilint.Install,
          func(ctx context.Context) error {
              return pocket.Exec(ctx, golangcilint.Name, "run", "./...")
          },
      )
    }

I wonder if we implemented something unified and recursive now. Maybe it would
be easier to look at this from a purely functional standpoint. The composition
tree should maybe only consist of pocket.Serial|Parallel calls and inside we
would have Runnables. And we can nest pocket.Serial|Parallel with Runnables
anyway we like. At all times. Would this make the implementation simpler and
debugging simpler too?

There would be no need for the user to specify Run functions. They would only
work with pocket.Serial|Parallel and always return Runnables. And eventually
they would use pocket.Exec for system commands inside their functions.
