# Tool patterns — complete examples

## Pattern 1: Go package (`go install`)

The simplest pattern. Use for any tool installable via `go install`.

**Example: golangci-lint** (`tools/golangcilint/golangcilint.go`)

```go
package golangcilint

import (
    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tools/golang"
)

const Name = "golangci-lint"

// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.1.6"

var Install = &pk.Task{
    Name:   "install:golangci-lint",
    Usage:  "install golangci-lint",
    Body:   golang.Install("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
    Hidden: true,
    Global: true,
}
```

**Renovate datasource:** `datasource=go`

### When to use

- Tool is a Go module
- Available via `go install <pkg>@<version>`

### How it works

`golang.Install(pkg, version)` runs `go install` with `GOBIN` set to
`.pocket/tools/go/<pkg>/<version>/`, then symlinks the binary into
`.pocket/bin/`.

---

## Pattern 2: GitHub release binary

For pre-compiled binaries distributed via GitHub releases.

**Example: stylua** (`tools/stylua/stylua.go`)

```go
package stylua

import (
    "fmt"
    "path/filepath"

    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/pk/download"
)

const Name = "stylua"

// renovate: datasource=github-releases depName=JohnnyMorganz/StyLua
const Version = "2.3.1"

var Install = &pk.Task{
    Name:   "install:stylua",
    Usage:  "install stylua",
    Body:   installStylua(),
    Hidden: true,
    Global: true,
}

func installStylua() pk.Runnable {
    binDir := pk.FromToolsDir("stylua", Version, "bin")
    binaryName := pk.BinaryName("stylua")
    binaryPath := filepath.Join(binDir, binaryName)

    hostOS := pk.HostOS()
    hostArch := pk.HostArch()

    osName := hostOS
    if hostOS == pk.Darwin {
        osName = "macos"
    }
    archName := pk.ArchToX8664(hostArch)

    url := fmt.Sprintf(
        "https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
        Version, osName, archName,
    )

    return download.Download(url,
        download.WithDestDir(binDir),
        download.WithFormat("zip"),
        download.WithExtract(download.WithExtractFile(binaryName)),
        download.WithSymlink(),
        download.WithSkipIfExists(binaryPath),
    )
}
```

**Renovate datasource:** `datasource=github-releases`

### When to use

- Tool publishes pre-built binaries on GitHub releases
- No runtime dependency needed (standalone binary)

### Cross-platform URL mapping

Each project names its release assets differently. Build a `platformArch()`
helper. Common mappings:

```go
// Architecture helpers available in pk:
pk.ArchToX8664(arch)  // amd64 → x86_64, arm64 → aarch64
pk.ArchToX64(arch)    // amd64 → x64, arm64 → arm64

// OS name varies per project. Check the GitHub releases page:
// "darwin", "macos", "apple-darwin", etc.
// "linux", "unknown-linux-gnu", etc.
// "windows", "pc-windows-msvc", etc.
```

Example `platformArch()` function (from bun):

```go
func platformArch() string {
    switch pk.HostOS() {
    case pk.Darwin:
        if pk.HostArch() == pk.ARM64 {
            return "darwin-aarch64"
        }
        return "darwin-x64"
    case pk.Linux:
        if pk.HostArch() == pk.ARM64 {
            return "linux-aarch64"
        }
        return "linux-x64"
    case pk.Windows:
        return "windows-x64"
    default:
        return fmt.Sprintf("%s-%s", pk.HostOS(), pk.HostArch())
    }
}
```

### Download API options

```go
download.Download(url,
    download.WithDestDir(dir),              // extraction destination
    download.WithFormat("zip"),             // "tar.gz", "tar", "zip", "gz", ""
    download.WithExtract(                   // extraction options:
        download.WithExtractFile(name),     //   extract specific file
        download.WithRenameFile(src, dst),  //   extract and rename
        download.WithFlatten(),             //   flatten directory structure
    ),
    download.WithSymlink(),                 // symlink into .pocket/bin/
    download.WithSkipIfExists(path),        // idempotency check
    download.WithOutputName(name),          // for "gz" format
)
```

---

## Pattern 3: Python tool (via uv)

