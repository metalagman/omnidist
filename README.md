# omnidist

`omnidist` is a Go toolkit for distributing a Go CLI through npm as prebuilt platform packages.

It builds binaries for multiple targets, stages npm packages (meta + platform-specific), verifies package integrity, and publishes to npm.

## Why

- Install once with npm: `npm i -g <package>`
- No install-time download scripts
- npm `os`/`cpu` constraints select the right platform package
- Reproducible release flow from a single config file

## How It Works

`omnidist` creates:

- A meta package (for example `@scope/tool`) that contains:
  - a tiny Node shim entrypoint
  - `optionalDependencies` pointing to all platform packages at the same version
- Platform packages (for example `@scope/tool-linux-x64`) that contain:
  - prebuilt binary in `bin/`
  - `os` and `cpu` constraints in `package.json`

At install time, npm installs the meta package and the matching platform package.

## Requirements

- Go 1.25+
- Node.js + npm
- `git` (when `version.source: git-tag`)
- npm auth (`npm login`) before `omnidist npm publish`

## npm Auth and 2FA

For CI/non-interactive publish, use an npm **Automation Token** (2FA bypass for publish).

Create token:

- `https://www.npmjs.com/settings/<username>/tokens`

Use token:

```bash
export NPM_TOKEN=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
npm config set //registry.npmjs.org/:_authToken "$NPM_TOKEN"
omnidist npm publish
```

If you publish with regular account auth and 2FA is enabled for writes, pass OTP:

```bash
omnidist npm publish --otp 123456
```

## Installation

Install with Go toolchain (global binary):

```bash
go install github.com/metalagman/omnidist/cmd/omnidist@latest
omnidist --help
```

Add as a project tool (`go get -tool`) and run via `go tool`:

```bash
go get -tool github.com/metalagman/omnidist/cmd/omnidist@latest
go tool omnidist --help
```

Run via `npx` (no global install):

```bash
npx -y @omnidist/omnidist --help
```

Install globally with npm:

```bash
npm i -g @omnidist/omnidist
omnidist --help
```

Build locally from source:

```bash
go build -o omnidist ./cmd/omnidist
./omnidist --help
```

Or run directly:

```bash
go run ./cmd/omnidist --help
```

## Quick Start

1. Initialize config and npm folder structure:

```bash
omnidist init
```

2. Build binaries for configured targets:

```bash
omnidist build
```

3. Stage npm packages from `dist/`:

```bash
omnidist npm stage
```

4. Verify staged packages:

```bash
omnidist npm verify
```

5. Publish to npm:

```bash
omnidist npm publish
```

## Configuration

`omnidist.yaml`:

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
```

## Usage Examples

### 1) Build and Stage a Release From a Git Tag

```bash
git tag v1.2.0
omnidist build
omnidist npm stage
omnidist npm verify
```

With `version.source: git-tag`, package versions are derived from `git describe`.

### 2) Create a Dev Pre-release Stage

```bash
omnidist npm stage --dev
```

This converts git describe output to semver prerelease form like:

- `v1.2.0-5-gabc123` -> `1.2.0-dev.5.gabc123`

### 3) Use VERSION File as Version Source

```yaml
version:
  source: file
```

```bash
echo "1.3.0" > VERSION
omnidist npm stage
omnidist npm verify
```

### 4) Use Environment Variable as Version Source

```yaml
version:
  source: env
```

```bash
export VERSION=2.0.0
omnidist npm stage
omnidist npm verify
```

### 5) Dry-run Publish With Custom Registry/Tag

```bash
omnidist npm publish --dry-run --tag next --registry https://registry.npmjs.org
```

## Command Reference

Top-level:

- `omnidist init`
- `omnidist build`
- `omnidist npm`

NPM subcommands:

- `omnidist npm stage [--dev]`
- `omnidist npm verify`
- `omnidist npm publish [--dry-run] [--tag <tag>] [--registry <url>] [--otp <code>]`

## Project Layout

```text
cmd/omnidist/               CLI entrypoint and commands
internal/config/            Config model and YAML load/save
dist/                       Built binaries by os/arch
npm/                        Staged npm packages (meta + platform packages)
omnidist.yaml               Project configuration
```

## Notes

- `omnidist npm verify` enforces:
  - version parity across staged packages
  - presence of expected binaries
  - `optionalDependencies` correctness in meta package
  - no `scripts.postinstall` in staged packages
- Version resolution is strict: unresolved/empty version sources fail fast.
