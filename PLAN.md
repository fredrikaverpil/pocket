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

## Phase 1: Read the config into a "plan"

- [ ] Create a temporary "main.go" that runs (will be replaced later by the
      shim)
- [ ] Define a new "hello-world" Task that takes option(s) like "name"
- [ ] Define the Config struct
- [ ] Define a Serial function, so that we can compose the config
- [ ] Calculate the "plan"
- [ ] Have plan sent to an "executor", which executes the composition

Questions to answer:

- [ ] What is the composition actually, in technical terms. Is it a tree or a
      DAG? The action to show this could be called a "plan", but what is it,
      actually?
- [ ] When the phase 1 is done, do we think we can generate a shim from the
      plan/tree/graph?