For Python tools installed into isolated virtual environments.

**Always prefer `pyproject.toml`/`uv.lock` (pattern 3b) for new tools.** It
provides reproducible, locked builds. Pattern 3a (`requirements.txt`) exists in
older tools but should not be used for new ones.

### 3a: Using requirements.txt (legacy)

**Example: mdformat** (`tools/mdformat/mdformat.go`)

```go
package mdformat

import (
    "context"
    "crypto/sha256"
    _ "embed"
    "encoding/hex"
    "os"
    "path/filepath"

    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tools/uv"
)

const Name = "mdformat"
const pythonVersion = "3.13"

//go:embed requirements.txt
var requirements []byte

func Version() string {
    h := sha256.New()
    h.Write(requirements)
    h.Write([]byte(pythonVersion))
    return hex.EncodeToString(h.Sum(nil))[:12]
}

var Install = &pk.Task{
    Name:   "install:mdformat",
    Usage:  "install mdformat",
    Body:   pk.Serial(uv.Install, installMdformat()),
    Hidden: true,
    Global: true,
}

func installMdformat() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        venvDir := pk.FromToolsDir("mdformat", Version())
        binary := uv.BinaryPath(venvDir, "mdformat")

        if _, err := os.Stat(binary); err == nil {
            return nil
        }

        if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
            return err
        }

        reqPath := filepath.Join(venvDir, "requirements.txt")
        if err := os.WriteFile(reqPath, requirements, 0o644); err != nil {
            return err
        }

        return uv.PipInstallRequirements(ctx, venvDir, reqPath)
    })
}

func Exec(ctx context.Context, args ...string) error {
    venvDir := pk.FromToolsDir("mdformat", Version())
    python := uv.BinaryPath(venvDir, "python")
    execArgs := append([]string{"-m", "mdformat"}, args...)
    return pk.Exec(ctx, python, execArgs...)
}
```

**Renovate:** Put version pins in `requirements.txt` with Renovate comments.

### 3b: Using pyproject.toml + uv.lock

**Example: zensical** (`tools/zensical/zensical.go`)

```go
package zensical

import (
    "context"
    _ "embed"
    "os"
    "path/filepath"
    "regexp"
    "sync"

    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tools/uv"
)

const Name = "zensical"

//go:embed pyproject.toml
var pyprojectTOML []byte

//go:embed uv.lock
var uvLock []byte

var (
    versionOnce sync.Once
    version     string
)

// Version extracts the version from pyproject.toml.
func Version() string {
    versionOnce.Do(func() {
        // renovate: datasource=pypi depName=zensical
        re := regexp.MustCompile(`"zensical==([^"]+)"`)
        if m := re.FindSubmatch(pyprojectTOML); len(m) > 1 {
            version = string(m[1])
        }
    })
    return version
}

var Install = &pk.Task{
    Name:   "install:zensical",
    Usage:  "install zensical",
    Body:   pk.Serial(uv.Install, installZensical()),
    Hidden: true,
    Global: true,
}

