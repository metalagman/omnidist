# Configuration reference

Omnidist reads `.omnidist/omnidist.yaml` unless `--config` or `OMNIDIST_CONFIG` selects another file. `omnidist init` creates profiles mode:

```yaml
profiles:
  default:
    tool:
      name: mytool
      main: ./cmd/mytool
    version:
      source: git-tag
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
        access: public
        publish-auth: token
        include-readme: true
      uv:
        package: mytool
        index-url: https://upload.pypi.org/legacy/
        linux-tag: manylinux2014
        include-readme: true
      gem:
        package: mytool
        registry: https://rubygems.org
        publish-auth: token
        include-readme: true
```

This example is a complete, copyable profile. Select another profile with `--profile NAME` or `OMNIDIST_PROFILE`; `default` is selected otherwise. Profile names may contain letters, digits, `.`, `_`, and `-`, except `.` and `..`.

## Workspace modes

| Mode | YAML root | Workspace |
| --- | --- | --- |
| Profiles (recommended) | `profiles.<name>` | `.omnidist/<name>/` |
| Legacy | runtime fields at document root | `.omnidist/` |

Do not mix `profiles` with top-level runtime fields. Existing legacy files remain supported. In canonical configuration, each key present under `distributions` selects that backend. `version.fixed-version` is not supported; use `version.fixed`.

Legacy-only shape, shown for compatibility rather than new projects:

```yaml
tool:
  name: mytool
  main: ./cmd/mytool
version:
  source: file
  file: VERSION
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
enabled-distributions: [npm]
distributions:
  npm:
    package: "@my-org/mytool"
```

## Top-level profile fields

| Field | Type | Default / required condition | Meaning |
| --- | --- | --- | --- |
| `tool` | object | Required | Binary identity and Go entrypoint. |
| `tool.name` | string | Generated from detected command; required for useful output | Binary/command name embedded in packages. |
| `tool.main` | string | Generated from detected `cmd/*`; required | Package passed to `go build`, for example `./cmd/mytool`. |
| `version` | object | `source: git-tag` | Release version source. |
| `version.source` | enum | `git-tag`; one of `git-tag`, `file`, `env`, `fixed` | How build resolves the version. |
| `version.file` | string | `VERSION` when source is `file` | Version file path relative to project root. |
| `version.fixed` | string | Required when source is `fixed` | Literal version. |
| `readme-path` | string | `README.md` fallback | Project-level README source. Backend `readme-path` takes precedence. |
| `targets` | list | Generated default matrix | Go build and package targets. |
| `targets[].os` | string | Required, Go `GOOS` spelling | Use `windows`, not npm's `win32`. |
| `targets[].arch` | string | Required, Go `GOARCH` spelling | Use `amd64`, not npm's `x64`. |
| `targets[].variant` | string | Empty | Optional packaging variant; see [targets and variants](targets.md). |
| `build` | object | Generated defaults | Go build controls. |
| `build.ldflags` | string | `-s -w` | Passed to `go build -ldflags`; environment expansion supports `$VAR` and `${VAR}`. |
| `build.tags` | string list | `[]` | Go build tags. |
| `build.cgo` | boolean | `false` | Sets `CGO_ENABLED`; cross-compilation usually needs `false`. |
| `enabled-distributions` | legacy non-empty unique list | Omit in new configs | Compatibility selector for existing files. Every listed backend must have a matching `distributions` section. Empty, null, duplicate, and unknown values are rejected. |
| `distributions` | typed map | At least one backend section | Backend configuration and the canonical aggregate selection. Section presence enables npm, uv, or gem in canonical files. |

Aggregate commands and generated CI use configured sections in npm → uv → gem order. `--only` narrows that set and errors when asked for an unavailable backend. A backend-specific command requires its explicit section. In a legacy file, an extra section excluded by `enabled-distributions` remains available to a direct backend command and is validated when invoked.

## Distribution fields

Every backend-specific field is listed below. A dash means the field is invalid for that backend; strict loading rejects misplaced and unknown fields.

| Field | npm | uv | gem | Default / condition |
| --- | --- | --- | --- | --- |
| `package` | Meta package name | Python distribution name | Gem name | Required in every configured backend. `omnidist init` writes a project-derived value; loading never invents an identity. |
| `aliases` | Additional equivalent meta package names | — | — | Empty. npm stages and publishes the primary `package` followed by aliases in configured order. Values must be unique valid npm names. |
| `platform-package` | Base name for target-specific packages | — | — | Empty; falls back to npm `package`, preserving legacy names. |
| `registry` | npm registry | — | Gem host | npm registry / `https://rubygems.org`. |
| `access` | `public` or `restricted` | — | — | `public`. |
| `publish-auth` | `token` or `trusted` | — | `token` or `trusted` | `token`. Gem trusted mode requires rubygems.org; npm trusted mode requires `repository-url`. |
| `repository-url` | package metadata and trusted publishing identity | — | gem metadata | Empty; required for npm trusted publishing and strongly recommended for gem trusted publishing. |
| `license` | npm package license metadata | — | gem license metadata | Empty; npm can infer/include a project license file. |
| `keywords` | npm meta package keywords | — | — | Empty; whitespace and duplicates are removed. |
| `readme-path` | README override | README override | README override | Empty; precedence is backend field → top-level `readme-path` → `README.md`. |
| `index-url` | — | Upload endpoint | — | `https://upload.pypi.org/legacy/`. |
| `linux-tag` | — | Linux wheel policy | — | `manylinux2014`; also accepts `musllinux_1_2`. |
| `include-readme` | Include README | Include README | Include README | `true`; set `false` to omit it. |

