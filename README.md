# omnidist

[![Lint](https://github.com/metalagman/omnidist/actions/workflows/lint.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/lint.yml)
[![Test](https://github.com/metalagman/omnidist/actions/workflows/test.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/test.yml)
[![Security](https://github.com/metalagman/omnidist/actions/workflows/security.yml/badge.svg)](https://github.com/metalagman/omnidist/actions/workflows/security.yml)
[![npm](https://img.shields.io/npm/v/%40omnidist%2Fomnidist)](https://www.npmjs.com/package/@omnidist/omnidist)
[![PyPI](https://img.shields.io/pypi/v/omnidist)](https://pypi.org/project/omnidist/)
[![Gem](https://img.shields.io/gem/v/omnidist)](https://rubygems.org/gems/omnidist)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Build your Go CLI for macOS, Linux, and Windows, then distribute it through npm, PyPI, and RubyGems with one release workflow.

- **Easy installation for your users.** Ship prebuilt binaries through their package manager; they do not need Go.
- **One configuration for your releases.** Choose your platforms and package ecosystems, with shared descriptions, keywords, and licenses.
- **Packages you can inspect before publishing.** Build, stage, verify, and dry-run the release locally.
- **GitHub Actions included.** Generate a workflow that builds your binaries, publishes packages, and creates a GitHub Release.

npm packages use platform-specific optional dependencies, with no `postinstall` downloader.

## Install

Choose the package manager you already use:

| Channel | Command |
| --- | --- |
| npm | `npm install -g @omnidist/omnidist` |
| uv / PyPI | `uv tool install omnidist` |
| RubyGems | `gem install omnidist` |
| Go | `go install github.com/metalagman/omnidist/cmd/omnidist@latest` |

Ensure your package manager's executable directory is on `PATH`, then run `omnidist --help`. You can also try it without a persistent install using `npx -y @omnidist/omnidist@latest --help` or `uvx omnidist --help`.

## Quick start

Start in an existing Go CLI repository. Building requires Go 1.25+; the selected package channels need npm, uv, or Ruby/RubyGems for their packaging and publish checks. See [tooling requirements](docs/releases.md#tooling) for the details.

Initialize your configuration:

```bash
omnidist init
```

Omnidist discovers a main package under `cmd/*`. If your project has several commands, choose one explicitly: `omnidist init --name mytool --main ./cmd/mytool`.

Open `.omnidist/omnidist.yaml`, review the generated package names and targets, and keep the `npm`, `pypi`, or `gem` sections under `distributions` for the channels you want. For a first local trial, replace the `version` section inside `profiles.default` with:

```yaml
version:
  source: fixed
  fixed: "0.1.0"
```

This gives the trial a version without creating a Git tag. Keep the other generated fields, then run:

```bash
omnidist build
omnidist stage
omnidist verify
omnidist publish --dry-run
```

Your binaries and packages are now staged under `.omnidist/default/`. The dry-run checks publication readiness without uploading or requiring registry credentials.

When you are ready to publish, choose your [release version source](docs/configuration.md#versions-and-build-variables), configure registry access, and follow the [first-release guide](docs/releases.md#first-release-checklist).

## GitHub Actions

After choosing your release version source and configuring registry credentials or trusted publishing, generate the workflow:

```bash
omnidist ci
```

It creates `.github/workflows/omnidist-release.yml` for the channels in your configuration, including binary assets and checksums on GitHub Releases. See [CI releases](docs/releases.md#ci-releases) for setup and publication behavior.

## Agent skill

Use the [Omnidist agent skill](skills/omnidist/SKILL.md) to help your coding agent configure and run releases:

```bash
npx skills add metalagman/omnidist --skill omnidist
```

Run this in the project where you want to use the skill; add `--global` for use across projects. Install the Omnidist CLI separately using the options above.

## Documentation

- [Configuration](docs/configuration.md) — profiles, package names, metadata, versions, and CLI options.
- [Releases](docs/releases.md) — tooling, credentials, publishing, troubleshooting, and recovery.
- [Targets](docs/targets.md) — supported platforms and package variants.
- [Legacy compatibility](docs/configuration.md#legacy-compatibility) — existing uv configuration, commands, and environment variables.

To contribute to Omnidist, see [Contributing](CONTRIBUTING.md).
