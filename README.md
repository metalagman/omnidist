# omnidist

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
- npm auth (`npm login`) before `omnidist npm publish`
- PyPI token auth before `omnidist uv publish` (or `--dry-run`)

## Installation

Install with Go toolchain (global binary):

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

3. Stage and verify npm artifacts:

```bash
omnidist npm stage
omnidist npm verify
```

4. Stage and verify uv wheel artifacts:

```bash
omnidist uv stage
omnidist uv verify
```

5. Publish when verification passes:

```bash
omnidist npm publish
omnidist uv publish
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

## Command Reference

Top-level:

- `omnidist init`
- `omnidist build`
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
omnidist npm stage
omnidist npm verify
omnidist uv stage
omnidist uv verify
```

### Dev pre-release artifacts

Generate prerelease versions from git describe data:

```bash
omnidist npm stage --dev
omnidist uv stage --dev
```

### npm publishing flow with custom options

```bash
omnidist npm publish --dry-run --tag next --registry https://registry.npmjs.org
```

Before npm commands run, omnidist writes `.omnidist/.npmrc` from `distributions.npm.registry` using:
`//<registry>/:_authToken=${NPM_TOKEN}`.

If your npm account requires 2FA for publish operations:

```bash
omnidist npm publish --otp <6-digit-code>
```

### uv publishing flow with custom index/auth

```bash
omnidist uv publish --publish-url https://upload.pypi.org/legacy/ --token <pypi-token>
```

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
2. `omnidist npm stage`
3. `omnidist npm verify`
4. `omnidist uv stage`
5. `omnidist uv verify`
6. `omnidist npm publish`
7. `omnidist uv publish`

For CI verification-only jobs, run steps 1-5.

## Project Layout

```text
cmd/omnidist/               CLI entrypoint and commands
internal/config/            Config model and YAML load/save
internal/workflow/          build/init/npm/uv workflows
.omnidist/omnidist.yaml     Project configuration
.omnidist/.gitignore        Ignore rules for generated artifacts
.omnidist/dist/             Built binaries by os/arch
.omnidist/npm/              Staged npm packages
.omnidist/uv/dist/          Staged wheel artifacts
```
