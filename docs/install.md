# Installing gowt

<img src="img/art/install.png" alt="Gopher in a hard hat unpacking an app from a box" width="256" align="right">

Next: the [getting started](index.md) page. A quick hello is in [`examples/hello`](../examples/hello).

```
go get github.com/haiodo/gowt@latest
```

Needs Go 1.27 (`go` line of `go.mod`). No cgo, no C toolchain, no Java: the generated code is committed and the OS libraries are
loaded at run time. Cross-compiling works from any host (`GOOS=windows go build`).

The default application name is the executable's base name (without `.exe`); `gowt.SetAppName(name)` overrides it; call it before `gowt.Run`.

## What must exist on the user's machine

| OS | Needed | If missing |
|---|---|---|
| macOS | 13 (Ventura) or later: the minimum Go 1.27 writes to the binary (`LC_BUILD_VERSION minos 13.0`). AppKit only | - |
| Windows | Windows 10 or later (amd64 or arm64; run under Wine and on Windows 11 arm64; the minimum is not verified); comctl32 v6 comes from the manifest, see below. WebView2 Runtime (Evergreen; preinstalled on Windows 11) for the `webview` package; no `WebView2Loader.dll` is needed | `webview.New` returns an error naming the runtime and its download page |
| Linux | GTK 3 (`libgtk-3.so.0` and its dependencies: gdk, cairo, pango, fontconfig, libX11) and a Wayland or X11 display (GTK picks native Wayland on a Wayland session; `GDK_BACKEND=x11` forces Xwayland) | no GTK: an error naming the missing library and install commands. No display: an error ending in `No more handles [gtk_init_check() failed]`. `gowt.Run` returns both wrapped (`gowt: ...`); at the `swt` level they are panics with an `error` value from `NewDisplay`. Importing the packages works without GTK |

Linux packages: Debian/Ubuntu `libgtk-3-0`, Fedora `gtk3`, Arch `gtk3`. Distributions on musl (Alpine) are not supported: the binary needs glibc (`/lib/ld-linux-*.so.1`), and on Alpine it stops with `not found`; with `gcompat` it crashes in GLib.

On a minimal Debian or Ubuntu (`apt install --no-install-recommends libgtk-3-0`) also install `librsvg2-common`. It provides the SVG loader of gdk-pixbuf (`libpixbufloader-svg.so`, checked on ubuntu:24.04: absent without it); without it GTK cannot load the SVG that draws the check mark, so check boxes render empty and GTK prints `Could not load a pixbuf from .../Adwaita/assets/check-symbolic.svg`. A full desktop install has it.

The `webview` package on Linux also needs WebKitGTK 4.1: `libwebkit2gtk-4.1-0` on Debian and Ubuntu. Installing it and loading the `webview` example worked on ubuntu:24.04 (WebKitGTK 2.52.6) and debian:12 (2.50.6). Package names on other distributions were not checked. Without WebKitGTK the other widgets work and `webview.New` returns an error naming the missing library.

## Windows: the manifest

Without a manifest Windows loads the classic common controls (no visual styles, no `SysLink`) and a bitmap-scaled, DPI-unaware process.
A program built on the `gowt` facade needs nothing: `manifest_windows.go` in package `gowt` imports `winmanifest`. A program that uses package `swt` without `gowt` imports it once from the main program:

```go
import _ "github.com/haiodo/gowt/winmanifest"
```

It links `manifest_windows_{amd64,arm64}.syso` with comctl32 v6 and per-monitor DPI awareness (v2, falling back to per-monitor, then system).
The objects are made by `make winmanifest` (`tooling/mksyso`, pure Go, from `winmanifest/app.manifest`). Another `.syso` with its own
manifest in the same executable makes the linker fail with a duplicate resource; use one or the other (a program with its own manifest uses `swt` and does not import `gowt`). UTF-8 as the process code page is
not requested: the Windows port calls the wide API, so it would change nothing for it.

## macOS: app bundle

A bare executable runs, but Finder and the Dock list it by file name and a double click opens Terminal. `make app` builds `.app` bundles (ad-hoc signed)
of the demos: `tooling/darwin/mkapp.sh`.

## Checking a clean install

`make consumer-check` creates `../consumer` (module `example.com/app`, `replace` to this checkout) and builds it for darwin, windows and linux.
