# Build Tool Comparison

This document compares Pocket with other build systems and task runners to help
you choose the right tool for your project.

## Where Pocket Fits

Build tools exist on a spectrum from simple command runners to enterprise build
systems:

```
Simple                                                              Complex
  │                                                                      │
  ▼                                                                      ▼
┌─────┐  ┌──────┐  ┌──────┐  ┌──────────────┐  ┌─────────┐  ┌───────┐  ┌───────┐
│Just │──│ Task │──│ Mage │──│ Pocket / Sage│──│ Dagger  │──│Earthly│──│ Bazel │
└─────┘  └──────┘  └──────┘  └──────────────┘  └─────────┘  └───────┘  └───────┘
  │         │         │              │              │           │          │
  │         │         │              │              │           │          │
Command   YAML      Go code      Go code +      Containers  Containers  Hermetic
runner    tasks     tasks        tool mgmt      + pipelines + builds    monorepo
```

Pocket and Sage occupy the same niche with different trade-offs:

- **Pocket**: Pure Go execution, cross-platform shim, path filtering for
  monorepos
- **Sage**: Generates Makefiles (IDE integration), requires GNU Make, Go-focused
  tooling

**Pocket's sweet spot:** Projects that need more than a Makefile but don't
require container isolation or enterprise-scale features.

### When Pocket is a good fit

- **Polyglot projects** — Configuration is in Go, but tasks can target any
  language (Go, Python, Rust, etc.)
- **Cross-platform development** — Works on Windows, macOS, and Linux without
  shell script gymnastics
- **Tool management fatigue** — Tired of "please install X before running"?
  Pocket downloads and caches tools automatically
- **Monorepos with structure** — Auto-detection finds all Go modules, Python
  projects, etc. and runs consistent task groups across them
- **Path filtering** — Control which tasks are visible in which directories
  (`Paths().In("services/api")`)
- **CI simplicity** — Your CI just runs `./pok`; no tool installation steps

### When to look elsewhere

- **Container-first workflows** — If you need isolated, reproducible containers,
  consider Dagger or Earthly
- **Enterprise scale** — For 100+ developer monorepos with remote caching needs,
  Bazel is purpose-built
- **Minimal setup** — If you just need to run a few commands, Just or Task may
  be simpler

### Pocket vs the closest alternatives

| If you're using... | Consider Pocket if...                                        |
| ------------------ | ------------------------------------------------------------ |
| **Make**           | You want cross-platform support and built-in tool management |
| **Mage**           | You want bundled tasks and automatic tool downloading        |
| **Sage**           | You don't need Makefile generation and want path filtering   |
| **Task**           | You prefer Go code over YAML for complex logic               |
| **Dagger**         | You don't need container isolation and want less complexity  |

## Quick Comparison

| Tool        | Config Format          | Language | Tool Management | Containers | Best For                                |
| ----------- | ---------------------- | -------- | --------------- | ---------- | --------------------------------------- |
| **Pocket**  | Go code                | Go       | Built-in        | No         | Cross-platform, polyglot projects       |
| **Sage**    | Go code + Make         | Go       | Built-in        | No         | Go projects at Einride scale            |
| **Mage**    | Go code                | Go       | Manual          | No         | Go projects wanting full language power |
| **Task**    | YAML                   | Go       | Manual          | No         | Teams wanting readable configs          |
| **Just**    | Makefile-like          | Rust     | No              | No         | Simple command running                  |
| **Dagger**  | Go/Python/TS           | Go       | Via containers  | Yes        | Portable CI/CD pipelines                |
| **Earthly** | Dockerfile-like        | Go       | Via containers  | Yes        | Containerized builds, monorepos         |
| **Bazel**   | Starlark (Python-like) | Java/C++ | Hermetic        | Optional   | Large-scale monorepos                   |
| **Make**    | Makefile               | C        | No              | No         | Unix systems, C/C++ projects            |

## Feature Matrix

| Feature             | Pocket | Sage | Mage | Task | Just | Dagger | Earthly | Bazel |
| ------------------- | ------ | ---- | ---- | ---- | ---- | ------ | ------- | ----- |
| Go configuration    | ✅     | ✅   | ✅   | ❌   | ❌   | ✅     | ❌      | ❌    |
| Cross-platform      | ✅     | ⚠️   | ✅   | ✅   | ✅   | ✅     | ✅      | ✅    |
| Tool management     | ✅     | ✅   | ❌   | ❌   | ❌   | ✅     | ✅      | ✅    |
| Parallel execution  | ✅     | ✅   | ✅   | ✅   | ❌   | ✅     | ✅      | ✅    |
| Container isolation | ❌     | ❌   | ❌   | ❌   | ❌   | ✅     | ✅      | ⚠️    |
| Path filtering      | ✅     | ❌   | ❌   | ❌   | ❌   | ❌     | ❌      | ✅    |
| Auto-detection      | ✅     | ❌   | ❌   | ❌   | ❌   | ❌     | ❌      | ❌    |
| Remote caching      | ❌     | ❌   | ❌   | ❌   | ❌   | ✅     | ✅      | ✅    |
| CI portability      | ✅     | ✅   | ⚠️   | ⚠️   | ⚠️   | ✅     | ✅      | ✅    |
| Setup complexity    | Low    | Low  | Low  | Low  | Low  | Medium | Medium  | High  |

