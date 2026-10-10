# Platforms

## For app authors

Same Go program, three native back ends. What differs:

| | macOS | Windows | Linux |
|---|---|---|---|
| Toolkit | AppKit | Win32 | GTK 3 (X11 or XWayland) |
| Cross-compile | `GOOS=darwin go build` | `GOOS=windows go build` | `GOOS=linux go build` (all with `CGO_ENABLED=0`, from any host) |
| Verified on | real Mac | CrossOver/Wine only, never real Windows | Docker with Xvfb, not a real desktop |
| Web view | WKWebView | WebView2 Runtime (preinstalled on Windows 11) | WebKitGTK 4.1 |
| Glass, backdrop | Liquid Glass (macOS 26), translucent material | Mica/Acrylic (Windows 11), untested on real hardware | none |
| System tray | `App.Tray` returns nil where the system has none; not checked per OS | same | same |
| Dark mode | whole UI | title bar; content with `look.SetDarkContent` | from the portal; not verified live |
| Menu bar | application menu bar | window menu bar | window menu bar |

Missing on every platform in the facade: Clipboard, drag and drop, `StyledText`, `Accessible` (swt level only, being integrated). On Linux `Accessible` (ATK) is a stub; on Windows it is a stub too. On Windows the program should import `_ "github.com/haiodo/gowt/winmanifest"`, see [install.md](install.md).

Test counts per OS (lines in `tests/expected*.txt`, SWT's translated JUnit tests, `make test-swt`):

| | pass | fail | skip |
|---|---|---|---|
| macOS (`expected.txt`) | 3492 | 30 (+2 flaky) | 26 |
| Windows (`expected_windows.txt`, Wine) | 3275 | 232 | 43 |
| Linux (`expected_linux.txt`) | 3497 | 33 | 20 |

JFace tests (`expected_jface*.txt`): 86 pass, 1 fail on macOS and Windows; 85 pass, 2 fail on Linux.

The sections below are the port status written earlier; the counts inside them predate the table above and are kept as history.

Source of truth for numbers: `tests/expected.txt` (darwin), `tests/expected_windows.txt`, `tests/expected_linux.txt` (reasons per non-pass line) and
`tooling/j2go/README.md` rounds 19-20. The same translator output (`swt/*.go` shared + `_darwin`/`_windows`/`_linux` files) builds on all three; one `swt` API.

| | macOS (cocoa) | Windows (win32) | Linux (gtk 3) |
|---|---|---|---|
| Binding layer | `internal/cocoa`, translated from SWT PI/cocoa, objc runtime without cgo | `internal/win32`, translated from SWT PI/win32, `syscall.SyscallN` (no cgo) | `internal/gtk`, generated from GIR by `tooling/girgen` + hand glue, purego (no cgo) |
| Build | `make gen && make` | `GOOS=windows GOARCH=amd64 go build ./...` | `GOOS=linux CGO_ENABLED=0 go build ./...` |
| `test-swt` gate | 3309 pass / 21 fail / 1 flaky / 10 skip | 3252 pass / 46 fail / 43 skip | 2950 pass / 385 fail / 6 skip |
| Stand | the Mac itself (`make test-swt`, `make snap-check`) | CrossOver/Wine bottle `gowt` (`make win-swttest`; `make win-probe` is console-only); not run on real Windows | Docker `gowt-linux`, Xvfb + noVNC (`make linux-run CMD="make test-swt"`) |
| Snapshots (`tests/snapshots`) | yes, only here (`snap_darwin.go`) | `snap_windows.go`: BitBlt capture in `internal/shot` (`make win-snap-check`, `tests/snapshots_windows`) | `snap_linux.go` stub |
| DPI / zoom | points semantics (backing scale) | per-monitor DPI awareness in the manifest (`winmanifest`), but the port still uses the system DPI at start: device zoom = "integer" autoscale of the system DPI; no per-monitor runtime rescaling | stand-in `DPIUtil` (`swt/*_manual_linux.go`), not verified at other scales |

## Known gaps

**macOS** (22 non-pass in `expected.txt`)
- Java reflection on private members (`getDeclaredField/Method`) has no translator rule: 4 + 2 tests.
- SVG images are not loaded (Go stdlib codecs only): 3 tests.
- 10 skips are SWT's own assumptions/`@Disabled`; `Tree.test_Virtual` is flaky (SetData count timing).

**Windows**
- Never run on real Windows; all results are Wine (CrossOver), where DPI is 96 whatever the setting, so zoom above 100 is untested.
- 38 of 43 skips: SWT's "alpha for foreground colors does not exist on Win32" assumption.
- Not analysed failures (46): `java.lang.Character.isAlphabetic` / iterator `next` unresolved calls (14), reflection (6), Combo/CCombo `setItems`, ImageLoader on some streams, assertions that differ under Wine.
- Not translated: drag and drop, OLE; `Browser` needs the WebView2 engine of package `webview`; `Accessible` (MSAA) is a stub; `TextLayout` (Uniscribe) untested; the generic multi-zoom image-handle helpers are panic markers (paths/patterns/transforms at non-100 zoom).
- Dark-mode ordinals are unavailable.

**Linux**
- DateTime: `java.text` date formats are unresolved calls (panic), 318 of the 385 failures (`widgets_datetime_linux.go`). Other: `assert` statements unsupported in the translated gtk code, 6 SIGSEGV, reflection, image format.
- GTK 3 only: GTK 4 names compile but panic when called; GDBus paths not done; X11 backend (Wayland only partly, no input/snapshots on the stand).
- `Accessible`, `WidgetSpy`, CSS theming are stubs/stand-ins (`swt/*_manual_linux.go`; `swt/gtkres/*.css` are empty).
- Most failure reasons in `expected_linux.txt` are still `UNDESCRIBED`.

## Cross-platform notes
- Public API is compared by `make api-check` (`apidump -check` over darwin/windows/linux, `tooling/apidump/platform-only.txt` lists SWT's own per-OS API and the hand-stubbed IME/Tracker).
- Platform-only SWT API: `IME`/`Tracker` (win32), `GetPrimaryMonitorDisplay`, `DisplayExtractFreeGError`, `Image.Internal_gtk_refreshImageForZoom` (gtk).
- The shared (unsuffixed) files are generated only by the cocoa run; win32/gtk runs write their `_<goos>` files (README Round 20).
- The gtk binding must stay clean-room: nothing from SWT's `Eclipse SWT PI/gtk` (LGPL) is read or reused.