func installZensical() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        installDir := pk.FromToolsDir(Name, Version())
        binary := uv.BinaryPath(
            uv.VenvPath(installDir, ""),
            Name,
        )

        if _, err := os.Stat(binary); err == nil {
            return nil
        }

        if err := os.MkdirAll(installDir, 0o755); err != nil {
            return err
        }
        if err := os.WriteFile(
            filepath.Join(installDir, "pyproject.toml"), pyprojectTOML, 0o644,
        ); err != nil {
            return err
        }
        if err := os.WriteFile(
            filepath.Join(installDir, "uv.lock"), uvLock, 0o644,
        ); err != nil {
            return err
        }

        return uv.Sync(ctx, uv.SyncOptions{ProjectDir: installDir})
    })
}
```

Companion file `tools/zensical/pyproject.toml`:

```toml
[project]
name = "pocket-zensical"
version = "0.0.0"
requires-python = ">=3.11"
dependencies = ["zensical==0.0.21"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

Generate `uv.lock` by running `uv lock` in the tool directory.

### uv helper API

```go
uv.Install                                    // Task: ensures uv binary exists
uv.CreateVenv(ctx, venvDir, pythonVersion)     // create venv
uv.PipInstall(ctx, venvDir, pkg)               // pip install single package
uv.PipInstallRequirements(ctx, venvDir, path)  // pip install -r
uv.Sync(ctx, SyncOptions{...})                // install from pyproject.toml
uv.Run(ctx, RunOptions{...}, cmd, args...)    // run command in venv
uv.BinaryPath(venvDir, name)                   // path to binary in venv
uv.VenvPath(projectPath, pythonVersion)        // compute venv location
```

---

## Pattern 4: Node tool (via bun)

For Node.js tools installed with bun from a lockfile.

**Example: prettier** (`tools/prettier/prettier.go`)

```go
package prettier

import (
    "context"
    _ "embed"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"

    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tools/bun"
)

const Name = "prettier"

//go:embed package.json
var packageJSON []byte

//go:embed bun.lock
var lockfile []byte

var (
    versionOnce sync.Once
    version     string
)

func Version() string {
    versionOnce.Do(func() {
        var pkg struct {
            Dependencies map[string]string `json:"dependencies"`
        }
        if err := json.Unmarshal(packageJSON, &pkg); err == nil {
            version = pkg.Dependencies[Name]
        }
    })
    return version
}

var Install = &pk.Task{
    Name:   "install:prettier",
    Usage:  "install prettier",
    Body:   pk.Serial(bun.Install, installPrettier()),
    Hidden: true,
    Global: true,
}

func installPrettier() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        installDir := pk.FromToolsDir(Name, Version())
        binary := bun.BinaryPath(installDir, Name)

        if _, err := os.Stat(binary); err == nil {
            return nil
        }

        if err := os.MkdirAll(installDir, 0o755); err != nil {
            return err
        }
        if err := os.WriteFile(
            filepath.Join(installDir, "package.json"), packageJSON, 0o644,
        ); err != nil {
            return err
        }
        if err := os.WriteFile(
            filepath.Join(installDir, "bun.lock"), lockfile, 0o644,
        ); err != nil {
            return err
        }

        return bun.InstallFromLockfile(ctx, installDir)
    })
}

func Exec(ctx context.Context, args ...string) error {
    installDir := pk.FromToolsDir(Name, Version())
    return bun.Run(ctx, installDir, Name, args...)
}
```

Create `package.json` and generate `bun.lock` by running `bun install` in the
tool directory.

**Renovate:** Put version in `package.json` `dependencies`, Renovate updates it
automatically.

### bun helper API

```go
bun.Install                                 // Task: ensures bun binary exists
bun.InstallFromLockfile(ctx, dir)           // bun install --frozen-lockfile
bun.Run(ctx, installDir, pkgName, args...)  // run installed package
bun.BinaryPath(installDir, binaryName)      // path to binary in node_modules
```

### bun on Windows

Bun has limited Windows support. When building platform-aware download URLs for
bun itself, note that only `windows-x64` is available. Node tools running
through bun on Windows may need extra handling — test thoroughly.

---

## Ecosystem tools: uv and bun

`uv` and `bun` are dual-purpose: they are tools themselves (downloaded as GitHub
release binaries) and serve as ecosystems for other tools.

When creating a tool that depends on an ecosystem tool, chain installation:

```go
Body: pk.Serial(uv.Install, installMyPythonTool()),
Body: pk.Serial(bun.Install, installMyNodeTool()),
```

The `Global: true` flag on ecosystem tool Install tasks ensures they are only
installed once, even if multiple dependent tools reference them.

---

## Renovate datasource reference

| Source          | Renovate comment                                                      |
|-----------------|-----------------------------------------------------------------------|
| Go module       | `// renovate: datasource=go depName=github.com/org/repo`             |
| GitHub releases | `// renovate: datasource=github-releases depName=owner/repo`         |
| PyPI            | `// renovate: datasource=pypi depName=package-name`                  |
| npm             | Automatic via `package.json`                                          |

For GitHub releases with non-standard tag formats, use `extractVersion`:

```go
// renovate: datasource=github-releases depName=oven-sh/bun extractVersion=^bun-v(?<version>.*)$
const Version = "1.3.6"
```