Legend: ✅ Full support | ⚠️ Partial/Limited | ❌ Not available

**CI portability notes:**

- Pocket and Sage auto-download Go if not installed — CI config is just `./pok`
  or `make`
- Dagger, Earthly, Bazel use containers or hermetic builds for full isolation
- Mage, Task, Just require pre-installing the runtime on CI

## Detailed Comparison

### Pocket

**What it is:** A cross-platform build system with built-in tool management.
Tasks are defined in Go code with `Serial()` and `Parallel()` execution control.

**Key features:**

- Cross-platform without platform-specific files (no Makefiles)
- Automatic tool downloading and caching in `.pocket/`
- Auto-detection of project types (Go modules, Python projects, etc.)
- Task groups that run consistently across all detected locations
- Path filtering for monorepos (`Paths().In()`, `AutoDetect()`)
- Bootstrap shim (`./pok`) that auto-installs Go if needed
- Bundled task packages for Go, Python, Lua, Markdown

**Trade-offs:**

- Go knowledge required for configuration
- Newer project, smaller ecosystem
- No container isolation (runs directly on host)

**Example:**

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        // Auto-detect finds all go.mod and pyproject.toml locations
        // and runs these task groups in each
        pocket.AutoDetect(golang.Tasks()),
        pocket.AutoDetect(python.Tasks()),
    ),
}
```

---

### Sage (einride/sage)

**What it is:** A Make-like build tool inspired by Mage, maintained by Einride.
Generates Makefiles from Go code for IDE integration.

**Key features:**

- Curated set of tools for Go projects
- Generates Makefiles for discoverability
- Auto-downloads Go if not installed
- Dependabot integration out of the box
- Well-maintained tool ecosystem

**Trade-offs:**

- Requires GNU Make (Windows compatibility concerns)
- Tied to Einride's tooling opinions
- Makefile generation adds complexity

**Comparison to Pocket:** Pocket was inspired by Sage. Both auto-download Go if
not installed, enabling zero-setup CI. They differ in:

- No Makefile generation (pure Go execution)
- Cross-platform shim without Make dependency
- Path filtering and auto-detection for monorepos

**Example:**

```go
// .sage/main.go
func Build(ctx context.Context) error {
    sg.Deps(ctx, Format, Lint)
    return sg.Command(ctx, "go", "build", "./...").Run()
}
```

---

### Mage

**What it is:** A Make/rake-like build tool using plain Go functions as targets.

**Key features:**

- No dependencies other than Go
- Full access to Go ecosystem (any library is a "plugin")
- Automatic dependency handling between targets
- Parallel execution via goroutines

**Trade-offs:**

- No built-in tool management (DIY)
- Less opinionated (more setup required)
- Learning curve for Make users

**Comparison to Pocket:**

- Mage is more minimal; Pocket provides more batteries-included features
- Pocket has built-in tool management; Mage requires manual setup
- Pocket has explicit `Serial()`/`Parallel()` vs Mage's `mg.Deps()`

**Example:**

```go
// magefile.go
func Build() error {
    mg.Deps(Format, Lint)
    return sh.Run("go", "build", "./...")
}
```

---

### Task (go-task)

**What it is:** A task runner with YAML configuration, aiming for simpler syntax
than Make.

**Key features:**

- YAML-based configuration (Taskfile.yml)
- Checksum-based dependency detection (not timestamps)
- Cross-platform support
- Built-in support for dotenv files

**Trade-offs:**

- YAML limitations for complex logic
- No built-in tool management
- Less type safety than Go-based tools

**Comparison to Pocket:**

- Task uses YAML; Pocket uses Go code
- Task has checksum caching; Pocket currently doesn't
- Both are cross-platform

**Example:**

```yaml
# Taskfile.yml
version: "3"
tasks:
  build:
    deps: [format, lint]
    cmds:
      - go build ./...
```

---

### Just

**What it is:** A command runner with Makefile-inspired syntax, focused on
simplicity.

**Key features:**

- Simple, familiar syntax
- Cross-platform (written in Rust)
- Recipe parameters and variables
- Shell-agnostic command running

**Trade-offs:**

- No file dependency checking (command runner, not build tool)
- No built-in tool management
- Limited programmability

**Comparison to Pocket:**

- Just is a command runner; Pocket is a build system
- Just has no dependency tracking; Pocket has execution ordering
- Just is simpler but less powerful

**Example:**

```just
# justfile
build: format lint
    go build ./...
