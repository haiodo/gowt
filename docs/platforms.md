# Platforms

## For app authors

Same Go program, three native back ends. What differs:

| | macOS | Windows | Linux |
|---|---|---|---|
| Toolkit | AppKit | Win32 | GTK 3 (X11 or XWayland) |
| Cross-compile | `GOOS=darwin go build` | `GOOS=windows go build` | `GOOS=linux go build` (all with `CGO_ENABLED=0`, from any host) |
| Verified on | real Mac | real Windows 11 arm64 in a UTM virtual machine (since 2026-10-14) and CrossOver/Wine; Mica/Acrylic and two monitors not checked | Docker with Xvfb, not a real desktop |
| Web view | WKWebView | WebView2 Runtime (preinstalled on Windows 11) | WebKitGTK 4.1 |
| Glass, backdrop | Liquid Glass (macOS 26), translucent material | Mica/Acrylic (Windows 11), not checked by eye on the VM | none |
| System tray | `App.Tray` returns nil where the system has none; not checked per OS | same | same |
| App icon badge | `App.SetBadge`: the Dock tile badge | overlay on each window's taskbar button; not checked on the VM | count of the launcher entry (`com.canonical.Unity.LauncherEntry`: Ubuntu Dock, dash-to-dock, KDE Plasma), whole numbers only; the D-Bus signal checked in Docker, no dock there |
| Dark mode | whole UI | title bar; content with `look.SetDarkContent` | from the portal; not verified live |
| Menu bar | application menu bar | window menu bar | window menu bar |

Not in the facade on any OS: `Accessible` (a stub on all three OSes). Clipboard, drag and drop and `StyledText` are wrapped; see the caveats under Known gaps. On Windows the facade brings the manifest (visual styles, DPI awareness) itself, see [install.md](install.md).

## Test numbers

SWT's translated JUnit tests, `make test-swt`, counted from `tests/expected*.txt`:

| | pass | fail | skip |
|---|---|---|---|
| macOS (`expected.txt`) | 3818 | 4 (+3 flaky) | 69 |
| Windows (`expected_windows.txt`, recorded under Wine) | 3728 | 70 (+7 flaky) | 89 |
| Linux (`expected_linux.txt`) | 3840 | 2 (+1 flaky) | 58 |

3894 tests in all. JFace tests (`expected_jface*.txt`): 86 pass, 1 skip on macOS; 85 pass, 1 fail, 1 skip on Linux and Windows.

CI (GitHub Actions, v0.11.0, all green) builds and vets darwin, linux and windows on amd64 and arm64, runs `gen-check` and `xcheck`/`api-check`, and runs the GUI gates in Linux Docker under Xvfb: swt tests, jface tests, unit tests, and the GUI tests of every package that has them (`make check-gui`, `GOWT_GUI_TEST=1`, webview and the `qa/qatest` e2e among them). macOS and Windows GUI tests are not in CI, there are no runners with a display. Every non-pass line in the expected files has a reason; none is `UNDESCRIBED` now.

## How the ports differ

| | macOS (cocoa) | Windows (win32) | Linux (gtk 3) |
|---|---|---|---|
| Binding layer | `internal/cocoa`, translated from SWT PI/cocoa, purego | `internal/win32`, translated from SWT PI/win32, `syscall.SyscallN` | `internal/gtk`, generated from GIR by `tooling/girgen` plus hand glue, purego |
| Build | `GOOS=darwin go build` | `GOOS=windows go build` | `GOOS=linux go build` (all `CGO_ENABLED=0`) |
| Stand | the Mac itself | CrossOver (Wine) bottle `gowt`: `make win-swttest`, `make win-snap-check`; real Windows 11 arm64 in a UTM virtual machine on Apple silicon: `make win-testkit` builds a zip (swttest, winprobe, controlexample for amd64 and arm64, `run.cmd`), `swttest -compare` checks the returned result against `tests/expected_windows.txt` | Docker `gowt-linux`, Xvfb + noVNC: `make linux-run CMD="make test-swt"` |
| Snapshot references | `tests/snapshots` | `tests/snapshots_windows` (BitBlt capture in `internal/shot`) | `tests/snapshots_linux` |
| DPI / zoom | points (backing scale); one snapshot scale is recorded in `meta.txt` | manifest and `Display` request per-monitor DPI v2; the upstream SWT monitor-specific scaling (rescale on `WM_DPICHANGED`) is on unless `swt.autoScale.updateOnRuntime=false`; with it on, `swt.autoScale` other than `quarter` or `exact` (`integer`, `half`, `false`, a number) fails at `Display` creation with `ERROR_NOT_IMPLEMENTED` unless `swt.autoScale.updateOnRuntime` is `false` or `force`; Wine reports 96 whatever the setting | stand-in `DPIUtil`; snapshots exist only for scale 1 |

## Known gaps

Counted from the reasons in `tests/expected*.txt` and from the task notes. "Not verified" means no real run on that platform.

