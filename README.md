# omnidist

[![Go Report Card](https://goreportcard.com/badge/github.com/metalagman/omnidist)](https://goreportcard.com/report/github.com/metalagman/omnidist)
[![lint](https://github.com/metalagman/omnidist/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/lint.yml)
[![test](https://github.com/metalagman/omnidist/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/test.yml)
[![codecov](https://codecov.io/github/metalagman/omnidist/graph/badge.svg)](https://codecov.io/github/metalagman/omnidist)
[![version](https://img.shields.io/github/v/release/metalagman/omnidist?sort=semver)](https://github.com/metalagman/omnidist/releases)
[![npm](https://img.shields.io/npm/v/%40omnidist%2Fomnidist)](https://www.npmjs.com/package/@omnidist/omnidist)
[![PyPI](https://img.shields.io/pypi/v/omnidist)](https://pypi.org/project/omnidist/)
[![license](https://img.shields.io/github/license/metalagman/omnidist)](LICENSE)

Package and publish a Go CLI as npm and uv installable tools with prebuilt
cross-platform binaries.

`omnidist` gives Go CLI maintainers one repeatable release flow:

```text
build -> stage -> verify -> publish
```

The generated npm packages use platform-specific optional dependencies, so users
can run your CLI with `npx` without install-time downloader scripts. The uv
distribution stages wheel artifacts so users can run the same CLI from Python
tooling with `uvx`.

## Requirements

- Go 1.25+
- Node.js and npm for npm staging, verification, and publishing
- `uv` for uv staging, verification, and publishing
- `git` when `version.source: git-tag`
- `NPM_PUBLISH_TOKEN` for npm token publishing, unless using `--dry-run` or trusted publishing
- `UV_PUBLISH_TOKEN` or `omnidist uv publish --token` for uv publishing, unless using `--dry-run`

## Install

Run without installing:

```bash
npx -y @omnidist/omnidist@latest --help
uvx omnidist --help
```

Install with npm:

```bash
npm i -g @omnidist/omnidist
omnidist --help
```

Install with Go:

```bash
go install github.com/metalagman/omnidist/cmd/omnidist@latest
omnidist --help
```

Run from a checkout:

```bash
go run ./cmd/omnidist --help
```

## Quick Start

Initialize an existing Go CLI repository:

```bash
omnidist init
```

This creates `.omnidist/omnidist.yaml` and initial workspace directories under
`.omnidist/default/`. The generated config uses `profiles.default`.

Edit the generated config before building:

```bash
$EDITOR .omnidist/omnidist.yaml
```

At minimum, check these fields:

- `tool.name`: the binary name users will run.
- `tool.main`: the Go main package, for example `./cmd/mytool`.
- `distributions.npm.package`: the npm package, for example `@my-org/mytool`.
- `distributions.uv.package`: the uv/PyPI package, for example `mytool`.
- `version.source`: usually `git-tag`, `file`, `env`, or `fixed`.

Then run the local release pipeline:

```bash
omnidist build
omnidist stage
omnidist verify
```

Publish only after verification passes:

```bash
omnidist publish
```

Generate a GitHub Actions release workflow:

```bash
omnidist ci
```

The generated workflow is written to `.github/workflows/omnidist-release.yml`.
It runs on `v*` tag pushes, builds once, stages and verifies artifacts, publishes
npm and uv distributions, then uploads the built binaries and `checksums.txt` to
the GitHub release.

## Configuration

`omnidist init` writes profiles-mode config by default:

```yaml
profiles:
  default:
    tool:
      name: mytool
      main: ./cmd/mytool

    version:
      source: git-tag # git-tag | file | env | fixed
      file: VERSION # used only when source is file
      fixed: 1.2.3 # required only when source is fixed

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
        package: "@my-org/mytool"
        registry: https://registry.npmjs.org
        access: public # public | restricted
        publish-auth: token # token | trusted
        include-readme: true

      uv:
        package: mytool
        index-url: https://upload.pypi.org/legacy/
        linux-tag: manylinux2014 # manylinux2014 | musllinux_1_2
        include-readme: true
```

Select a profile with `--profile <name>` or `OMNIDIST_PROFILE`. If `profiles`
is present and no profile is selected, `default` is used. Profiles write
artifacts to `.omnidist/<profile>/`.

Legacy top-level config is still accepted when loading existing config files,
but do not mix top-level runtime fields with a `profiles` map in the same file.

Targets use Go values: `os` is `GOOS`, and `arch` is `GOARCH`. Distribution
workflows map them as needed, for example `windows/amd64` becomes npm
`win32/x64`.

## Versioning

`omnidist build` resolves the release version and writes it to
`.omnidist/<profile>/dist/VERSION`. Stage and publish commands use that build
version so npm and uv artifacts stay in sync.

Supported version sources:

- `git-tag`: `HEAD` must be on an exact SemVer tag, either `vX.Y.Z` or `X.Y.Z`.
- `file`: read the version from `version.file`, defaulting to `VERSION`.
- `env`: read `OMNIDIST_VERSION`.
- `fixed`: read `version.fixed`.

For local prerelease staging, use:

```bash
omnidist stage --dev
```

`--dev` generates prerelease artifact versions from git metadata. For npm
publishing, `-dev` prerelease versions are automatically published with
`--tag dev` when no tag is supplied.

## Environment and Build Variables

`omnidist` loads `.env` at startup when the file exists.

Environment variables:

- `OMNIDIST_VERSION`: used only when `version.source: env`. `VERSION` is not used.
- `OMNIDIST_CONFIG`: config file path, equivalent to `--config`.
- `OMNIDIST_PROFILE`: config profile name, equivalent to `--profile`.
- `OMNIDIST_OMNIDIST_ROOT`: project root directory, equivalent to `--omnidist-root`.
- `NPM_PUBLISH_TOKEN`: npm token for `distributions.npm.publish-auth: token`.
- `UV_PUBLISH_TOKEN`: uv publish token when `--token` is not provided.

Build `ldflags` template variables:

- `OMNIDIST_VERSION`: expanded in `build.ldflags` during `omnidist build`.
- `OMNIDIST_GIT_COMMIT`: populated by `omnidist build` when git metadata is available.
- `OMNIDIST_BUILD_DATE`: populated by `omnidist build` as UTC RFC3339.

Example:

```yaml
build:
  ldflags: >-
    -s -w
    -X github.com/your-org/mytool/internal/version.version=${OMNIDIST_VERSION}
    -X github.com/your-org/mytool/internal/version.gitCommit=${OMNIDIST_GIT_COMMIT}
    -X github.com/your-org/mytool/internal/version.buildDate=${OMNIDIST_BUILD_DATE}
```

`build.ldflags` uses `os.ExpandEnv`, so both `$VAR` and `${VAR}` are supported.
Unset variables expand to empty strings.

## npm Distribution

The npm distribution has two package types:

- Meta package: `distributions.npm.package`, with a small Node shim.
- Platform packages: one package per target, selected by npm `os` and `cpu` constraints.

The meta package lists platform packages as `optionalDependencies` at the same
version. Published packages do not use `postinstall` scripts or network
downloaders.

Common npm commands:

```bash
omnidist npm stage
omnidist npm verify
omnidist npm publish --tag latest
```

Token publishing uses `NPM_PUBLISH_TOKEN`. `omnidist` writes a workspace
`.omnidist/.npmrc` from `distributions.npm.registry` with npm's token
substitution syntax.

Trusted publishing uses npm OIDC instead of `NPM_PUBLISH_TOKEN`:

```yaml
distributions:
  npm:
    publish-auth: trusted
    repository-url: git+https://github.com/your-org/your-repo.git
```

In trusted mode:

- `repository-url` is required and is written to staged package metadata.
- `omnidist npm publish` skips token-only auth preflight.
- GitHub Actions must grant `id-token: write`.
- Each npm package, including platform packages, needs a trusted publisher configured on npm.

Print trusted publisher setup commands:

```bash
omnidist npm trust
```

Apply them directly:

```bash
omnidist npm trust --apply
```

Useful trust options:

- `--workflow-file <name>` for a workflow filename other than `omnidist-release.yml`.
- `--repo <owner/repo>` to override `distributions.npm.repository-url`.
- `--environment <name>` for npm trusted publishers restricted to a GitHub Actions environment.
- `--allow-stage-publish` to allow npm stage publish.

## uv Distribution

The uv distribution stages wheel artifacts under `.omnidist/<profile>/uv/dist`.
During staging, versions are converted to PEP 440 where needed.

Common uv commands:

```bash
omnidist uv stage
omnidist uv verify
omnidist uv publish
```

Publish to a different PyPI-compatible index:

```bash
omnidist uv publish --publish-url https://test.pypi.org/legacy/ --token <token>
```

`omnidist uv verify` rejects versions with local metadata (`+...`) for
PyPI/TestPyPI publishing, because those indexes reject local versions.

## README and License Staging

README source precedence during staging:

```text
distributions.<name>.readme-path -> readme-path -> README.md
```

If a configured `readme-path` is set and cannot be read, staging fails. Set
`include-readme: false` for a distribution to skip README inclusion.

For npm packages, `license` can be set explicitly under `distributions.npm`.
When it is omitted, omnidist includes the project license file when present and
writes package metadata that points to that file.

## CI Releases

Generate a release workflow:

```bash
omnidist ci
```

Overwrite an existing generated workflow:

```bash
omnidist ci --force
```

The workflow installs omnidist with npm. When generating this repository's own
release workflow, it uses `go run ./cmd/omnidist` instead. The workflow also
sets `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` for JavaScript-based GitHub
Actions.

Before the first tag release, configure the required registry credentials:

- npm token mode: `NPM_PUBLISH_TOKEN` GitHub secret.
- npm trusted mode: npm trusted publishers for every staged npm package.
- uv publishing: `UV_PUBLISH_TOKEN` GitHub secret.

Release by pushing a SemVer tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

## Command Reference

Top-level commands:

```bash
omnidist init
omnidist quickstart
omnidist build
omnidist stage [--dev] [--only npm|uv|npm,uv]
omnidist verify [--only npm|uv|npm,uv]
omnidist publish [--dry-run] [--only npm|uv|npm,uv]
omnidist ci [--force] [--dry-run]
omnidist version
```

Global flags:

```bash
--config <path>
--profile <name>
--omnidist-root <path>
```

npm commands:

```bash
omnidist npm stage [--dev]
omnidist npm verify
omnidist npm publish [--dry-run] [--tag <tag>] [--registry <url>] [--otp <code>]
omnidist npm trust [--apply] [--workflow-file <name>] [--repo <owner/repo>] [--environment <name>] [--allow-stage-publish]
```

uv commands:

```bash
omnidist uv stage [--dev]
omnidist uv verify
omnidist uv publish [--dry-run] [--publish-url <url>] [--token <token>]
```

Top-level `stage`, `verify`, and `publish` run distributions in deterministic
order: npm first, then uv. Use `--only` to limit the command to one or more
distributions.

## Troubleshooting

`omnidist build` says no version could be resolved:

Check `version.source`. For `git-tag`, `HEAD` must be exactly on a SemVer tag.
For `env`, set `OMNIDIST_VERSION`. For `file`, create the configured version
file.

`omnidist stage` says the build version file is missing:

Run `omnidist build` first. Stage commands use the build version persisted under
`.omnidist/<profile>/dist/VERSION`.

npm install cannot find a platform package:

Run `omnidist npm verify` before publishing. It checks package versions,
platform `os` and `cpu` fields, required binaries, forbidden `postinstall`
scripts, repository metadata, and optional dependency parity.

Trusted npm publish fails:

Verify that `distributions.npm.repository-url` matches the GitHub repository,
the workflow grants `id-token: write`, and every npm package has a trusted
publisher configured.

uv publish rejects the version:

Restage with a publishable version. PyPI and TestPyPI reject local version
metadata containing `+`.

## Contributing

For repository layout, development workflow, package model details, and migration
notes, see [CONTRIBUTING.md](CONTRIBUTING.md).
