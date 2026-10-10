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

Not in the facade on any OS: `Accessible` (a stub on all three OSes). Clipboard, drag and drop and `StyledText` are wrapped; see the caveats under Known gaps. On Windows import `_ "github.com/haiodo/gowt/winmanifest"`, see [install.md](install.md).

## Test numbers

SWT's translated JUnit tests, `make test-swt`, counted from `tests/expected*.txt`:

| | pass | fail | skip |
|---|---|---|---|
| macOS (`expected.txt`) | 3492 | 30 (+2 flaky) | 26 |
| Windows (`expected_windows.txt`, Wine) | 3275 | 232 | 43 |
| Linux (`expected_linux.txt`) | 3497 | 33 | 20 |

JFace tests (`expected_jface*.txt`): 86 pass, 1 fail on macOS and Windows; 85 pass, 2 fail on Linux. Every non-pass line in the expected files has a reason; none is `UNDESCRIBED` now.

## How the ports differ

| | macOS (cocoa) | Windows (win32) | Linux (gtk 3) |
|---|---|---|---|
| Binding layer | `internal/cocoa`, translated from SWT PI/cocoa, purego | `internal/win32`, translated from SWT PI/win32, `syscall.SyscallN` | `internal/gtk`, generated from GIR by `tooling/girgen` plus hand glue, purego |
| Build | `GOOS=darwin go build` | `GOOS=windows go build` | `GOOS=linux go build` (all `CGO_ENABLED=0`) |
| Stand | the Mac itself | CrossOver (Wine) bottle `gowt`: `make win-swttest`, `make win-snap-check` | Docker `gowt-linux`, Xvfb + noVNC: `make linux-run CMD="make test-swt"` |
| Snapshot references | `tests/snapshots` | `tests/snapshots_windows` (BitBlt capture in `internal/shot`) | `tests/snapshots_linux` |
| DPI / zoom | points (backing scale); one snapshot scale is recorded in `meta.txt` | the manifest requests per-monitor DPI v2, but the port keeps one scale from the system DPI at start; Wine reports 96 whatever the setting | stand-in `DPIUtil`; snapshots exist only for scale 1 |

## Known gaps

Counted from the reasons in `tests/expected*.txt` and from the task notes. "Not verified" means no real run on that platform.

**All platforms**
- `Browser` (over package `webview`) does not support the OpenWindow, VisibilityWindow, CloseWindow and StatusText events, `setUrl` with post data or headers, `setJavascriptEnabled` or the cookie statics. On macOS and on Linux 20 Browser tests fail each: 13 for these reasons, 1 because a `System.setOut` capture is not translated, 6 "Round 23, not diagnosed" (BrowserFunction callbacks, a nil pointer). 16 Browser tests are skipped on macOS and 14 on Linux (mostly SWT assumptions for other engines).
- `CoolItem`: 4 tests (`getBounds`, `getPreferredSize`, `setControl`, `setSize`) are skipped everywhere: upstream comments the class out of `AllWidgetTests` ("Failing test"), and the emulated CoolBar gives a lone item the whole bar width.
- `Accessible` is a stub.
- Drag and drop was never driven with a real mouse (no test does); `FileTransfer` and `URLTransfer` native-to-Java tests are disabled on macOS, `URLTransfer` is not wrapped. Cross-process clipboard tests are skipped (no peer process), so only same-process round trips are tested.
- `StyledText`: style rendering tests are skipped on macOS (upstream bugs 553090, 536588); editing text with surrogate pairs (emoji) panics in `swt`.

**macOS**
- `TextLayout.test_bug568740_multilineTextStyle` (pixel search finds nothing), `Text.test_backspaceAndDelete` (`Display.post` returns false for the key event): not diagnosed.
- `Image.test_drawImageAtSize_reevaluatesSizabilityWhenFileNameChanges`: SVG files are not loaded by `swt`.
- Flaky: one `Tree` test (`SetData` count timing) and one multi-monitor DPI test.
- Table and Tree use the cell-based AppKit views; the view-based ones are not started ([view-based-table.md](view-based-table.md)).
- Deprecated AppKit that stays: `lockFocus` around `scrollRect:by:` (3 places); Carbon and `CPSSetProcessName` have no public replacement.
- `look.Classic()` was not confirmed for a binary without an app bundle; `-ldflags=-macsdk=15.0` is the reliable way.
- NSGlassEffectContainerView, NSBackgroundExtensionView and a unified tool bar do not fit SWT's view tree and are not offered.

**Windows** (Wine only; nothing was run on real Windows)
- 209 of the 232 failures are Browser tests: the CrossOver bottle has no WebView2 Runtime. The WebView2 code (COM callbacks in Go, no `WebView2Loader.dll`) has never run.
- 38 of the 43 skips are SWT's own "alpha for foreground colors does not exist on Win32" assumption.
- 23 failures outside Browser, mostly "not analysed": `Image` 7, `TextLayout` 4 (three of them carry a stale reason about `isAlphabetic` and need a rerun of `make win-swttest-update`; one hits an untranslated local class), `GC` 3, `Table` 2 (`test_Virtual`, `test_getItemHeight`), and one each of `CTabFolder` and `Display` (Wine's `SendInput` does not deliver `Display.post`).
- Mica, Acrylic and rounded corners were not seen on Windows 11. Per-monitor DPI changes while running (`WM_DPICHANGED`) are not handled: `Display` puts the thread in system-aware mode and `DPIUtil` holds one scale. Wine reports 96 DPI, so zoom above 100% is untested.
- Dark content is opt-in (`look.SetDarkContent`) and uses undocumented uxtheme exports.

**Linux** (2 failures outside Browser)
- `CTabFolder.test_childControlOverlap`: at the step topRight `RIGHT | WRAP` after `showChevron` with minimize and maximize visible, items and controls fill the 255 px folder exactly and the chevron, moved to `lastItem.x + width + SPACING`, overlaps the topRight label by 6 px. The widths come from this stand's fonts.
- `Shell.test_Issue450_NoShellActivateOnSetFocus`: openbox gives no focus to the second shell when the active one is disposed, and `Shell.setActive` does nothing while no shell is active.
- `DateTime` with `SWT.BORDER` loses the bit (`checkBorder`, no scrolled handle), which fails `TestUnitDateTimeFactory.createsDateTime` in JFace.
- GTK 3 only. The theme and accent colour come from xdg-desktop-portal; this was not checked against a live portal.
- WebKitGTK does not report the navigation frame, so the navigation policy always gets `mainFrame=true`.
- The stand is X11 only (Xvfb); there is no Wayland run.
- `swt/gtkres/*.css` are 74 bytes or less, so SWT's CSS theming fixes are not applied.

## Cross-platform notes

- `make api-check` compares the exported `swt` API of darwin, windows and linux (`tooling/apidump/platforms.txt`); `tooling/apidump/platform-only.txt` lists what SWT itself declares for some platforms only (for example `IME` and `Tracker`, and `GetPrimaryMonitorDisplay` for gtk).
- The shared (unsuffixed) files in `swt/` are generated by the cocoa run; the win32 and gtk runs write their `_<goos>` files only.
- The gtk binding is clean room: nothing from SWT's `Eclipse SWT PI/gtk` (LGPL) is read or reused.