**All platforms**
- `Browser` (over package `webview`) does not support the Authentication event; the other events, `setUrl` with post data or headers, `setJavascriptEnabled` and the cookie statics are implemented (Browser tests that fail: macOS 2, Windows 61 plus 5 flaky, Linux none and 1 flaky). 16 Browser tests are skipped on macOS and 8 on Linux (mostly SWT assumptions for other engines).
- `CoolItem`: 4 tests (`getBounds`, `getPreferredSize`, `setControl`, `setSize`) are skipped everywhere: upstream comments the class out of `AllWidgetTests` ("Failing test"), and the emulated CoolBar gives a lone item the whole bar width.
- `Accessible` is a stub.
- Drag and drop was never driven with a real mouse (no test does); `FileTransfer` and `URLTransfer` native-to-Java tests are disabled on macOS, `URLTransfer` is not wrapped. Cross-process clipboard tests are skipped (no peer process), so only same-process round trips are tested.
- `StyledText`: style rendering tests are skipped on macOS (upstream bugs 553090, 536588); editing text with surrogate pairs (emoji) panics in `swt`.

**macOS**
- 5 failures: `TextLayout.test_bug568740_multilineTextStyle` (pixel search finds nothing), `Text.test_backspaceAndDelete` and `StyledText.test_backspaceAndDelete` (`Display.post` returns false for the key event), and two `Browser` tests (`BrowserFunction_callback_stackedCalls`, `StatusTextListener_hoverMouseOverLink`). The first three are not diagnosed.
- `Image.test_drawImageAtSize_reevaluatesSizabilityWhenFileNameChanges`: SVG files are not loaded by `swt`.
- 3 flaky: `Tree.test_Virtual` (`SetData` count timing), `Image.test_getBoundsInPixels` (multi-monitor DPI), `StyledText.test_notFixedLineHeightDoesntChangeLinePixelIfUnnecessary`.
- Table and Tree use the cell-based AppKit views; the view-based ones are not started ([view-based-table.md](view-based-table.md)).
- Deprecated AppKit that stays: `lockFocus` around `scrollRect:by:` (3 places); Carbon and `CPSSetProcessName` have no public replacement.
- `look.Classic()` was not confirmed for a binary without an app bundle; `-ldflags=-macsdk=15.0` is the reliable way.
- NSGlassEffectContainerView, NSBackgroundExtensionView and a unified tool bar do not fit SWT's view tree and are not offered.

**Windows** (gate file recorded under Wine; also run on a real Windows 11 arm64 in a UTM virtual machine)
- `tests/expected_windows.txt` has 70 failures and 7 flaky. 61 failures and 5 flaky are `Browser` tests: under CrossOver WebView2 starts, the page commits but never finishes loading (no GL), so `NavigationCompleted` does not arrive. Whether these pass on the VM is not recorded in the file.
- 9 failures outside Browser: `TextLayout` 4, `GC` 2, `Table` 2 (`test_Virtual`, `test_getItemHeight`), `Display` 1. The reasons name Wine (gdiplus, `DrawText`, `ScriptItemize`, `ScriptBreak`, `ScriptPlace`, `LVM_APPROXIMATEVIEWRECT`, `VkKeyScan`) or a font missing in the bottle. Flaky outside Browser: `StyledText.test_backspaceAndDelete`, `Text.test_consistency_Segments`.
- 41 of the 89 skips are SWT's own "alpha for foreground colors does not exist" assumption.
- Real Windows 11 arm64, 2026-10-14: GDI+ float arguments were wrong on windows/arm64. Go's `syscall.SyscallN` loads only x0-x7, AAPCS64 needs floats in s0-s7. About 31 `f32` calls in `internal/win32/gdip_manual.go` were affected (matrices, pens, font size, arcs, paths). Fixed in `61701c5`; the full `swttest` then gave 3740 passed, 65 failed, 89 skipped, the same as amd64.
- Real Windows 11 arm64, 2026-10-22: `swttest` 3741 passed, 64 failed, 89 skipped of 3894. The gate reports two problems: `GC.test_getFontMetrics_notAffectedByTransform` ("No more handles") and `Text`/`StyledText` `test_backspaceAndDelete` (flaky, passes alone). The `look` package: 25 tests pass. The root package: no regressions; `TestEnabledSwitchAndTextLog` fails (CRLF in `Text.TrimHead`) and `TestScrollOnScroll` is flaky, about 50% (timer granularity about 15.6 ms).
- Not checked on real Windows: Mica, Acrylic and rounded corners by eye; per-monitor DPI changes with two monitors (the VM has one display); the QA driver ([qa.md](qa.md)). Per-monitor DPI changes while running (`WM_DPICHANGED`) use upstream code and are untested. Manual test on Windows 10 1809+ with two monitors of different scale: start any `cmd/*` program on the 100% monitor, drag the window to the 150% one, expect controls, fonts and window size to rescale without restart; change the scale in Settings while it runs, expect the same. Wine reports 96 DPI.
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
