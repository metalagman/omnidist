# omnidist

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/omnidist)](https://goreportcard.com/report/github.com/metalagman/omnidist)
[![lint](https://github.com/metalagman/omnidist/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/omnidist/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/test.yml)
[![codecov](https://codecov.io/github/metalagman/omnidist/graph/badge.svg)](https://codecov.io/github/metalagman/omnidist)
[![version](https://img.shields.io/github/v/release/metalagman/omnidist?sort=semver)](https://github.com/metalagman/omnidist/releases)
[![npm](https://img.shields.io/npm/v/%40omnidist%2Fomnidist)](https://www.npmjs.com/package/@omnidist/omnidist)
[![PyPI](https://img.shields.io/pypi/v/omnidist)](https://pypi.org/project/omnidist/)
[![license](https://img.shields.io/github/license/metalagman/omnidist)](LICENSE)

Run your Go CLI everywhere with `npx` and `uvx`, without requiring Go on end-user machines.

`omnidist` turns one Go project into cross-platform npm and uv distributions with prebuilt binaries, then stages, verifies, and publishes them in a deterministic release flow.

Release flow: `build -> stage -> verify -> publish` so users can run your tool from JavaScript and Python ecosystems out of the box.

For project background, packaging model details, migration notes, and contributor-oriented repo layout, see [CONTRIBUTING.md](CONTRIBUTING.md).

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

1. Print repo-tailored onboarding/release commands:

```bash
omnidist quickstart
```

2. Initialize config and distribution folder structure:

```bash
omnidist init
```

This creates:
- `.omnidist/omnidist.yaml`
- `.omnidist/` workspace directories
- `.omnidist/.gitignore` for generated artifacts

3. Build binaries for configured targets:

```bash
omnidist build
```

This also writes the resolved build version to `.omnidist/dist/VERSION`.

4. Stage and verify artifacts:

```bash
omnidist stage
omnidist verify
```

`omnidist uv stage` converts the resolved version to PEP 440 and writes
`.omnidist/uv/pyproject.toml` with that version.
It also recreates `.omnidist/uv/dist` to prevent stale wheel artifacts from previous runs.

5. Publish when verification passes:

```bash
omnidist publish
```

6. Generate tag-triggered release workflow:

```bash
omnidist ci
```

## Common Commands

```bash
# Build binaries for configured targets and persist build version
omnidist build

# Print a quickstart command sequence for this repo
omnidist quickstart

# Show runtime version/build metadata
omnidist version

# Stage and verify both distributions (npm -> uv)
omnidist stage
omnidist verify

# Stage dev/pre-release artifacts
omnidist stage --dev

# Publish both distributions (fail-fast, npm -> uv)
omnidist publish

# Generate GitHub Actions workflow for tagged releases
omnidist ci

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

- `OMNIDIST_VERSION`: used when `version.source: env`; also expanded in `build.ldflags` templates (for example `${OMNIDIST_VERSION}`).
  `VERSION` is not used.
- `OMNIDIST_GIT_COMMIT`: optional ldflags template variable for build metadata; populated automatically by `omnidist build` when git metadata is available.
- `OMNIDIST_BUILD_DATE`: optional ldflags template variable for build metadata; populated automatically by `omnidist build` as UTC RFC3339.
- `NPM_PUBLISH_TOKEN`: required for npm publish commands when not using `--dry-run`
- `UV_PUBLISH_TOKEN`: used by uv publish when `--token` is not provided

Example `.env`:

```dotenv
OMNIDIST_VERSION=1.2.3
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
  - os: windows
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
    include-readme: true # include project README.md in staged packages when present

  uv:
    package: omnidist
    index-url: https://upload.pypi.org/legacy/
    linux-tag: manylinux2014 # manylinux2014 | musllinux_1_2
    include-readme: true # include project README.md in staged wheels when present
```

`targets` use Go values (`GOOS`/`GOARCH`). Distribution workflows map them as needed (for example `windows/amd64` -> npm `win32/x64`).

For appkit version injection, configure `build.ldflags` in your project config:

```yaml
build:
  ldflags: -s -w -X github.com/metalagman/appkit/version.version=${OMNIDIST_VERSION} -X github.com/metalagman/appkit/version.gitCommit=${OMNIDIST_GIT_COMMIT} -X github.com/metalagman/appkit/version.buildDate=${OMNIDIST_BUILD_DATE}
```

With `version.source: git-tag`, release workflows require `HEAD` to be on an exact SemVer tag (`vX.Y.Z` or `X.Y.Z`).

## Command Reference

Top-level:

- `omnidist init`
- `omnidist build`
- `omnidist quickstart`
- `omnidist version`
- `omnidist ci [--force]`
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

### CI bootstrap for tag releases

Generate `.github/workflows/omnidist-release.yml`:

```bash
omnidist ci
```

The generated workflow triggers on `v*` tag pushes and runs:
`build -> stage -> verify -> publish`.

If workflow already exists:

```bash
omnidist ci --force
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
export OMNIDIST_VERSION=2.0.0
omnidist npm stage
omnidist uv stage
```
