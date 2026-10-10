# Packaging and distribution

A gowt program is one Go executable; the OS libraries are loaded at run time. This page covers what to put around it per OS. What the user must have installed is in [install.md](install.md).

## Cross-compiling

No cgo and no C toolchain, so any host builds for any target:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o app.exe .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o app-linux .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o app-macos .
```

`make consumer-check` builds an outside module for darwin, windows and linux this way. Building is checked everywhere; running was checked on macOS, on Windows only under Wine, and on Linux only in Docker with Xvfb.

For a smaller file add `-ldflags='-s -w' -trimpath`; `make release` does this for the demos and keeps the pclntab so stack traces still work.

## macOS: app bundle

A bare executable runs, but Finder and the Dock list it by file name and a double click opens Terminal. An `.app` fixes that. `tooling/darwin/mkapp.sh <binary> <out-dir> [name]` builds one, and `make app APPS="..."` runs it over release builds of `cmd/` programs. The bundle layout:

```
App.app/Contents/Info.plist
App.app/Contents/MacOS/<executable>
App.app/Contents/Resources/AppIcon.icns   (optional)
```

The `Info.plist` that `mkapp.sh` writes has these keys; copy the script and edit them for your app:

| Key | Value in `mkapp.sh` |
|---|---|
| `CFBundleExecutable` | the executable's file name |
| `CFBundleIdentifier` | `com.github.haiodo.gowt.<executable>`: use your own reverse-DNS id |
| `CFBundleName`, `CFBundleShortVersionString`, `CFBundleVersion` | display name, `0.1`, `1` |
| `CFBundlePackageType` | `APPL` |
| `LSMinimumSystemVersion` | `13.0`, the minimum Go 1.27 writes into the binary |
| `NSHighResolutionCapable` | `true` |
| `NSPrincipalClass` | `NSApplication` |
| `UIDesignRequiresCompatibility` | only with `CLASSIC_LOOK=1`, see below |

**Icon.** `mkapp.sh` does not add one. Put an `.icns` file in `Contents/Resources` and add `<key>CFBundleIconFile</key><string>AppIcon</string>` to the plist. Make the `.icns` from an iconset folder with Apple's `iconutil -c icns AppIcon.iconset`. This step was not run here.

**Signing.** `mkapp.sh` signs ad hoc (`codesign --force --sign -`): arm64 macOS refuses unsigned code, and an ad-hoc signature is enough to run on the machine that built it. A copy sent to another Mac is quarantined by the browser and Gatekeeper rejects it unless it is signed with a Developer ID Application certificate and notarized. The Apple tools for that are `codesign --options runtime --timestamp --sign "Developer ID Application: ..."` and `xcrun notarytool submit ... --wait`, then `xcrun stapler staple App.app`. Neither was run for gowt and whether the hardened runtime needs entitlements for the libraries gowt loads was not checked.

**Look: Liquid Glass or classic.** Go 1.27 records SDK 26.2 in the binary, so on macOS 26 AppKit draws the new Liquid Glass chrome without any call. To keep the earlier look there are two ways:

- Link against an older SDK: `go build -ldflags=-macsdk=15.0`. This is the reliable one and works for a bare binary as well as a bundle.
- Set `UIDesignRequiresCompatibility` in the plist (`CLASSIC_LOOK=1 tooling/darwin/mkapp.sh ...`), or call `look.Classic()` before `gowt.Run`. For a bare binary with no `Info.plist` AppKit may not read it; this was only checked by eye. Both ways rely on the system keeping its compatibility behaviour for older SDKs; whether a later macOS does is outside gowt's control.

## Windows

- **Manifest.** Package `winmanifest` links `manifest_windows_{amd64,arm64}.syso` with comctl32 v6 (visual styles, `Link`) and per-monitor DPI awareness v2. Programs written with the `gowt` facade get it automatically, because `gowt` imports it. A program on `swt` directly adds `import _ "github.com/haiodo/gowt/winmanifest"`. `make winmanifest` rebuilds the objects with `tooling/mksyso` (pure Go, `-arch`, `-manifest`, `-o`). A second `.syso` carrying its own manifest makes the linker fail with a duplicate resource: use one or the other.
- **DPI.** The manifest requests per-monitor DPI v2, but the port keeps the scale it read from the system DPI at start; moving a window between monitors with different scales is not handled (`WM_DPICHANGED`). Under Wine the system reports 96 whatever its setting.
- **Icon and version info.** `mksyso` writes only the manifest. A file icon and version resource need a separate `.syso` from a tool such as `rsrc` or `goversioninfo`, and then that `.syso` has to include the same manifest (see above), since two manifests collide. Not tried here. A window icon at run time: `w.Unwrap().SetImage(img.Unwrap())` with a `*gowt.Image`.
- **Runtime.** Windows 10 or later (not verified on real hardware); WebView2 Runtime only for the `webview` package, no `WebView2Loader.dll`.
- `-ldflags=-H=windowsgui` makes the executable a GUI-subsystem program with no console window. Not run here.

## Linux

- **Libraries.** GTK 3 (`libgtk-3.so.0` and its dependencies) and an X11 or XWayland display. The `webview` package also needs WebKitGTK 4.1 (`libwebkit2gtk-4.1-0` on Debian and Ubuntu). Packages per distribution are in [install.md](install.md); the WebKitGTK package names were not verified for every distribution. Without GTK the program returns an error naming the missing library; without a display `gowt.Run` returns an error ending in `No more handles [gtk_init_check() failed]`.
- **Desktop file.** The system menu lists a program from a `.desktop` file, for example `~/.local/share/applications/myapp.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=My App
Exec=/opt/myapp/myapp
Icon=myapp
Categories=Utility;
StartupWMClass=myapp
```

`gowt.SetAppName("myapp")` sets the name the window manager sees as the WM class; `StartupWMClass` must match it for the menu entry and the window to group together. Icons go to `~/.local/share/icons/hicolor/<size>/apps/myapp.png`. This was not tried on a real desktop; the stand is Xvfb with openbox.
- **Wayland.** The port uses GTK 3 over X11 or XWayland.
- **glibc.** A Linux build is a dynamically linked ELF that needs `libc.so.6` and a glibc dynamic loader (`readelf -d` on the Docker stand's build shows `libc.so.6` as the only `NEEDED` entry). GTK itself is loaded with `dlopen` at start, so it is not a link-time dependency. A musl system such as Alpine was not checked.
