# omnidist

[![Lint](https://github.com/metalagman/omnidist/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/lint.yml)
[![Test](https://github.com/metalagman/omnidist/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/test.yml)
[![Security](https://github.com/metalagman/omnidist/actions/workflows/security.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/security.yml)
[![npm](https://img.shields.io/npm/v/%40omnidist%2Fomnidist)](https://www.npmjs.com/package/@omnidist/omnidist)
[![PyPI](https://img.shields.io/pypi/v/omnidist)](https://pypi.org/project/omnidist/)
[![Gem](https://img.shields.io/gem/v/omnidist)](https://rubygems.org/gems/omnidist)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Omnidist builds one Go CLI and packages its prebuilt binaries for npm, PyPI-compatible indexes through uv, and RubyGems. npm packages contain no `postinstall` downloader; installation uses platform-specific optional dependencies already present in registry tarballs.

## Install or run

Consumers only need the package manager for the channel they use:

```bash
npx -y @omnidist/omnidist@latest --help
uvx omnidist --help
gem install omnidist && omnidist --help
```

Maintainers can also install from Go:

```bash
go install github.com/metalagman/omnidist/cmd/omnidist@latest
```

## Maintainer prerequisites

| Activity | Required locally or in CI |
| --- | --- |
| Build the Go CLI | Go 1.25+; Git when `version.source: git-tag` |
| npm stage/verify | Go build artifacts; npm is required at publish preflight/upload |
| uv stage/verify/publish | `uv` |
| gem stage/verify/publish | Ruby and RubyGems (`gem`) |
| Token publish | The selected backend's token environment variable |

You do not need tooling for disabled backends.

## Safe quick start

Run this in an existing Go repository with one `package main` under `cmd/*`:

```bash
omnidist init
$EDITOR .omnidist/omnidist.yaml
omnidist build
omnidist stage
omnidist verify
omnidist publish --dry-run
```

`init` detects the command name and main package. If discovery is ambiguous, specify both explicitly:

```bash
omnidist init --name mytool --main ./cmd/mytool
```

It refuses to replace an existing config. `omnidist init --force` intentionally replaces that file and is destructive; review or back it up first.

The generated file uses profiles mode and stores artifacts beneath `.omnidist/default/`. Select which package ecosystems participate in aggregate commands and generated CI:

```yaml
profiles:
  default:
    enabled-distributions: [npm, uv]
    tool:
      name: mytool
      main: ./cmd/mytool
    version:
      source: git-tag
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
    distributions:
      npm:
        package: "@my-org/mytool"
      uv:
        package: mytool
      gem:
        package: mytool
```

Valid enabled values are `npm`, `uv`, and `gem`. An omitted field preserves compatibility by enabling all three. Aggregate commands always execute in npm → uv → gem order. `--only` can narrow the configured set but cannot enable a disabled backend; backend-specific commands remain available for repair and recovery.

To keep the npm package installed by users unscoped while publishing platform binaries under a scope, set an independent platform-package base:

```yaml
distributions:
  npm:
    package: omnidist
    platform-package: "@omnidist/omnidist"
    access: public
```

This produces platform packages such as `@omnidist/omnidist-linux-x64`, while the root package remains `omnidist`. If `platform-package` is omitted or blank, it defaults to `package`, preserving existing package names. The configured npm identity must be allowed to publish the unscoped name and every package in the selected scope; trusted publishing must be configured for every generated package.

## Release flow

```bash
omnidist build
omnidist stage
omnidist verify
omnidist publish --dry-run
omnidist publish
```

Aggregate publish preflights every selected backend before the first upload. This prevents locally detectable late-backend failures from causing an avoidable partial release. Registry uploads are still external and are not transactionally reversible: a network or registry failure after uploads begin can leave a partial release.

Generate backend-aware GitHub Actions after the config is final:

```bash
omnidist ci
# use --force only to intentionally replace the generated workflow
```

The workflow contains setup, credentials, and publish jobs only for enabled backends, plus a GitHub Release job for the built binaries.

## Reference

- [Configuration reference](docs/configuration.md) — profiles, every YAML field, defaults, paths, and compatibility.
- [Release runbook](docs/releases.md) — credentials, first release, preflight, trusted publishing, and partial-release recovery.
- [Targets and variants](docs/targets.md) — Go target values and npm/wheel/gem mappings.
- [Contributing](CONTRIBUTING.md) — repository layout, tests, lint, and development workflow.

## Commands

```text
omnidist init [--force] [--name <name>] [--main <package>]
omnidist quickstart
omnidist build
omnidist stage [--dev] [--only npm,uv,gem]
omnidist verify [--only npm,uv,gem]
omnidist publish [--dry-run] [--only npm,uv,gem]
omnidist ci [--force] [--dry-run]
omnidist npm stage|verify|publish|trust
omnidist uv stage|verify|publish
omnidist gem stage|verify|publish
```

Global flags are `--config`, `--profile`, and `--omnidist-root`. Omnidist also loads `.env`; corresponding selectors are `OMNIDIST_CONFIG`, `OMNIDIST_PROFILE`, and `OMNIDIST_OMNIDIST_ROOT`.

Use `omnidist <command> --help` for backend-specific options.

## Common failures

- Version resolution: an exact SemVer tag is required for `git-tag`; `file` reads `version.file`; `env` reads `OMNIDIST_VERSION`; `fixed` requires `version.fixed`.
- Missing build version: run `omnidist build` before stage. Profiles store it at `.omnidist/<profile>/dist/VERSION`; legacy config uses `.omnidist/dist/VERSION`.
- Missing npm platform package: run `omnidist npm verify` and confirm the target matrix and equal versions.
- PyPI rejects `+...`: public PyPI/TestPyPI reject local version metadata; stage a publishable version.
- Trusted publish fails: local preflight cannot prove remote OIDC configuration. Confirm repository metadata, workflow filename/environment, `id-token: write`, and registry-side trusted publisher settings.
