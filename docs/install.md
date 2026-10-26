# Installing gowt

<img src="img/art/install.png" alt="Gopher in a hard hat unpacking an app from a box" width="220" align="right">

Next: the [getting started](README.md) page. A quick hello is in [`examples/hello`](../examples/hello).

```
go get github.com/haiodo/gowt@latest
```

Needs Go 1.27 (`go` line of `go.mod`). No cgo, no C toolchain, no Java: the generated code is committed and the OS libraries are
loaded at run time. Cross-compiling works from any host (`GOOS=windows go build`).

The default application name is the executable's base name (without `.exe`); `gowt.SetAppName(name)` overrides it; call it before `gowt.Run`.

<br clear="right">

## What must exist on the user's machine

| OS | Needed | If missing |
|---|---|---|
| macOS | 13 (Ventura) or later: the minimum Go 1.27 writes to the binary (`LC_BUILD_VERSION minos 13.0`). AppKit only | - |
| Windows | Windows 10 or later (amd64 or arm64; run under Wine and on Windows 11 arm64; the minimum is not verified); comctl32 v6 comes from the manifest, see below. WebView2 Runtime (Evergreen; preinstalled on Windows 11) for the `webview` package; no `WebView2Loader.dll` is needed | `webview.New` returns an error naming the runtime and its download page |
| Linux | GTK 3 (`libgtk-3.so.0` and its dependencies: gdk, cairo, pango, fontconfig, libX11) and a Wayland or X11 display (GTK picks native Wayland on a Wayland session; `GDK_BACKEND=x11` forces Xwayland) | no GTK: an error naming the missing library and install commands. No display: an error ending in `No more handles [gtk_init_check() failed]`. `gowt.Run` returns both wrapped (`gowt: ...`); at the `swt` level they are panics with an `error` value from `NewDisplay`. Importing the packages works without GTK |

Linux packages: Debian/Ubuntu `libgtk-3-0`, Fedora `gtk3`, Arch `gtk3`. Distributions on musl (Alpine) are not supported: the binary needs glibc (`/lib/ld-linux-*.so.1`), and on Alpine it stops with `not found`; with `gcompat` it crashes in GLib.

On a minimal Debian or Ubuntu (`apt install --no-install-recommends libgtk-3-0`) also install `librsvg2-common`. It provides the SVG loader of gdk-pixbuf (`libpixbufloader-svg.so`, checked on ubuntu:24.04: absent without it); without it GTK cannot load the SVG that draws the check mark, so check boxes render empty and GTK prints `Could not load a pixbuf from .../Adwaita/assets/check-symbolic.svg`. A full desktop install has it.

The `webview` package on Linux also needs WebKitGTK 4.1: `libwebkit2gtk-4.1-0` on Debian and Ubuntu. Installing it and loading the `webview` example worked on ubuntu:24.04 (WebKitGTK 2.52.6) and debian:12 (2.50.6). Package names on other distributions were not checked. Without WebKitGTK the other widgets work and `webview.New` returns an error naming the missing library.

## Requirements and troubleshooting by OS

Every OS: Go 1.27 or newer on `PATH` (`go version`) and network access for the first `go get`. A Go older than 1.21 cannot switch toolchains (not run here); Go 1.21 to 1.26 downloads the 1.27 toolchain by itself on the first build (`go: downloading go1.27.x`, about 75 MB), which needs the same network and disk as an install. `GOTOOLCHAIN=local` turns that off and gives `go: go.mod requires go >= 1.27 (running go 1.2x; GOTOOLCHAIN=local)`. Distribution packages are too old for that (Ubuntu 22.04 `golang-go` is 1.18, 24.04 is 1.22, 26.04 is 1.26): install Go from go.dev.

Disk: a cold build of `examples/hello` filled a build cache of 113 to 228 MB on Linux arm64 (Go 1.27.3) and about 750 MB on macOS (a cache that also held other targets, so the figure is an upper bound), plus 71 MB of module cache. The first build took 1.5 to 2 minutes on Linux arm64 in Docker and about 15 s on a Mac. Memory: compiling package `swt` peaked at 1.9 GB resident on macOS (`/usr/bin/time -l go build`); six such builds at once in a 12 GB Docker VM ended with `github.com/haiodo/gowt/swt: /usr/local/go/pkg/tool/linux_arm64/compile: signal: killed`. On a small VM or container give it 3 GB or more. A near-full system drive is the usual cause of a failed first build.

