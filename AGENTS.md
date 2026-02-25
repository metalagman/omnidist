# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started, then `bd prime` for session workflow context.

## Quick Reference

```bash
bd prime              # Load Beads workflow context for this session
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
omnidist init            # Bootstrap npm workspace packaging in existing Go repo
omnidist build           # Compile Go binaries for configured targets
omnidist npm stage       # Assemble npm packages from built artifacts
omnidist npm verify      # Enforce correctness before publishing
omnidist npm publish     # Publish to npm registry
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
| `omnidist npm stage` | Assemble npm packages from artifacts |
| `omnidist npm verify` | Enforce correctness before publish |
| `omnidist npm publish` | Publish to npm registry |

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
   - `omnidist npm stage`
   - `omnidist npm verify`
   - `omnidist npm publish`

All packages share identical semver.

## MVP Acceptance Criteria

- `npm i -g <pkg>` works on: macOS arm64+x64, Linux arm64+x64, Windows x64
- Installed command runs: `<name> --version`
- No npm scripts in any published package
- `omnidist npm verify` passes on CI
- Version parity enforced across all packages

## Backlog (Post-MVP)

- linux musl split targets (linuxmusl-*)
- SBOM generation
- Signature / provenance (cosign)
- Homebrew / Scoop release generation
- Auto-generated changelog
- GitHub Actions templates


<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

bd automatically syncs with git:

- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

<!-- END BEADS INTEGRATION -->
