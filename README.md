# omnidist

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/omnidist)](https://goreportcard.com/report/github.com/metalagman/omnidist)
[![lint](https://github.com/metalagman/omnidist/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/omnidist/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/test.yml)
[![go](https://img.shields.io/github/go-mod/go-version/metalagman/omnidist)](https://github.com/metalagman/omnidist/blob/master/go.mod)
[![version](https://img.shields.io/github/v/release/metalagman/omnidist?sort=semver)](https://github.com/metalagman/omnidist/releases)
[![license](https://img.shields.io/github/license/metalagman/omnidist)](LICENSE)

`omnidist` is a Go toolkit for distributing a Go CLI through npm and uv as prebuilt platform artifacts.

It builds binaries for multiple targets, stages distribution artifacts, verifies integrity, and publishes to registries.

## Why

- Install once with npm: `npm i -g <package>`
- Publish Python wheel artifacts to PyPI-compatible indexes with uv
- No install-time download scripts for npm
- Reproducible release flow from a single config file

## How It Works

`omnidist` supports two additive backends:

- `npm`:
  - meta package (for example `@scope/tool`) with shim and `optionalDependencies`
  - platform packages (for example `@scope/tool-linux-x64`) with prebuilt binaries
- `uv`:
  - per-target platform wheel artifacts in `.omnidist/uv/dist/`
  - one wheel per configured target with embedded binary in `<pkg>/bin/`

## Requirements

- Go 1.25+
- Node.js + npm (for npm distribution commands)
- `uv` (for uv distribution commands)
- `git` (when `version.source: git-tag`)
- `NPM_PUBLISH_TOKEN` for npm publish (unless `--dry-run`)
- `UV_PUBLISH_TOKEN` (or `--token`) for uv publish (unless `--dry-run`)

## Installation

Run without installation first:

```bash
npx @omnidist/omnidist --help
uvx omnidist --help
```

Install globally with npm:

```bash
npm i -g @omnidist/omnidist
omnidist --help
```

Install with Go toolchain:

```bash
go install github.com/metalagman/omnidist/cmd/omnidist@latest
omnidist --help
```

Build locally from source:

```bash
go build -o ./bin/omnidist ./cmd/omnidist
./bin/omnidist --help
```

Or run directly:

```bash
go run ./cmd/omnidist --help
```

## Quick Start

1. Initialize config and distribution folder structure:

```bash
omnidist init
```

This creates:
- `.omnidist/omnidist.yaml`
- `.omnidist/` workspace directories
- `.omnidist/.gitignore` for generated artifacts

2. Build binaries for configured targets:

```bash
omnidist build
```

This also writes the resolved build version to `.omnidist/dist/VERSION`.

3. Stage and verify artifacts:

```bash
omnidist stage
omnidist verify
```

`omnidist uv stage` converts the resolved version to PEP 440 and writes
`.omnidist/uv/pyproject.toml` with that version.
It also recreates `.omnidist/uv/dist` to prevent stale wheel artifacts from previous runs.

4. Publish when verification passes:

```bash
omnidist publish
```

## Common Commands

```bash
# Build binaries for configured targets and persist build version
omnidist build

# Show runtime version/build metadata
omnidist version

# Stage and verify both distributions (npm -> uv)
omnidist stage
omnidist verify

# Stage dev/pre-release artifacts
omnidist stage --dev

# Publish both distributions (fail-fast, npm -> uv)
omnidist publish

# Limit orchestration to one distribution
omnidist stage --only npm
omnidist verify --only uv

# Distribution-specific publishing options
omnidist npm publish --tag next --otp <6-digit-code>
omnidist uv publish --publish-url https://test.pypi.org/legacy/ --token <pypi-token>
```

## Environment Variables and .env

`omnidist` loads `.env` automatically at startup (via `godotenv`) if present.

Supported variables:

- `VERSION`: used when `version.source: env`
- `NPM_PUBLISH_TOKEN`: required for npm publish commands when not using `--dry-run`
- `UV_PUBLISH_TOKEN`: used by uv publish when `--token` is not provided

Example `.env`:

```dotenv
VERSION=1.2.3
NPM_PUBLISH_TOKEN=npm_xxx
UV_PUBLISH_TOKEN=pypi-xxx
```

## Configuration

`.omnidist/omnidist.yaml`:

```yaml
tool:
  name: omnidist
  main: ./cmd/omnidist

version:
  source: git-tag # git-tag | file | env

targets:
  - os: darwin
    arch: amd64
  - os: darwin
    arch: arm64
  - os: linux
    arch: amd64
  - os: linux
    arch: arm64
  - os: win32
    arch: amd64

build:
  ldflags: -s -w
  tags: []
  cgo: false

distributions:
  npm:
    package: "@omnidist/omnidist"
    registry: https://registry.npmjs.org
    access: public # public | restricted

  uv:
    package: omnidist
    index-url: https://upload.pypi.org/legacy/
    linux-tag: manylinux2014 # manylinux2014 | musllinux_1_2
```

With `version.source: git-tag`, release workflows require `HEAD` to be on an exact SemVer tag (`vX.Y.Z` or `X.Y.Z`).

## Command Reference

Top-level:

- `omnidist init`
- `omnidist build`
- `omnidist version`
- `omnidist stage [--dev] [--only npm|uv|npm,uv]`
- `omnidist verify [--only npm|uv|npm,uv]`
- `omnidist publish [--dry-run] [--only npm|uv|npm,uv]`
- `omnidist npm`
- `omnidist uv`

NPM subcommands:

- `omnidist npm stage [--dev]`
- `omnidist npm verify`
- `omnidist npm publish [--dry-run] [--tag <tag>] [--registry <url>] [--otp <code>]`

UV subcommands:

- `omnidist uv stage [--dev]`
- `omnidist uv verify`
- `omnidist uv publish [--dry-run] [--publish-url <url>] [--token <pypi-token>]`

## Usage Patterns

### Local development loop

Use this when iterating on the CLI binary and validating artifact generation locally:

```bash
omnidist build
omnidist stage
omnidist verify
```

### Dev pre-release artifacts

Generate prerelease versions from git describe data:

```bash
omnidist stage --dev
```

### Unified multi-distribution orchestration

Top-level `stage`, `verify`, and `publish` run distributions in deterministic order:
`npm` first, then `uv`, and stop on first failure.

Select a subset with `--only`:

```bash
omnidist stage --only uv
omnidist verify --only npm
omnidist publish --dry-run --only npm,uv
```

### npm publishing flow with custom options

```bash
omnidist npm publish --dry-run --tag next --registry https://registry.npmjs.org
```

Before npm commands run, omnidist writes `.omnidist/.npmrc` from `distributions.npm.registry` using:
`//<registry>/:_authToken=${NPM_PUBLISH_TOKEN}`.
If staged package version contains a `-dev` prerelease and `--tag` is not provided, omnidist auto-publishes with `--tag dev`.

If your npm account requires 2FA for publish operations:

```bash
omnidist npm publish --otp <6-digit-code>
```

### uv publishing flow with custom index/auth

```bash
omnidist uv publish --publish-url https://upload.pypi.org/legacy/ --token <pypi-token>
```

`omnidist uv publish` uses token authentication.  
Provide token via `--token` or `UV_PUBLISH_TOKEN` (required for non-dry-run).
`omnidist uv verify` and `omnidist uv publish` use the staged version from
`.omnidist/uv/pyproject.toml` when present.
For PyPI/TestPyPI, `omnidist uv verify` fails if the staged version contains local metadata (`+...`), since those indexes reject local versions.

TestPyPI dry-run style validation:

```bash
omnidist uv publish --dry-run --publish-url https://test.pypi.org/legacy/
```

## Usage Examples

### npm release path

```bash
git tag v1.2.0
omnidist build
omnidist npm stage
omnidist npm verify
omnidist npm publish
```

### uv release path

```bash
git tag v1.2.0
omnidist build
omnidist uv stage
omnidist uv verify
omnidist uv publish --publish-url https://upload.pypi.org/legacy/
```

### uv dry-run publish

```bash
omnidist uv publish --dry-run --publish-url https://test.pypi.org/legacy/
```

### version from environment

```yaml
version:
  source: env
```

```bash
export VERSION=2.0.0
omnidist npm stage
omnidist uv stage
```

## Migration Guide (npm -> dual backend)

1. Pull latest `omnidist` and run `omnidist init` in a clean branch to get uv defaults in config.
2. Keep existing `distributions.npm` unchanged.
3. Add/update `distributions.uv` values:
   - `package` for wheel distribution name
   - `index-url` for target registry
   - `linux-tag` policy (`manylinux2014` default)
4. Extend CI pipeline with uv stage/verify gates (see next section).
5. Release both backends in the same version cycle.

This is additive: npm support remains first-class and is not deprecated.

## CI and Release Flow (Dual Backend)

Recommended release sequence:

1. `omnidist build`
2. `omnidist stage`
3. `omnidist verify`
4. `omnidist publish`

For CI verification-only jobs, run steps 1-3.

When you need distribution-specific publish options (`npm --tag/--otp/--registry`, `uv --publish-url/--token`), use `omnidist npm ...` and `omnidist uv ...` subcommands directly.

## Project Layout

```text
cmd/omnidist/               CLI entrypoint and commands
internal/config/            Config model and YAML load/save
internal/workflow/          build/init/npm/uv workflows
.omnidist/omnidist.yaml     Project configuration
.omnidist/.gitignore        Ignore rules for generated artifacts
.omnidist/dist/             Built binaries by os/arch
.omnidist/dist/VERSION      Version captured at build time
.omnidist/npm/              Staged npm packages
.omnidist/uv/pyproject.toml UV staging project with PEP 440 version
.omnidist/uv/dist/          Staged wheel artifacts
```