### Windows (x64 and arm64)

- Install Go from the MSI at go.dev/dl or with `winget install GoLang.Go`. The installer puts `C:\Program Files\Go\bin` on `PATH`; a terminal that was open during the install does not see it. Error: `go : The term 'go' is not recognized as the name of a cmdlet, function, script file, or operable program.` Fix: open a new terminal, or call `& "C:\Program Files\Go\bin\go.exe" version`. Keep Go in a permanent folder: a copy unpacked into a temporary folder is not on `PATH` in other terminals.
- Caches go to `%LocalAppData%\go-build` and `%USERPROFILE%\go`. To keep them off a small `C:` drive: `go env -w GOCACHE=D:\gocache GOMODCACHE=D:\gomod`. Error on a full drive: `no space left on device` or `write ...: There is not enough space on the disk.`
- No C compiler, no MSYS, no Visual Studio is needed. `CGO_ENABLED=0` is set automatically when no `gcc` is found.
- Run: `go run ./examples/hello`, or `go build` and start the `.exe`. Add `-ldflags=-H=windowsgui` to get no console window.
- `webview` needs the WebView2 Runtime (Evergreen; part of Windows 11 and of current Microsoft Edge installs). Without it `webview.New` returns at once: `webview: the WebView2 Runtime is not installed (Windows 10 needs the Evergreen runtime from https://developer.microsoft.com/microsoft-edge/webview2/, Windows 11 has it); looked in: ...`. Check by hand: `reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv` (the GUID is easy to mistype by one character; a mistyped key reports "not found" for a runtime that is installed).
- If the runtime is found but the view cannot be created (broken install, a session without a desktop), `OnNavigationFailed` and `Eval` callbacks get `webview: the WebView2 view was not created in time (runtime broken, or no desktop session)` after 20 s. Without it a failed creation of that kind could leave loads pending with no message at all (a Windows tester saw a 40 s test hang; the cause on that machine is not known).
- GUI programs need an interactive desktop session. Run from a logged-on desktop (a scheduled task with "run only when user is logged on" worked on Windows 11 arm64); a start from plain SSH was not tried.
- Not checked: Windows 10, SmartScreen or antivirus reactions to an unsigned `.exe`, `winget` on a machine without it.

### macOS

- Needs macOS 13 or later (`minos 13.0` in the binary, checked with `otool -l`). Xcode or the Command Line Tools are not needed: the program is pure Go, and the linker signs it ad hoc. On Apple silicon and Intel alike `go build` is enough.
- A binary copied to another Mac through a browser or chat is quarantined and Gatekeeper refuses it (`"app" cannot be opened because the developer cannot be verified`). Fix: sign and notarize ([packaging.md](packaging.md)), or on your own test machine `xattr -d com.apple.quarantine ./app`.
- Not checked in this round: a run on a clean Mac and on macOS 13 to 15; the floor is from the binary header only.

### Linux

- Go: see above. Then `go get github.com/haiodo/gowt@latest`, build, run. The binary links `libc.so.6`, `libdl.so.2`, `libpthread.so.0` only; GTK is opened at start.
- No GTK 3: `gowt: GTK 3 is not installed (libgtk-3.so.0: cannot open shared object file: No such file or directory); install it: apt install libgtk-3-0 | dnf install gtk3 | pacman -S gtk3 (glibc distributions only)`. The exit code is 1 and it comes at once. Checked on Ubuntu 24.04.
- No display: see the table above for the error. A container, an SSH session and CI have no display; `xvfb-run ./app` (package `xvfb`) gives one.
- No display, checked on Ubuntu 24.04 with GTK installed: `gowt: No more handles [gtk_init_check() failed]`, exit code 1, at once.
- With `webview` in a container or VM without GPU access, stderr shows `libEGL warning: DRI3 error: Could not get DRI3 device` and `Overriding existing handler for signal 10. Set JSC_SIGNAL_FOR_GC if you want WebKit to use a different signal` (Ubuntu 24.04 under Xvfb). The program kept running; whether the page rendered was not checked.
- No WebKitGTK 4.1 for `webview`: `webview: WebKitGTK 4.1 is not available (...); install it: apt install libwebkit2gtk-4.1-0 | dnf install webkit2gtk4.1 | pacman -S webkit2gtk-4.1`.
- Alpine and other musl systems: `not found` at start, see above.

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
