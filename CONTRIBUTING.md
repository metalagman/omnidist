# Contributing

Contributor and project-reference material lives here. For installation and user-facing usage, start with [README.md](README.md).

## Why

- One release system for JavaScript, Python, and Ruby package ecosystems
- Run Go binaries via `npx`/`uvx` on machines without a Go runtime
- Install once with npm: `npm i -g <package>`
- Publish wheel artifacts to PyPI-compatible indexes with uv
- No install-time download scripts for npm
- Reproducible, CI-friendly flow from a single config file

## How It Works

`omnidist` supports three additive backends:

- `npm`:
  - meta package (for example `@scope/tool`) with shim and `optionalDependencies`
  - platform packages (for example `@scope/tool-linux-x64`) with prebuilt binaries
- `uv`:
  - per-target platform wheel artifacts in `.omnidist/uv/dist/`
  - one wheel per configured target with embedded binary in `<pkg>/bin/`
- `gem`:
  - per-target platform gem artifacts in `.omnidist/gem/pkg/`
  - one gem per configured target with embedded binary in `libexec/`

## Migration Guide (npm -> multi-backend)

1. Pull latest `omnidist` and run `omnidist init` in a clean branch to get uv/gem defaults in config.
2. Keep existing `distributions.npm` unchanged.
3. Add/update `distributions.uv` values:
   - `package` for wheel distribution name
   - `index-url` for target registry
   - `linux-tag` policy (`manylinux2014` default)
4. Add/update `distributions.gem` values:
   - `package` for RubyGems package name
   - `publish-auth` (`token` or `trusted`)
   - `repository-url` for gem metadata/trusted publishing setup
5. Extend CI pipeline with uv/gem stage/verify gates (see next section).
6. Release all backends in the same version cycle.

This is additive: npm support remains first-class and is not deprecated.

## CI and Release Flow (Multi-backend)

Recommended release sequence:

1. `omnidist build`
2. `omnidist stage`
3. `omnidist verify`
4. `omnidist publish`

For CI verification-only jobs, run steps 1-3.

When you need distribution-specific publish options (`npm --tag/--otp/--registry`, `uv --publish-url/--token`, `gem --host/--api-key/--otp`), use the backend subcommands directly.

## Project Layout

```text
cmd/omnidist/               CLI entrypoint and commands
internal/config/            Config model and YAML load/save
internal/workflow/          build/init/npm/uv/gem workflows
.omnidist/omnidist.yaml     Project configuration
.omnidist/.gitignore        Ignore rules for generated artifacts
.omnidist/dist/             Built binaries by os/arch
.omnidist/dist/VERSION      Version captured at build time
.omnidist/npm/              Staged npm packages
.omnidist/uv/pyproject.toml UV staging project with PEP 440 version
.omnidist/uv/dist/          Staged wheel artifacts
.omnidist/gem/build/        Per-platform gem staging directories
.omnidist/gem/pkg/          Staged gem artifacts
```
