# Targets and variants

Configuration always uses Go spellings: `os` is `GOOS` and `arch` is `GOARCH`. Use `windows/amd64`, not `win32/x64`. Omnidist maps that source target into each package ecosystem.

## Default matrix

| Config target | Binary | npm suffix (`os`/`cpu`) | uv wheel platform tag | RubyGems platform |
| --- | --- | --- | --- | --- |
| `darwin/amd64` | `<name>` | `darwin-x64` | `macosx_10_13_x86_64` | `x86_64-darwin` |
| `darwin/arm64` | `<name>` | `darwin-arm64` | `macosx_11_0_arm64` | `arm64-darwin` |
| `linux/amd64` | `<name>` | `linux-x64` | `manylinux2014_x86_64` | `x86_64-linux` |
| `linux/arm64` | `<name>` | `linux-arm64` | `manylinux2014_aarch64` | `aarch64-linux` |
| `windows/amd64` | `<name>.exe` | `win32-x64` | `win_amd64` | `x64-mingw-ucrt` |

uv Linux tags use `distributions.uv.linux-tag`; replacing the default with `musllinux_1_2` yields `musllinux_1_2_x86_64` or `musllinux_1_2_aarch64`.

## Variant behavior

`targets[].variant` is packaging metadata; it does not change the Go compiler target or automatically select a different libc/toolchain.

| Variant | npm | uv | gem |
| --- | --- | --- | --- |
| Empty | Standard `<package>-<os>-<cpu>` | Platform tag determined by OS/arch and uv `linux-tag` | Standard platform mapping above |
| `musl` on Linux | Adds `-musl` to platform package name | Set `linux-tag: musllinux_1_2` separately; target variant itself does not select the wheel tag | `x86_64-linux-musl` / `aarch64-linux-musl` |
| `mingw32` on Windows amd64 | Adds `-mingw32` suffix | Still `win_amd64` | `x64-mingw32` |
| `mingw-ucrt` on Windows amd64 | Adds `-mingw-ucrt` suffix | Still `win_amd64` | `x64-mingw-ucrt` |
| Other value | Appended to npm package name | No special mapping | Windows becomes `x64-<variant>`; other OS mappings generally ignore it |

Only amd64 and arm64 are supported by the current uv wheel mapper for Linux, macOS, and Windows. The generated default matrix does not include Windows arm64 even though the wheel mapper can name it; treat non-default targets as opt-in and verify all generated packages before release.

With `build.cgo: false` (the default), Go binaries are generally more portable. A musl label does not make a CGO-linked binary musl-compatible; provide the correct toolchain and runtime compatibility yourself when enabling CGO or custom variants.
