# SWT JUnit tests: results (TSK-2026-09-23-051, -052, -053)

`make test-swt` on macOS arm64, SWT `af630a9093`. The run is gated by `tests/expected.txt` (below).

**Total: 1822 tests - 1662 passed, 149 failed, 11 skipped.** Before task 052 (graphics, layout, events only):
352 tests - 292 passed, 55 failed, 5 skipped. The graphics failures are now 50 (was 55): the
`NSIntersectionRect`/`PtInRgn` binding fix (below) repaired GC clipping and Region.contains.

Translated: `graphics`, `layout` (except `BorderLayout`, see below), `events`, and the widget classes
`Widget`, `Control`, `Scrollable`, `Composite`, `Canvas`, `Decorations`, `Shell`, `Display`, `Button`, `Label`,
`Text`, `Tree`, `Table`, `Combo`, `TabFolder`, `Group`, `Menu`, `Caret`, `ScrolledComposite`, plus `SwtTestUtil`,
`ImageTestUtil`, `CapturedOutput`, `ConsistencyUtility`, `ImageDataTestHelper`. `Widget`, `Control`, `Scrollable`
and `Decorations` are abstract: their tests run once per concrete subclass (so one Widget test counts in every
widget row). `CoolBar` was neither translated nor excluded: in these classes it only appears as a string key in
`ConsistencyUtility`, and j2go parses them without it. SWT has no `Test_..._SashForm`; `SashForm` is not
covered by any JUnit test.

Not translated: `Test_org_eclipse_swt_layout_BorderLayout` (`BorderLayout`/`BorderData` are not in `swt`; its
`MockControl extends Canvas` subclasses a widget from another Go package). Not attempted: the other widget
tests (`List`, `Sash`, `MenuItem`, `TreeItem`, `TableItem`, ... ) - not in the task list.

## Per class

| Class | Pass | Fail | Skip |
|---|---:|---:|---:|
| events_* (16 classes, 2 tests each) | 32 | 0 | 0 |
| graphics_Color | 27 | 0 | 0 |
| graphics_Cursor | 7 | 0 | 0 |
| graphics_DeviceData | 1 | 0 | 0 |
| graphics_Font | 7 | 2 | 0 |
| graphics_FontData | 6 | 5 | 0 |
| graphics_FontMetrics | 8 | 0 | 0 |
| graphics_GC | 62 | 11 | 1 |
| graphics_Image | 17 | 18 | 3 |
| graphics_ImageData | 15 | 6 | 0 |
| graphics_ImageLoader | 3 | 5 | 0 |
| graphics_ImageLoaderEvent | 1 | 1 | 0 |
| graphics_PaletteData | 5 | 0 | 0 |
| graphics_Path | 3 | 0 | 1 |
| graphics_Pattern | 16 | 0 | 0 |
| graphics_Point | 5 | 0 | 0 |
| graphics_RGB | 6 | 0 | 0 |
| graphics_RGBA | 6 | 0 | 0 |
| graphics_Rectangle | 17 | 0 | 0 |
| graphics_Region | 21 | 0 | 0 |
| graphics_TextLayout | 18 | 2 | 0 |
| graphics_Transform | 4 | 0 | 0 |
| layout_FormAttachment | 7 | 0 | 0 |
| layout_GridData | 3 | 0 | 0 |
| widgets_Button | 97 | 3 | 2 |
| widgets_Canvas | 91 | 5 | 0 |
| widgets_Caret | 17 | 1 | 0 |
| widgets_Combo | 130 | 11 | 0 |
| widgets_Composite | 86 | 5 | 0 |
| widgets_Display | 50 | 11 | 3 |
| widgets_Group | 89 | 5 | 0 |
| widgets_Label | 83 | 3 | 0 |
| widgets_Menu | 28 | 1 | 0 |
| widgets_ScrolledComposite | 96 | 5 | 0 |
| widgets_Shell | 131 | 8 | 1 |
| widgets_TabFolder | 96 | 11 | 0 |
| widgets_Table | 125 | 11 | 0 |
| widgets_Text | 128 | 5 | 0 |
| widgets_Tree | 118 | 14 | 0 |

Skips: the tests' own `assumeTrue/assumeFalse`/`@Disabled*` for cocoa, plus `Display.test_setCursorLocation*`
(`@TempDir`-style parameter injection, not supported by the runner).

## Failures by cause (all recorded in `tests/expected.txt` with their reason)

| Cause | Count |
|---|---:|
| null String is `""` in the port: guard `s == null` is dead (README "Round 11 null-string") | 36 |
| `TestInfo.getTestMethod` (`Optional<Method>`) not translated: event-consistency tests | 21 |
| anonymous class not translated (subclass of a widget from another package, or `Thread`) | 20 |
| `java.nio.file.Path`/`@TempDir` not translated | 15 |
| HiDPI image path not translated (TSK-2026-09-23-048) | 14 |
| `SWT.Activate` not received in 3s (`waitShellActivate`), then `PrintStream.println` untranslated; activation cause not diagnosed | 11 |
| `widgets.List` not translated | 8 |
| generic method `Display.syncCall` / `SwtCallable` lambda not translated | 7 |
| other JDK surface (Locale, WeakReference, CountDownLatch, Thread, `System.setProperty`, Random, StringBuilder, lambdas) | 12 |
| image codec differences (listener never fires; wrong exception type) | 2 |
| port bug not diagnosed (`TextLayout.getSegments`), pixel test `bug568740_multilineTextStyle` | 2 |
| local class declaration not translated | 1 |

The null-String and `TestInfo` groups are the two large ones. The first is a documented contract of the port
(a test passing `null` for a String expects `ERROR_NULL_ARGUMENT`; `""` is a valid value, so no guard can be
emitted); it is not a translator bug. The second needs `Optional`/`java.lang.reflect.Method` (JDK surface).

## Regression gate: `tests/expected.txt`

One line per test, tab separated: `pass Class.method`, `fail Class.method reason`, `skip Class.method reason`,
`flaky Class.method reason` (either outcome accepted). `make test-swt` exits 1 when

- a test listed as `pass` no longer passes (or is skipped);
- a test fails that is not listed, is listed as `skip`, or whose reason starts with `UNDESCRIBED`;
- (full run only) a listed test was not run.

A test that does better than listed (a failure now passes, a new test passes) does not fail the run; the gate
prints how many and `make test-swt-update` records them. Refresh after `make gen`: see README "Round 13 widget
tests".
