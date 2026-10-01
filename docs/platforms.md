# Platforms

Source of truth for numbers: `tests/expected.txt` (darwin), `tests/expected_windows.txt`, `tests/expected_linux.txt` (3342 lines each, reasons per non-pass line) and
`tooling/j2go/README.md` rounds 19-20. The same translator output (`swt/*.go` shared + `_darwin`/`_windows`/`_linux` files) builds on all three; one `swt` API.

| | macOS (cocoa) | Windows (win32) | Linux (gtk 3) |
|---|---|---|---|
| Binding layer | `internal/cocoa`, translated from SWT PI/cocoa, objc runtime without cgo | `internal/win32`, translated from SWT PI/win32, `syscall.SyscallN` (no cgo) | `internal/gtk`, generated from GIR by `tooling/girgen` + hand glue, purego (no cgo) |
| Build | `make gen && make` | `GOOS=windows GOARCH=amd64 go build ./...` | `GOOS=linux CGO_ENABLED=0 go build ./...` |
| `test-swt` gate | 3309 pass / 21 fail / 1 flaky / 10 skip | 3252 pass / 46 fail / 43 skip | 2950 pass / 385 fail / 6 skip |
| Stand | the Mac itself (`make test-swt`, `make snap-check`) | CrossOver/Wine bottle `gowt` (`make win-swttest`; `make win-probe` is console-only); not run on real Windows | Docker `gowt-linux`, Xvfb + noVNC (`make linux-run CMD="make test-swt"`) |
| Snapshots (`tests/snapshots`) | yes, only here (`snap_darwin.go`) | `snap_windows.go`: PrintWindow in `internal/shot` (`make win-snap-check`, `tests/snapshots_windows`) | `snap_linux.go` stub |
| DPI / zoom | points semantics (backing scale) | system DPI aware (manifest), device zoom = "integer" autoscale of the system DPI; no per-monitor runtime rescaling | stand-in `DPIUtil` (`swt/*_manual_linux.go`), not verified at other scales |

## Known gaps

**macOS** (22 non-pass in `expected.txt`)
- Java reflection on private members (`getDeclaredField/Method`) has no translator rule: 4 + 2 tests.
- SVG images are not loaded (Go stdlib codecs only): 3 tests.
- 10 skips are SWT's own assumptions/`@Disabled`; `Tree.test_Virtual` is flaky (SetData count timing).

**Windows**
- Never run on real Windows; all results are Wine (CrossOver), where DPI is 96 whatever the setting, so zoom above 100 is untested.
- 38 of 43 skips: SWT's "alpha for foreground colors does not exist on Win32" assumption.
- Not analysed failures (46): `java.lang.Character.isAlphabetic` / iterator `next` unresolved calls (14), reflection (6), Combo/CCombo `setItems`, ImageLoader on some streams, assertions that differ under Wine.
- Not translated: Browser/WebView2, drag and drop, OLE; `Accessible` (MSAA) is a stub; `TextLayout` (Uniscribe) untested; the generic multi-zoom image-handle helpers are panic markers (paths/patterns/transforms at non-100 zoom).
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