If an explicitly configured README cannot be read, staging fails. Metadata fields that a backend does not consume should be omitted.

### Multiple npm meta packages

`aliases` adds equivalent user-installable meta packages without duplicating target binaries. Every meta package shares the npm distribution metadata, release version, CLI shim, and optional dependencies on one shared platform package set:

```yaml
distributions:
  npm:
    package: omnidist
    aliases:
      - "@omnidist/omnidist"
    platform-package: "@omnidist/omnidist"
    registry: https://registry.npmjs.org
    access: public
```

For a Linux amd64 target, this stages the primary package followed by aliases—`omnidist` and `@omnidist/omnidist`—plus one binary package, `@omnidist/omnidist-linux-x64`. Both meta packages list the same scoped binary package in `optionalDependencies` and resolve it at runtime. Meta aliases do not redirect through or depend on the primary package.

Aliases are optional. Existing configs with only `package`, including configs that already use an independent `platform-package`, retain their current artifacts and publication order. An alias may be scoped or unscoped, but it must not duplicate `package` or another alias after whitespace normalization.

### Independent npm platform package names

`platform-package` changes only the base used for npm target packages. Omnidist appends the existing `-<os>-<cpu>` suffix and an optional target variant. The root package continues to use `package`:

```yaml
distributions:
  npm:
    package: omnidist
    platform-package: "@omnidist/omnidist"
    registry: https://registry.npmjs.org
    access: public
```

For a Linux amd64 target, this stages the root package as `omnidist` and the binary package as `@omnidist/omnidist-linux-x64`. The root package lists that scoped name in `optionalDependencies`, and its shim resolves the same name at runtime.

The field is opt-in. If it is absent, empty, or whitespace-only, target names continue to derive from `package` exactly as in earlier configurations. No migration is required for existing files. All meta and platform packages still share one version, registry, authentication mode, access setting, and platform-first publish order. Platform packages publish first; the primary package followed by aliases publishes afterward.

For public scoped packages, keep `access: public`. The publishing npm account or organization must own every configured scope and have permission to publish each unscoped name. In trusted mode, configure every meta package and platform package as a trusted publisher; `omnidist npm trust` derives the complete package set.

## Versions and build variables

- `git-tag`: `HEAD` must be exactly at `vX.Y.Z` or `X.Y.Z`.
- `file`: reads `version.file`.
- `env`: reads `OMNIDIST_VERSION`.
- `fixed`: reads `version.fixed`.

Build writes the resolved value to `<workspace>/dist/VERSION`; every backend stages from that value. `stage --dev` derives a development version from Git metadata. PyPI/TestPyPI reject local `+...` metadata, and npm automatically uses the `dev` dist-tag for detected dev versions unless overridden.

`build.ldflags` may reference `OMNIDIST_VERSION`, `OMNIDIST_GIT_COMMIT`, and `OMNIDIST_BUILD_DATE`. Unset variables expand to an empty string.

## Paths and precedence

| Artifact | Profiles mode | Legacy mode |
| --- | --- | --- |
| Build binaries/version | `.omnidist/<profile>/dist/` | `.omnidist/dist/` |
| npm stage | `.omnidist/<profile>/npm/` | `.omnidist/npm/` |
| uv stage/wheels | `.omnidist/<profile>/uv/` | `.omnidist/uv/` |
| gem stage/packages | `.omnidist/<profile>/gem/` | `.omnidist/gem/` |

CLI flags take precedence over their environment-backed selectors. Omnidist loads `.env` before command execution. Publishing credentials and their precedence are documented in the [release runbook](releases.md).

## Safe migration

Do not run `omnidist init --force` merely to upgrade an existing config. Edit the current file in place:

1. Keep only the `distributions` sections you publish, and set each section's `package` explicitly.
2. Remove `enabled-distributions` to adopt the canonical presence-based shape. If a legacy file intentionally keeps inactive sections, retain a non-empty selector whose values all have matching sections.
3. Run `omnidist build`, `omnidist stage`, and `omnidist verify`.
4. Inspect `omnidist ci --dry-run`, then replace an existing generated workflow only with `omnidist ci --force`.

Omitting `enabled-distributions` selects exactly the backend sections present. For example, a pre-selector config containing only `distributions.npm` continues to run only npm workflows. A selector that names a missing section now fails before staging, CI generation, authentication, or publication.
