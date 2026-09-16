# Agent instructions

## Session context and tracking

This repository uses Beads 1.1 for all durable issue tracking. At the beginning of a session, run:

```bash
bd prime
```

Follow the workflow and git authority printed by `bd prime`; it is the current source of truth. Do not invent a second task list in Markdown and do not assume permission to commit, pull, rebase, or push.

Common read/write operations are:

```bash
bd ready --json
bd show <id> --json
bd update <id> --status in_progress --json
bd create "Title" --description="Context" -t bug -p 1 --deps discovered-from:<id> --json
bd close <id> --reason "Completed" --json
```

Keep discovered follow-up work in Beads and link it with `discovered-from`. Preserve unrelated worktree and Beads state.

## Product invariants

Omnidist builds a Go CLI and distributes prebuilt binaries through npm, uv/PyPI-compatible indexes, and RubyGems.

- The release sequence is build → stage → verify → publish.
- `enabled-distributions` is a non-empty subset of `npm`, `uv`, and `gem`. If absent in an older config, all three are enabled.
- Aggregate commands and generated CI use enabled backends in npm → uv → gem order. `--only` may narrow, never enable a disabled backend.
- Aggregate publish preflights every selected backend before upload. Registry publication remains non-transactional and cannot be rolled back atomically.
- npm packages must contain no `postinstall` script or install-time downloader.
- All backend artifacts in one release derive from the build version file.
- Existing valid profile and legacy configs remain supported.

## Configuration facts

The default path is `.omnidist/omnidist.yaml`. New configs use `profiles.default` and artifacts under `.omnidist/default/`; legacy top-level configs use `.omnidist/` directly. Never mix profile and top-level runtime fields.

Targets use Go syntax:

```yaml
profiles:
  default:
    enabled-distributions: [npm, uv, gem]
    tool:
      name: mytool
      main: ./cmd/mytool
    targets:
      - os: windows
        arch: amd64
```

Use `os: windows`, `arch: amd64`, and the field name `arch`. npm mappings such as `win32/x64` belong only to generated package metadata.

`omnidist init` is non-destructive by default. `--force` replaces an existing config; use it only when explicitly intended. When `cmd/*` discovery is ambiguous, use `--name` and `--main`.

## CLI surface

```text
omnidist init [--force] [--name <name>] [--main <package>]
omnidist build
omnidist stage|verify|publish [--only npm,uv,gem]
omnidist publish [--dry-run]
omnidist ci [--dry-run] [--force]
omnidist npm stage|verify|publish|trust
omnidist uv stage|verify|publish
omnidist gem stage|verify|publish
```

Use command help and the current code as authority for detailed flags. User references live in `docs/configuration.md`, `docs/releases.md`, and `docs/targets.md`.

## Repository map and boundaries

```text
cmd/omnidist/                 CLI orchestration
internal/config/              Config/default/validation model
internal/paths/               Profile and legacy layouts
internal/workflow/            Shared build/init/CI workflows
internal/workflow/{npm,uv,gem}/ Backend implementations
docs/                         User references
```

Do not modify `pack/callee/**`. Do not overwrite user changes. Use `apply_patch` for edits and keep generated artifacts out of source changes unless regeneration is the task.

## Verification

Run focused tests during implementation. Before completing code changes, run:

```bash
go test -race ./...
go tool golangci-lint run --timeout=5m
git diff --check
```

Documentation or config examples should have focused drift tests where practical. Do not perform real registry publication as verification; use unit fakes and `--dry-run`.
