❯ ultrathink on how we can pass in argumetns to the tasks. Perhaps we can use a
"--" to denote that what follows  
 after that are args that gets passed into e.g. `go test`, `mdformat` etc.

⏺ Let me explore the codebase first to understand the current CLI and task
execution flow.

⏺ Read(cli.go) ⎿  Read 243 lines

⏺ Read(task.go) ⎿  Read 210 lines

⏺ Read(runnable.go) ⎿  Read 117 lines

⏺ Let me also look at a task implementation to understand how commands are
currently built.

⏺ Read(tasks/golang/tasks.go) ⎿  Read 231 lines

❯ I'm thinking maybe `./pok -v go-test foo=bar -- -race`

⏺ Now I have a clear picture of the current architecture. Let me think through
this design carefully.

Deep Analysis: Pass-through Arguments with --

Proposed Syntax

./pok -v go-test foo=bar -- -race -run TestFoo

Where:

- -v → pocket flag
- go-test → task name
- foo=bar → task argument (existing key=value)
- -- → separator
- -race -run TestFoo → extra args passed to underlying tool

Design Decisions

1. Where to store extra args?

Two options:

- Context (like IsVerbose(ctx)) — consistent with existing patterns
- RunContext (add ExtraArgs []string field) — keeps it with other runtime data

I'd recommend context for consistency. You already have verboseKey, cwdKey,
skipKey. Adding extraArgsKey follows the sam e pattern and can be accessed
anywhere without threading.

2. Multiple tasks with --?

./pok go-lint go-test -- -v # What happens? ./pok -- -v # Default "all" task

Options:

- Apply to all: Extra args go to every task (mirrors npm test -- --watch)
- Error: Require single task when using --
- Apply to CLI-specified task only: Only explicit task gets args, not its
  dependencies

I'd suggest apply to all, let tasks opt-in. Tasks call ExtraArgs(ctx) if they
want them. Tasks that don't call it simply ignore them. If the underlying tool
doesn't understand the args, it fails clearly.

3. Opt-in vs opt-out?

Tasks should opt-in by calling ExtraArgs(ctx) and appending. This means:

- Existing tasks work unchanged (ignore extra args)
- No need for AcceptsExtraArgs bool field
- Clear which tasks support it (those that call the helper)

Implementation Plan

1. CLI parsing (cli.go):

// Before flag.Parse(), split on -- var pocketArgs, extraArgs []string for i,
arg := range os.Args[1:] { if arg == "--" { pocketArgs = os.Args[1:i+1]
extraArgs = os.Args[i+2:] break } } if pocketArgs == nil { pocketArgs =
os.Args[1:] } os.Args = append([]string{os.Args[0]}, pocketArgs...)

2. Context helper (task.go):

const extraArgsKey contextKey = iota + 3 // after skipKey

func WithExtraArgs(ctx context.Context, args []string) context.Context { return
context.WithValue(ctx, extraArgsKey, args) }

func ExtraArgs(ctx context.Context) []string { if args, ok :=
ctx.Value(extraArgsKey).([]string); ok { return args } return nil }

3. Task usage (e.g., go-test):

args := []string{"test", "-race", "./..."} args = append(args,
pocket.ExtraArgs(ctx)...) // Append extra args cmd := pocket.Command(ctx, "go",
args...)

Edge Cases ┌─────────────────────────────┬─────────────────────────────────────┐
│ Input │ Behavior │
├─────────────────────────────┼─────────────────────────────────────┤ │ ./pok
go-test -- │ Empty extra args (not nil) │
├─────────────────────────────┼─────────────────────────────────────┤ │ ./pok
go-test -- -v -- more │ Extra args: ["-v", "--", "more"] │
├─────────────────────────────┼─────────────────────────────────────┤ │ ./pok -h
go-test -- │ Shows help (flags take precedence) │
├─────────────────────────────┼─────────────────────────────────────┤ │ ./pok --
│ Runs default task, empty extra args │
└─────────────────────────────┴─────────────────────────────────────┘ Help Text
Update

Usage: pok [flags] <task> [key=value...] [-- extra-args...]

Arguments after -- are passed to the underlying tool.

Alternative Considered: Environment Variable

POCKET_EXTRA_ARGS="-v -run TestFoo" ./pok go-test

Rejected because:

- Verbose for one-off use
- Poor discoverability
- Awkward shell quoting

Summary

The -- approach is the right choice:

- Familiar — Standard Unix convention (npm test --, cargo run --, etc.)
- Clean — Clear separation between pocket args and tool args
- Opt-in — Tasks choose to use extra args via ExtraArgs(ctx)
- Minimal — Small CLI change, context helper, tasks opt-in as needed

Want me to implement this?

⏺ Yes, that syntax is clean and follows the standard convention. The parsing
order is clear:

./pok -v go-test foo=bar -- -race │ │ │ │ │ │ │ │ │ └── extra args → passed to
`go test` │ │ │ └─────── separator │ │ └──────────── task arg (key=value) │
└──────────────────── task name └───────────────────────── pocket flag

This mirrors how npm test -- --watch and cargo run -- args work.

The implementation is straightforward:

1. Split os.Args on -- before flag.Parse()
2. Store extra args in context via WithExtraArgs(ctx, extraArgs)
3. Tasks call pocket.ExtraArgs(ctx) to get them and append to their commands

Want me to implement it?
