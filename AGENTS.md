# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>        # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

---

# Omnidist - Omni-platform Binary Distribution Toolkit

A repeatable way to build, package, and publish a Go CLI as an npm-installable tool.

## Quick Reference

```bash
omnidist init       # Bootstrap npm workspace packaging in existing Go repo
omnidist build      # Compile Go binaries for configured targets
omnidist stage      # Assemble npm packages from built artifacts
omnidist verify     # Enforce correctness before publishing
omnidist publish    # Publish to npm registry
```

## Product Overview

**Purpose**: Provide a repeatable way to build, package, and publish a Go CLI as an npm-installable tool where:
- Users install once: `npm i -g <pkg>`
- The correct binary is available immediately
- No install-time scripts (postinstall / downloaders) are used
- Cross-platform support via platform packages selected through npm's os/cpu constraints

**Primary Users**:
- Maintainers of Go CLIs who want distribution via npm
- Teams distributing internal CLIs using private npm registries

**Key Non-Functional Requirements**:
- Reproducible builds
- Strict version parity between Go release and npm packages
- No network fetch on install beyond npm registry tarballs
- Clear error messages when platform is unsupported

## Packaging Model

### Meta Package
- Name: `<pkg>` (e.g., `mytool` or `@scope/mytool`)
- Contains Node entrypoint shim (tiny)
- Contains `optionalDependencies` listing all platform packages at the same version
- Must contain no `scripts.postinstall`

### Platform Packages
- Name pattern: `<pkg>-<os>-<cpu>` (e.g., `@scope/mytool-linux-x64`)
- Contains prebuilt Go binary at `bin/<name>` or `bin/<name>.exe`
- Declares `os` and `cpu` in package.json
- Must contain no npm scripts

### Selection Behavior
npm installs the meta package, then attempts optionalDependencies. Only the matching platform package is installed due to os/cpu constraints.

## Supported Targets (Default Matrix)

| OS      | CPU  |
|---------|------|
| darwin  | x64  |
| darwin  | arm64 |
| linux   | x64  |
| linux   | arm64 |
| win32   | x64  |

Default: `CGO_ENABLED=0` for portability.

## Repository Layout

```
repo/
  cmd/<name>/           # Go CLI main package
  npm/
    <pkg>/              # meta package
    <pkg>-darwin-arm64/
    <pkg>-darwin-x64/
    <pkg>-linux-arm64/
    <pkg>-linux-x64/
    <pkg>-win32-x64/
  dist/
    darwin/arm64/<name>
    darwin/x64/<name>
    linux/arm64/<name>
    linux/x64/<name>
    win32/x64/<name>.exe
  omnidist.yaml         # configuration
```

## Configuration Spec

`omnidist.yaml` at repo root:

```yaml
tool:
  name: <binary name>
  main: ./cmd/<name>

npm:
  package: @scope/<pkg>
  registry: https://registry.npmjs.org
  access: public

version:
  source: git-tag  # git-tag | file | env

targets:
  - os: darwin
    cpu: x64
  - os: darwin
    cpu: arm64
  - os: linux
    cpu: x64
  - os: linux
    cpu: arm64
  - os: win32
    cpu: x64

build:
  ldflags: -s -w
  tags: []
  cgo: false
```

## Toolkit CLI Surface

The toolkit uses [Cobra](https://github.com/spf13/cobra) for CLI implementation.

| Command | Description |
|---------|-------------|
| `omnidist init` | Bootstrap npm workspace packaging |
| `omnidist build` | Compile Go binaries for targets |
| `omnidist stage` | Assemble npm packages from artifacts |
| `omnidist verify` | Enforce correctness before publish |
| `omnidist publish` | Publish to npm registry |

## Runtime Behavior (Meta Shim)

The shim must:
1. Determine `process.platform` and `process.arch`
2. Map to expected platform package name
3. Resolve that package's installation path
4. Derive binary path inside it
5. Execute binary with argument/stdio/exit code passthrough
6. Support Windows `.exe` resolution

## Error UX

If platform package missing, message includes:
- Detected platform/arch
- Expected package name
- Common causes and next steps

## Versioning & Release Flow

1. Tag repo: `git tag v1.0.0`
2. CI runs:
   - `omnidist build`
   - `omnidist stage`
   - `omnidist verify`
   - `omnidist publish`

All packages share identical semver.

## MVP Acceptance Criteria

- `npm i -g <pkg>` works on: macOS arm64+x64, Linux arm64+x64, Windows x64
- Installed command runs: `<name> --version`
- No npm scripts in any published package
- `omnidist verify` passes on CI
- Version parity enforced across all packages

## Backlog (Post-MVP)

- linux musl split targets (linuxmusl-*)
- SBOM generation
- Signature / provenance (cosign)
- Homebrew / Scoop release generation
- Auto-generated changelog
- GitHub Actions templates
