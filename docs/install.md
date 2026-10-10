# Installing gowt

Next: the [getting started](index.md) page. A quick hello is in [`examples/hello`](../examples/hello).

```
go get github.com/haiodo/gowt@latest
```

Needs Go 1.27 (`go` line of `go.mod`). No cgo, no C toolchain, no Java: the generated code is committed and the OS libraries are
loaded at run time. Cross-compiling works from any host (`GOOS=windows go build`).

The default application name is the executable's base name (without `.exe`); `swt.DisplaySetAppName` overrides it.

## What must exist on the user's machine

| OS | Needed | If missing |
|---|---|---|
| macOS | 13 (Ventura) or later: the minimum Go 1.27 writes to the binary (`LC_BUILD_VERSION minos 13.0`). AppKit only | - |
| Windows | Windows 10 or later (amd64 or arm64; only run under Wine so far, the minimum is not verified); comctl32 v6 comes from the manifest, see below. WebView2 Runtime (Evergreen; preinstalled on Windows 11) for the `webview` package; no `WebView2Loader.dll` is needed | `webview.New` returns an error naming the runtime and its download page |
| Linux | GTK 3 (`libgtk-3.so.0` and its dependencies: gdk, cairo, pango, fontconfig, libX11) and an X11 or XWayland display | no GTK: a panic with an error naming the missing library and install commands (`gowt.Run` returns it as an error). No display: `panic: No more handles [gtk_init_check() failed]` from `NewDisplay` |

Linux packages: Debian/Ubuntu `libgtk-3-0`, Fedora `gtk3`, Arch `gtk3`, Alpine `gtk+3.0`. The `webview` package on Linux also needs WebKitGTK 4.1 (`libwebkit2gtk-4.1-0` on Debian/Ubuntu); the exact package names per distribution were not verified here.

## Windows: the manifest

Without a manifest Windows loads the classic common controls (no visual styles, no `SysLink`) and a bitmap-scaled, DPI-unaware process.
Import the manifest package once from the main program:

```go
import _ "github.com/haiodo/gowt/winmanifest"
```

It links `manifest_windows_{amd64,arm64}.syso` with comctl32 v6 and per-monitor DPI awareness (v2, falling back to per-monitor, then system).
The objects are made by `make winmanifest` (`tooling/mksyso`, pure Go, from `winmanifest/app.manifest`). Another `.syso` with its own
manifest in the same executable makes the linker fail with a duplicate resource; use one or the other. UTF-8 as the process code page is
not requested: the Windows port calls the wide API, so it would change nothing for it.

## macOS: app bundle

A bare executable runs, but Finder and the Dock list it by file name and a double click opens Terminal. `make app` builds `.app` bundles (ad-hoc signed)
of the demos: `tooling/darwin/mkapp.sh`.

## Checking a clean install

`make consumer-check` creates `../consumer` (module `example.com/app`, `replace` to this checkout) and builds it for darwin, windows and linux.
