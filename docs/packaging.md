# Packaging and distribution

<img src="img/art/packaging.png" alt="Gopher tying a ribbon on a gift box next to parcels" width="220" align="right">

A gowt program is one Go executable; the OS libraries are loaded at run time. This page covers what to put around it per OS. What the user must have installed is in [install.md](install.md).

<br clear="right">

## Cross-compiling

No cgo and no C toolchain, so any host builds for any target:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o app.exe .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o app-linux .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o app-macos .
```

`make consumer-check` builds an outside module for darwin, windows and linux this way. Building is checked everywhere; running was checked on macOS, on Windows under Wine and on a real Windows 11 arm64 virtual machine (the swttest gate and the GUI tests; a packaged or installed app was not run there), and on Linux only in Docker with Xvfb.

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

- **Libraries.** GTK 3 (`libgtk-3.so.0` and its dependencies) and a Wayland or X11 display. The `webview` package also needs WebKitGTK 4.1 (`libwebkit2gtk-4.1-0` on Debian and Ubuntu). Packages per distribution are in [install.md](install.md); WebKitGTK was checked on ubuntu:24.04 and debian:12 only. Without GTK the program returns an error naming the missing library; without a display `gowt.Run` returns an error ending in `No more handles [gtk_init_check() failed]`.
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

`gowt.SetAppName("myapp")` sets the name the window manager sees as the WM class; `StartupWMClass` must match it for the menu entry and the window to group together. The same name names the launcher entry that `app.SetBadge` updates (`application://myapp.desktop`), so the badge count shows only when the desktop file is `myapp.desktop`; without `SetAppName` the name is the executable's file name. gowt does not use `GApplication`, so there is no application id other than this name, except the one an app passes to `gowt.SingleInstance` (see below). Icons go to `~/.local/share/icons/hicolor/<size>/apps/myapp.png`. This was not checked against a launcher on a real desktop.
- **Wayland.** The port uses GTK 3. On a Wayland session GTK uses native Wayland; `GDK_BACKEND=x11` gives Xwayland. The gate `tests/expected_linux_desktop.txt` was recorded on native Wayland (Ubuntu 26.04 GNOME); under Xwayland `CTabFolder.test_chevronAppearanceChanged` and the `backspaceAndDelete` tests fail differently.
- **glibc.** A Linux build is a dynamically linked ELF that needs `libc.so.6` and a glibc dynamic loader (`readelf -d` on the Docker stand's build shows `libc.so.6` as the only `NEEDED` entry). GTK itself is loaded with `dlopen` at start, so it is not a link-time dependency. A musl system such as Alpine is not supported: the binary stops with `not found` there.

## File association and document icon

What the program does with a file it is asked to open is `App.OnOpenFiles`, see [events.md](events.md#opening-files). This section makes the OS offer the program for a file type and show a document icon. Writing the files is covered here; installing them (copying to the right place, registering) is yours. `tooling/assoc` writes the Windows and Linux files: `go run ./tooling/assoc -kind reg|desktop|service|mime ...` prints one file to standard output.

**macOS.** `tooling/darwin/mkapp.sh` writes `CFBundleDocumentTypes` when `DOC_EXTS` is set:

```sh
DOC_EXTS="tnb tenniarb" DOC_NAME="Tenniarb document" DOC_ICON=doc.icns DOC_UTI=com.example.tenniarb \
  tooling/darwin/mkapp.sh ./myapp out "My App"
```

| Variable | Effect |
|---|---|
| `DOC_EXTS` | space-separated extensions without dots: `CFBundleTypeExtensions`, role `Editor`, `LSHandlerRank` `Owner` |
| `DOC_NAME` | `CFBundleTypeName`, default `<name> Document` |
| `DOC_ICON` | a `.icns` file, copied to `Contents/Resources`, named in `CFBundleTypeIconFile` |
| `DOC_UTI` | a reverse-DNS type id: adds `LSItemContentTypes` and a `UTExportedTypeDeclarations` entry (conforms to `public.data`) |

The script runs `plutil -lint` on the result. Launch Services reads the plist when the bundle is first seen (copy it to `/Applications` or run `lsregister -f App.app`); after that a double click, Open With and `open -a App.app file` reach `OnOpenFiles`, at launch and while the app runs. Whether the app becomes the default for the type is the user's choice in Finder (Get Info, Change All), `LSHandlerRank` only ranks it. Checked: the plist lints, `open -a` with a file starts the bundle and the path reaches the handler, and a second `open -a` while it runs does too (macOS 27 on a second Mac, `examples/openfiles`). Not checked: registering the type with `lsregister` and a double click on a file of that type, and how the icon looks.

**Windows.** Per-user registration lives under `HKCU\Software\Classes`; no administrator rights are needed:

```
HKCU\Software\Classes\.tnb                                  (Default) = Org.App.Doc
HKCU\Software\Classes\Org.App.Doc                           (Default) = App document
HKCU\Software\Classes\Org.App.Doc\DefaultIcon               (Default) = C:\App\doc.ico   (or C:\App\app.exe,1)
HKCU\Software\Classes\Org.App.Doc\shell\open\command        (Default) = "C:\App\app.exe" "%1"
```

`go run ./tooling/assoc -kind reg -exe 'C:\App\app.exe' -ext .tnb,.tenniarb -progid Org.App.Doc -name 'App document' -icon 'C:\App\doc.ico' > assoc.reg` writes the same as a `.reg` file (UTF-16 with a BOM when a name is not ASCII), imported with `reg import assoc.reg` or a double click. Explorer starts `app.exe "C:\path\file.tnb"`, which `OnOpenFiles` gets as a command-line argument; selecting several files starts one process each, so call `gowt.SingleInstance` to get them in one window. The icon must be an `.ico` file; an icon inside the executable needs a resource `.syso` (see above). A file type that the user has chosen another program for in Explorer keeps that choice: Windows protects it with a `UserChoice` key that a script cannot set. The generator output is tested; the import and a double click were not run here, the argument delivery and the forwarding to a running instance were (Windows 11 arm64 VM, `examples/openfiles`).

**Linux.** Two files: a `.desktop` entry with `MimeType=` and `%U` in `Exec`, and, for an own type, a shared-mime-info XML:

```ini
[Desktop Entry]
Type=Application
Name=My App
Exec=/opt/myapp/myapp %U
MimeType=application/x-myapp;
```

```sh
go run ./tooling/assoc -kind desktop -id org.example.MyApp -name 'My App' -exe /opt/myapp/myapp -mime application/x-myapp -icon myapp > org.example.MyApp.desktop
go run ./tooling/assoc -kind mime -mime application/x-myapp -name 'My App document' -ext tnb,tenniarb > myapp-mime.xml
install -D myapp-mime.xml ~/.local/share/mime/packages/myapp.xml && update-mime-database ~/.local/share/mime
install -D org.example.MyApp.desktop ~/.local/share/applications/ && update-desktop-database ~/.local/share/applications
```

A file manager runs `Exec` with `%U`, the files as URIs; the program takes `file:` URIs and plain paths in its arguments alike. For a type that already exists (`image/png`) leave the XML out. The document icon is a theme icon named after the MIME type with the slash as a dash, `application-x-myapp`, in `~/.local/share/icons/hicolor/<size>/mimetypes/`.

To get a second launch into the running window, call `gowt.SingleInstance("org.example.MyApp")` with the same id (a D-Bus well-known name: reverse-DNS, letters, digits, `_` and `-`). It owns that name on the session bus and serves the freedesktop `org.freedesktop.Application` interface (`Open`, `Activate`, `ActivateAction` is accepted and ignored), so `Exec=app %U` started twice forwards the second one's files. Adding `DBusActivatable=true` (`-dbus`) makes the desktop skip `Exec` for a running app and call `Open` on the name instead; it needs a D-Bus service file, `go run ./tooling/assoc -kind service -id org.example.MyApp -exe /opt/myapp/myapp`, installed as `~/.local/share/dbus-1/services/org.example.MyApp.service`. The `.desktop` file should be named after the id. Without `SingleInstance` a Linux app gets command-line files only. The bus client is gowt's own, written from the D-Bus specification (EXTERNAL authentication, little-endian messages); `gdbus call` of `Open` and `Activate` and `gdbus introspect` were run against it in Docker with a real `dbus-daemon`. Not checked on a real desktop: a file manager's Open With, the `DBusActivatable` start of a program that is not running, the icon lookup.
