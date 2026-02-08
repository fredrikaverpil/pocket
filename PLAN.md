# Plan

- [ ] Do we have enough tests (AAA pattern) so that we avoid breaking
      functionality?
  - [ ] Measure code coverage and inspect non-covered parts and whether we
        should cover with tests.
  - [ ] Can we create complete test scenarios with composed Config.Auto setups?
  - [ ] Can we end-to-end test the bootstrapper (`pocket init`)?
- [x] Add [zensical](https://github.com/zensical/zensical) as tool into the
      "tools" package, with flags -serve and -build. For this tool, we want to
      maintain the version string in Pocket (with Renovate-ability to bump), and
      it would be ideal if we simply abstract away the whole Python setup and
      need of a .venv from projects.
- [ ] Analyze Pocket
  - [ ] DX - do we have good developer experience?
  - [ ] From a DX perspective; is the API surface easy to understand?
  - [ ] Long-term maintainability, is the codebase simple and ideomatic to Go?
  - [ ] From a files/packages perspective; is the git project laid out well?
  - [ ] From a Go ideomatic view; is the project following Go ideoms, leveraging
        std lib, easy to understand?
  - [ ] Compare with pocket-v1 (~/code/public/pocket-v1); which areas have been
        improved, which areas were done better/simpler in pocket-v1?
- [ ] Verify that all documentation is up to date. We need to mention all public
      API methods and configurable parts.
- [ ] We are in documentation sometimes distinguishing "end-users" and
      "task/tool authors". These are for the most part going to be the same
      person. The difference is really that one person might build tasks/tools
      and then reuse them in multiple projects. We should make that more clear
      in the different markdown documentation files we have.
- [ ] We keep documentation on reference, architecture, user guides. Is there
      overlap and/or risk of updating one but not the other and then cause a
      drift where the documentation is not aligned? Can we consolidate the
      documentation better, and here I'm actually thinking alot about LLMs
      finding one place where documentation needs updating and doesn't see the
      other file which also needs updating.
- [ ] Explore Nix as a package manager backend for tool installation. See
      [Nix exploration notes](#nix-as-a-package-manager-backend) below.

## Nix as a package manager backend

Treat Nix as another package manager alongside `bun`, `uv`, `golang`, and
`download`. When Nix is available on the host, tool packages could delegate
installation to Nix instead of implementing per-tool download/platform logic.

### Core idea

A `tools/nix/` package provides an `Install(pkg)` helper, analogous to how
`tools/bun/` provides `bun.EnsureInstalled()` or `tools/golang/` provides
`golang.Install()`. Tool packages become one-liners:

```go
// tools/golangcilint/golangcilint.go
var Install = nix.Install("golangci-lint")
```

Pocket runs `nix build`, symlinks the result into `.pocket/bin/`, done. No
platform-specific URL construction, no archive extraction, no architecture
conversion helpers.

### Version pinning

Nix versions are pinned via a `flake.lock` (analogous to `bun.lock` /
`uv.lock`). The project has a `flake.nix` that declares which packages to
expose, and `flake.lock` pins the exact nixpkgs revision.

| Backend    | Manifest         | Lock file    | Version model            |
| ---------- | ---------------- | ------------ | ------------------------ |
| `download` | URL with version | —            | Explicit per-tool        |
| `golang`   | `pkg@version`    | —            | Explicit per-tool        |
| `bun`      | `package.json`   | `bun.lock`   | Explicit per-tool        |
| `uv`       | `pyproject.toml` | `uv.lock`    | Explicit per-tool        |
| **`nix`**  | `flake.nix`      | `flake.lock` | Implicit via nixpkgs pin |

**Recommended approach: single nixpkgs pin.** All tools share one nixpkgs
revision. Versions move together as a coherent set (this is how nixpkgs is
tested). Renovate already has a Nix manager that can update `flake.lock`.

Per-tool nixpkgs pins (multiple flake inputs) are possible but add complexity
and defeat Nix's "coherent set" strength. Tools that need a specific version can
still use another backend.

### File layout

```
tools/nix/          ← package manager package (like tools/bun/, tools/uv/)
  nix.go            ← Install() helper, nix build + symlink logic
flake.nix           ← declares packages (project root or .pocket/)
flake.lock          ← pins nixpkgs revision (auto-managed by nix)
```

### CI bootstrapping caveat

Pocket's core value is "take CI local" — only Go is needed, everything else
self-bootstraps. The current backends (download, golang, bun, uv) all work
because Pocket can fetch and install them from Go alone.

Nix breaks this property. Nix installation is invasive (needs `/nix/store`, a
daemon on Linux) and is not a single binary Pocket can drop into
`.pocket/tools/`. This means Nix as a backend **only makes sense for projects
that already have Nix in their CI** (or are willing to add it).

CI platform support varies:

- **GitHub Actions**: well-supported via
  `DeterminateSystems/nix-installer-action` (~30s install overhead).
- **Forgejo/Codeberg/Tangled**: less mature Nix support, often self-hosted
  runners where Nix must be installed manually.
- **Nix-native CI** (Garnix, Hercules CI): Nix is already there.

This reinforces that the Nix backend should be **opt-in per tool**, not a
default strategy. Projects without Nix continue using the existing backends with
zero external dependencies. The positioning: "If your CI already has Nix, Pocket
can leverage it to eliminate tool installation boilerplate."

### What to explore

- Can `nix build` output be symlinked into `.pocket/bin/` reliably across
  platforms (Linux, macOS)?
- What is the cold-start performance of `nix build` for a typical tool vs the
  current download approach?
- Should Pocket generate the `flake.nix` from tool declarations, or should the
  user maintain it?
- How does this interact with NixOS users who may already have tools via their
  system config?
- Could Pocket detect Nix availability and auto-select the Nix backend, or
  should it be an explicit opt-in per tool?
- Evaluate `nix shell` (ephemeral, no install step) vs `nix build` + symlink
  (persistent, matches current model) as the execution strategy.
