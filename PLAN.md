# PLAN

## Goals

Rewrite [Pocket](https://github.com/fredrikaverpil/pocket) (also available in
the parent folder from here) with feature parity in this v2 git worktree folder.

The previous public API and architecture can be viewed in:

- ../README.md
- ../ARCHITECTURE.md

It will be very important to quicky get to an end to end state, where we can
execute a task from the compositional configuration.

I want you to be my pair-programming buddy and help me gradually go through
every step carefully when planning and implementing.

Guidelines:

- Simplistic, easy-to-understand, ideomatic to Go, aims to use standard library
  for everything
- Clear API so that IDE code completion will help understanding how to construct
  config, tasks, tools. We can use different packages to signal the scope in
  which the user is currently configuring/building something.
- Core and internals are kept private/unexported. Avoid making internals
  available to users.
- Throughout the work, let's populate a new ARCHITECTURE.md file.
- Stop at inbetween each change, ask if I am happy with the implementation. Then
  git commit and move on.

## Phase 1: Read the config into a "plan"

- [x] Create a temporary "main.go" that runs (will be replaced later by the
      shim)
- [x] Define a new "hello-world" Task that takes option(s) like "name"
- [x] Define the Config struct
- [x] Define a Serial function, so that we can compose the config
- [x] Calculate the "plan"
- [x] Have plan sent to an "executor", which executes the composition

Questions to answer:

- [x] What is the composition actually, in technical terms. Is it a tree or a
      DAG? The action to show this could be called a "plan", but what is it,
      actually? Answer: It's a tree, not a DAG.
- [x] When the phase 1 is done, do we think we can generate a shim from the
      plan/tree/graph? Answer: Answer: Yes! The plan has everything needed:

  ```
  plan.Tasks // All task names
  plan.ModuleDirectories // Where to create shims
  plan.PathMappings // Which dirs each task runs in
  ```

  A shim in services/api/pok would:
  1. Parse CLI args: ./pok lint
  2. Look up "lint" in plan.Tasks
  3. Filter by current directory (services/api)
  4. Execute that specific task

  Gap: The current plan doesn't distinguish between:
  - CLI-invocable tasks (top-level, user wants ./pok lint)
  - Internal tasks (composition-only, like a helper task inside Serial)

## Phase 2: Shim generation

- [ ] CLI-invocable tasks (top-level, user wants ./pok lint)
- [ ] Internal tasks (composition-only, like a helper task inside Serial)
- [ ] Deduplication of tasks, a task only needs to run once
- [ ] The pok shims must be generated based on what the plan looks like
  - [ ] Simple initial pok shim is created in root
  - [ ] Pok shim gets created in each "module path"

Questions to answer:

- [ ] Where in the logic is shim generation invoked? Question: TBD

## Phase 2: Output and error handling

- [x] Implement pk.Parallel
- [ ] Buffered output
  - [ ] When a single task runs, run it without buffered output?
  - [ ] Waitgroups with errors?
  - [ ] Collect errors?
  - [ ] Fail fast; signal that something failed, abort other go-routines
  - [ ] First task to complete outputs the results

## Phase 3: Plan output

- [ ] Implement `./pok plan`
- [ ] Implement -json flag

Questions to answer:

- TBD

## Phase 4: Tasks and tools package structures

- [ ] Discuss how we will store tools and packages in pocket. In registry/tasks,
      registry/tools packages (or just tasks, tools packages)?
- [ ] Tools installations should be tested.
- [ ] Review the different kinds of tasks; automatically run on `./pok`,
      manually run, builtins.

## Phase 5:

TBD...

## Phase TBD (last phase)

- [ ] Go through each go file and add an equivalent \_test.go file, for adding
      unit tests.
- [ ] Keep Windows in mind. We need to support Windows.
