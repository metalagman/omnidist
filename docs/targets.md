# Targets and variants

Configuration always uses Go spellings: `os` is `GOOS` and `arch` is `GOARCH`. Use `windows/amd64`, not `win32/x64`. Omnidist maps that source target into each package ecosystem.

## Default matrix

| Config target | Binary | npm suffix (`os`/`cpu`) | PyPI wheel platform tag | RubyGems platform |
| --- | --- | --- | --- | --- |
| `darwin/amd64` | `<name>` | `darwin-x64` | `macosx_10_13_x86_64` | `x86_64-darwin` |
| `darwin/arm64` | `<name>` | `darwin-arm64` | `macosx_11_0_arm64` | `arm64-darwin` |
| `linux/amd64` | `<name>` | `linux-x64` | `manylinux2014_x86_64` | `x86_64-linux` |
| `linux/arm64` | `<name>` | `linux-arm64` | `manylinux2014_aarch64` | `aarch64-linux` |
| `windows/amd64` | `<name>.exe` | `win32-x64` | `win_amd64` | `x64-mingw-ucrt` |

PyPI Linux tags use `distributions.pypi.linux-tag`; replacing the default with `musllinux_1_2` yields `musllinux_1_2_x86_64` or `musllinux_1_2_aarch64`.

## Opt-in Windows ARM64

Add `os: windows`, `arch: arm64` explicitly to target Windows on ARM; it is not part of the generated default matrix.

| Config target | Binary | npm suffix (`os`/`cpu`) | PyPI wheel platform tag | RubyGems platform |
| --- | --- | --- | --- | --- |
| `windows/arm64` | `<name>.exe` | `win32-arm64` | `win_arm64` | `aarch64-mingw-ucrt` |

Windows ARM64 gems use the `aarch64` CPU prefix, keeping their metadata, staging directories, and artifact filenames distinct from AMD64's `x64` gems. With an empty variant or `mingw-ucrt`, the platform is `aarch64-mingw-ucrt`, matching [RubyInstaller's native Windows ARM64 platform](https://rubyinstaller.org/2025/01/19/rubyinstaller-3.4.1-2-released.html).

## Variant behavior

`targets[].variant` is packaging metadata; it does not change the Go compiler target or automatically select a different libc/toolchain.

| Variant | npm | PyPI | gem |
| --- | --- | --- | --- |
| Empty | Standard `<package>-<os>-<cpu>` | Platform tag determined by OS/arch and PyPI `linux-tag` | Standard platform mapping above |
| `musl` on Linux | Adds `-musl` to platform package name | Set `linux-tag: musllinux_1_2` separately; target variant itself does not select the wheel tag | `x86_64-linux-musl` / `aarch64-linux-musl` |
| `mingw32` on Windows amd64 | Adds `-mingw32` suffix | Still `win_amd64` | `x64-mingw32` |
| `mingw-ucrt` on Windows amd64 | Adds `-mingw-ucrt` suffix | Still `win_amd64` | `x64-mingw-ucrt` |
| Other value | Appended to npm package name | No special mapping | Windows amd64 becomes `x64-<variant>` and arm64 becomes `aarch64-<variant>`; other OS mappings generally ignore it |

Windows ARM64 retains the same variant suffix rules: `mingw32` produces `aarch64-mingw32`, and custom values produce `aarch64-<variant>`. These labels are packaging metadata; they do not establish native Ruby runtime support for those variants.

Only amd64 and arm64 are supported by the current PyPI wheel mapper for Linux, macOS, and Windows. Treat non-default targets as opt-in and verify all generated packages before release.

With `build.cgo: false` (the default), Go binaries are generally more portable. A musl label does not make a CGO-linked binary musl-compatible; provide the correct toolchain and runtime compatibility yourself when enabling CGO or custom variants.