```

---

### Dagger

**What it is:** A programmable CI/CD engine that runs pipelines in containers.
Created by Solomon Hykes (Docker co-founder).

**Key features:**

- Pipelines as code (Go, Python, TypeScript SDKs)
- Container-based execution (portable across CI systems)
- Built-in caching and artifact management
- LLM integration for AI-powered workflows
- Run locally or in any CI

**Trade-offs:**

- Requires container runtime (Docker/Podman)
- More complex setup than simple task runners
- Heavier resource usage

**Comparison to Pocket:**

- Dagger is CI/CD focused; Pocket is build-focused
- Dagger uses containers; Pocket runs directly on host
- Dagger is more portable across CI systems
- Pocket is lighter weight for simple builds

**Example:**

```go
func (m *MyPipeline) Build(ctx context.Context) error {
    return dag.Container().
        From("golang:1.22").
        WithDirectory("/src", dag.Host().Directory(".")).
        WithWorkdir("/src").
        WithExec([]string{"go", "build", "./..."}).
        Sync(ctx)
}
```

---

### Earthly

**What it is:** A build framework combining Dockerfile and Makefile concepts.

**Key features:**

- Familiar syntax (Dockerfile + Makefile hybrid)
- Container-based reproducibility
- Parallel execution and caching
- Works with existing language tools (npm, pip, go build)

**Trade-offs:**

- Requires container runtime
- Learning Earthfile syntax
- Not byte-for-byte reproducible (unlike Bazel)

**Comparison to Pocket:**

- Earthly uses containers; Pocket runs on host
- Earthly has broader language support via containers
- Pocket is simpler when you don't need container isolation

**Example:**

```dockerfile
# Earthfile
VERSION 0.8
build:
    FROM golang:1.22
    COPY . .
    RUN go build ./...
    SAVE ARTIFACT ./myapp
```

---

### Bazel

**What it is:** Google's build system for large-scale monorepos with hermetic
builds.

**Key features:**

- Truly reproducible builds (byte-for-byte)
- Massive parallelization and caching
- Remote execution support
- Language-agnostic (with rules)

**Trade-offs:**

- Steep learning curve (Starlark DSL)
- Significant setup investment
- Replaces language-native tools (not a wrapper)
- Overkill for small projects

**Comparison to Pocket:**

- Bazel is enterprise-scale; Pocket is project-scale
- Bazel replaces `go build`; Pocket wraps it
- Bazel takes months to adopt; Pocket takes minutes
- Bazel has remote caching; Pocket is local-only

**When to consider Bazel:**

- Very large monorepos (100+ developers)
- Need byte-for-byte reproducibility
- Remote build execution requirements

---

### Make

**What it is:** The classic Unix build tool, ubiquitous but showing its age.

**Key features:**

- Universal availability on Unix systems
- Well-understood by most developers
- File-based dependency tracking
- Extensive documentation

**Trade-offs:**

- Poor Windows support
- Arcane syntax (tabs matter!)
- No built-in tool management
- Limited programmability

**Comparison to Pocket:**

- Make requires shell scripting; Pocket uses Go
- Make has platform issues; Pocket is cross-platform
- Make is everywhere; Pocket needs bootstrap

---

## Decision Guide

### Choose Pocket if you:

- Want cross-platform builds without Makefiles
- Need automatic tool management
- Have a monorepo with path-based task filtering
- Prefer Go code over YAML/DSL configuration

### Choose Sage if you:

- Are in the Einride ecosystem
- Want Makefile integration for IDE support
- Need a curated, maintained tool set

### Choose Mage if you:

- Want maximum flexibility with minimal opinions
- Are comfortable setting up your own tooling
- Need something battle-tested and stable

### Choose Task if you:

- Prefer YAML configuration
- Want something simpler than Make
- Need checksum-based caching

### Choose Dagger if you:

- Need portable CI/CD pipelines
- Want container isolation
- Are building complex multi-stage pipelines

### Choose Earthly if you:

- Need containerized builds
- Like Dockerfile syntax
- Work across multiple languages

### Choose Bazel if you:

- Have a very large monorepo
- Need hermetic, reproducible builds
- Can invest significant setup time

---

## Sources

- [Mage](https://magefile.org/)
- [Sage (einride)](https://github.com/einride/sage)
- [Task](https://taskfile.dev/)
- [Just](https://github.com/casey/just)
- [Dagger](https://dagger.io/)
- [Earthly](https://earthly.dev/)
- [Bazel](https://bazel.build/)
- [Go Build Tools Comparison](https://www.amazingcto.com/go-build-tools-make-bazel-just-xc-taskfile-mage/)
- [When to use Bazel?](https://earthly.dev/blog/bazel-build/)

---

_This comparison was generated with the help of Claude Opus 4.5 on 2026-01-12.
Information may become outdated as these tools evolve._
