# GitHub Actions Integration

Pocket provides two approaches for GitHub Actions CI/CD integration:

1. **Simple Workflow** - A static workflow that runs all tasks on configured
   platforms
2. **Matrix Workflow** - A dynamic two-phase workflow that generates a job
   matrix from your task configuration

## Simple Workflow

The simple approach runs `./pok` on each configured platform:

```bash
./pok github-workflows
```

This generates `.github/workflows/pocket.yml`:

```yaml
jobs:
  pocket:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - run: ./pok
```

**Pros**: Simple, predictable, easy to understand.

**Cons**: All tasks run serially; no per-task platform customization; no
visibility into which task failed.

## Matrix Workflow

The matrix workflow is a two-phase approach that dynamically generates a GitHub
Actions matrix from your task configuration:

**Phase 1 (Plan)**: Runs `./pok gha-matrix` to generate a JSON matrix based on
your config.

**Phase 2 (Run)**: Uses the matrix to run each task as a separate job, with
per-task platform configuration.

### Setup

1. Add the matrix task to your config:

```go
var matrixConfig = github.MatrixConfig{
    DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
    TaskOverrides: map[string]github.TaskOverride{
        "go-lint": {Platforms: []string{"ubuntu-latest"}}, // lint only on Linux
    },
    ExcludeTasks: []string{"github-workflows"}, // don't run in CI
}

var Config = &pk.Config{
    Auto: pk.Parallel(
        golang.Tasks(),
        pk.WithOptions(
            github.Workflows,
            pk.WithIncludePath(`^\.$`),
            pk.WithFlag(github.Workflows, "skip-pocket", true),
            pk.WithFlag(github.Workflows, "include-pocket-matrix", true),
        ),
    ),
    Manual: []pk.Runnable{
        github.Matrix(matrixConfig),
    },
}
```

2. Generate the workflow:

```bash
./pok  # regenerates pocket-matrix.yml with your config
```

### How It Works

The generated `.github/workflows/pocket-matrix.yml` has two jobs:

```yaml
jobs:
  plan:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - run: echo "matrix=$(./pok gha-matrix)" >> "$GITHUB_OUTPUT"

  run:
    needs: plan
    strategy:
      matrix: ${{ fromJson(needs.plan.outputs.matrix) }}
    runs-on: ${{ matrix.os }}
    steps:
      - run: ${{ matrix.shim }} ${{ matrix.task }}
      - run: git diff --exit-code # if matrix.gitDiff
```

Running `./pok gha-matrix` outputs JSON like:

```json
{
  "include": [
    {
      "task": "go-lint",
      "os": "ubuntu-latest",
      "shell": "bash",
      "shim": "./pok",
      "gitDiff": true
    },
    {
      "task": "go-test",
      "os": "ubuntu-latest",
      "shell": "bash",
      "shim": "./pok",
      "gitDiff": true
    },
    {
      "task": "go-test",
      "os": "macos-latest",
      "shell": "bash",
      "shim": "./pok",
      "gitDiff": true
    },
    {
      "task": "go-test",
      "os": "windows-latest",
      "shell": "pwsh",
      "shim": ".\\pok.ps1",
      "gitDiff": true
    }
  ]
}
```

### MatrixConfig Options

```go
type MatrixConfig struct {
    // DefaultPlatforms for all tasks. Default: ["ubuntu-latest"]
    DefaultPlatforms []string

    // TaskOverrides provides per-task platform configuration.
    // Keys are regular expressions matched against task names.
    TaskOverrides map[string]TaskOverride

    // ExcludeTasks removes tasks from the matrix entirely.
    ExcludeTasks []string

    // WindowsShell: "powershell" (default) or "bash"
    WindowsShell string

    // WindowsShim: "ps1" (default) or "cmd"
    WindowsShim string
}

type TaskOverride struct {
    // Platforms overrides DefaultPlatforms for this task.
    Platforms []string

    // SkipGitDiff disables the git-diff check after this task.
    // Useful for tasks that intentionally modify files.
    SkipGitDiff bool
}
```

### Example: Python Matrix with Multiple Versions

```go
var matrixConfig = github.MatrixConfig{
    DefaultPlatforms: []string{"ubuntu-latest", "macos-latest"},
    TaskOverrides: map[string]github.TaskOverride{
        // Match all py-test variants with regex
        "py-test:.*": {SkipGitDiff: true},
        // Lint only on one platform
        "py-lint": {Platforms: []string{"ubuntu-latest"}},
    },
}
```

### Benefits

| Feature                          | Simple    | Matrix   |
| -------------------------------- | --------- | -------- |
| Per-task visibility in GitHub UI | No        | Yes      |
| Per-task platform configuration  | No        | Yes      |
| Parallel task execution          | No        | Yes      |
| Fail-fast granularity            | All tasks | Per task |
| Configuration complexity         | Low       | Medium   |

### When to Use Which

**Use Simple Workflow when:**

- You have few tasks
- All tasks should run on all platforms
- You want minimal configuration

**Use Matrix Workflow when:**

- You want each task as a separate GitHub Actions job
- You need per-task platform configuration (e.g., lint only on Linux)
- You want parallel task execution in CI
- You want clear visibility into which specific task failed
