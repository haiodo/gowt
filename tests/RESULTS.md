# SWT JUnit tests: results (TSK-2026-09-23-051, -052, -053)

`make test-swt` on macOS arm64, SWT `af630a9093`. The run is gated by `tests/expected.txt` (below).

**Total: 3341 tests - 3310 passed, 21 failed, 10 skipped** (Round 17 added the tests of Scale, Slider, Spinner, ProgressBar, Link, ToolBar, ToolItem, ExpandBar, ExpandItem, CoolBar, CoolItem, DateTime, FileDialog, DirectoryDialog and custom CLabel, CCombo, CTabFolder, CTabItem; before: 1805 / 8 / 9 of 1822) (Round 15; before it 1728 / 83 / 11). Before the merge of TSK-051 and TSK-052/053:
widget tests alone 1662 / 149 / 11; graphics, layout and events alone 325 / 22 / 5 (TSK-051 seed: 292 / 55 / 5).
`Tree.test_Virtual` is `flaky` (SetData count timing).

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
| custom_CCombo | 130 | 2 | 0 |
| custom_CLabel | 96 | 0 | 0 |
| custom_CTabFolder | 120 | 7 | 1 |
| custom_CTabItem | 15 | 0 | 0 |
| events_* (16 classes, 2 tests each) | 32 | 0 | 0 |
| graphics_Color | 27 | 0 | 0 |
| graphics_Cursor | 7 | 0 | 0 |
| graphics_DeviceData | 1 | 0 | 0 |
| graphics_Font | 9 | 0 | 0 |
| graphics_FontData | 11 | 0 | 0 |
| graphics_FontMetrics | 8 | 0 | 0 |
| graphics_GC | 71 | 2 | 1 |
| graphics_Image | 34 | 1 | 3 |
| graphics_ImageData | 19 | 2 | 0 |
| graphics_ImageLoader | 8 | 0 | 0 |
| graphics_ImageLoaderEvent | 2 | 0 | 0 |
| graphics_PaletteData | 5 | 0 | 0 |
| graphics_Path | 3 | 0 | 1 |
| graphics_Pattern | 16 | 0 | 0 |
| graphics_Point | 5 | 0 | 0 |
| graphics_RGB | 6 | 0 | 0 |
| graphics_RGBA | 6 | 0 | 0 |
| graphics_Rectangle | 17 | 0 | 0 |
| graphics_Region | 21 | 0 | 0 |
| graphics_TextLayout | 19 | 1 | 0 |
| graphics_Transform | 4 | 0 | 0 |
| layout_FormAttachment | 7 | 0 | 0 |
| layout_GridData | 3 | 0 | 0 |
| widgets_Button | 100 | 0 | 2 |
| widgets_Canvas | 96 | 0 | 0 |
| widgets_Caret | 18 | 0 | 0 |
| widgets_Combo | 141 | 0 | 0 |
| widgets_Composite | 91 | 0 | 0 |
| widgets_CoolBar | 96 | 0 | 0 |
| widgets_CoolItem | 20 | 4 | 0 |
| widgets_DateTime_Style_CALENDAR | 106 | 0 | 0 |
| widgets_DateTime_Style_DATE | 106 | 0 | 0 |
| widgets_DateTime_Style_TIME | 106 | 0 | 0 |
| widgets_DirectoryDialog | 7 | 0 | 0 |
| widgets_Display | 62 | 1 | 1 |
| widgets_ExpandBar | 104 | 0 | 0 |
| widgets_ExpandItem | 18 | 0 | 0 |
| widgets_FileDialog | 11 | 0 | 0 |
| widgets_Group | 94 | 0 | 0 |
| widgets_Label | 86 | 0 | 0 |
| widgets_Link | 86 | 0 | 0 |
| widgets_Menu | 29 | 0 | 0 |
| widgets_ProgressBar | 84 | 0 | 0 |
| widgets_Scale | 87 | 0 | 0 |
| widgets_ScrolledComposite | 101 | 0 | 0 |
| widgets_Shell | 139 | 0 | 1 |
| widgets_Slider | 98 | 0 | 0 |
| widgets_Spinner | 100 | 0 | 0 |
| widgets_TabFolder | 107 | 0 | 0 |
| widgets_Table | 136 | 0 | 0 |
| widgets_Text | 132 | 1 | 0 |
| widgets_ToolBar | 101 | 0 | 0 |
| widgets_ToolItem | 14 | 0 | 0 |
| widgets_Tree | 132 | 0 | 0 |

Skips: the tests' own `assumeTrue/assumeFalse`/`@Disabled*` for cocoa, plus `Display.test_setCursorLocation*`
(`@TempDir`-style parameter injection, not supported by the runner).

## Remaining failures

Round 17 (13, reasons in `tests/expected.txt`): CoolItem size/bounds (4), CTabFolder private-member reflection (4), `stream()` (1), pixel colour (1), a SIGSEGV closing a freed NSWindow after a zoom change (1), CCombo null text and a missing IllegalArgumentException (2).

Round 15:

Each is in `tests/expected.txt` with its reason: HiDPI/SVG loading (GC `withTransform` x2, Image `reevaluatesSizability..`),
private-method reflection (ImageData `blit` x2), TextLayout `bug568740` (rendering), Display `setSynchronizer` (local
class), Text `backspaceAndDelete` (`Display.post` of a key event returns false). Sections below describe earlier rounds.

## Null String arguments in tests

A test's literal `null` String becomes `jrt.NullString` only where the callee guards that parameter
(`NullArgGuards`, the same check as `NumericEmitter.stringParamNullCheck`); elsewhere it is `""`, the port's
contract. Before that, the sentinel leaked into `setToolTipText(null)` and `getFontList(null, ..)` (13 tests).

## Graphics failures by cause (TSK-051, before the merge)


| Cause | Count |
|---|---:|
| HiDPI image path in `swt` (TSK-048) | 18 |
| private-method reflection, `WeakReference`, cross-package anonymous subclass | 3 |
| rendering: exact colors in a drawn TextLayout, not diagnosed | 1 |

### HiDPI image path (18, TSK-2026-09-23-048)

The tests now get past their JDK needs (`Path`/`Files`/`@TempDir`, `Optional` aside) and stop in the HiDPI loading code:

- `Optional`/`Stream` in `Image`'s provider path (`unresolved call empty`/`of` in `swt/graphics_image.go`): GC
  `drawImage..IIII`, `..IIII_ImageDataProvider`, `..IIII_ImageDataAtSizeProvider[1..4]`.
- `ImageDataLoaderLoadByZoom`/`LoadBySize` and `DPIUtil.validateAndGetImagePathAtZoom` stubs (`stub: ImageLoader not
  ported`, `graphics_stubs_manual.go`): GC `drawImage..IIII_withTransform`, `.._zeroTargetSize`; Image
  `DeviceImageI_inputStream`, `DeviceLjava_io_InputStream`, `DeviceLjava_lang_String`, `Device_ImageFileNameProvider`,
  `drawImageAtSize_doesNotReReadFileOfNonSizableFormat`, `drawImageAtSize_reevaluatesSizabilityWhenFileNameChanges`;
  ImageData `getTransparencyMask`, `getTransparencyType`.
- Image `Device_ImageDataProvider`: `ERROR_INVALID_IMAGE` expected, `ERROR_UNSUPPORTED_FORMAT` got, from the provider's
  `loadByZoom` on a corrupt file (the direct `ImageData(String)` path is fixed, see below). Not checked that it is
  only this.
- `DPIUtil.ElementAtZoom`: ImageLoader `loadSingleFrameGifReportedAsAnimation_bug3404`.

### Not translated / not portable (3)

- ImageData `blit`, `blit_MsbLsb`: `ImageDataTestHelper` calls the private `ImageData.getByteOrder()` and the private static
  `blit(...)` through `Class.getDeclaredMethod`. The Go methods are unexported and the reflection registry (`swtreflect`)
  holds exported API only; no translator rule for it.
- GC `noMemoryLeakAfterDispose`: `WeakReference` + `System.gc()` (Go has `weak.Pointer`, but a `GC` here stays reachable
  through the cascade's `impl` and Display's registries; not tried).
- GC `drawImage_nonAutoScalableGC_bug_2504`: `new Canvas(...) { isAutoScalable() }` subclasses a class of another Go
  package (unsupported anonymous class); the test is a HiDPI scaling check anyway.

### Rendering, not diagnosed (1)

TextLayout `bug568740_multilineTextStyle` (line 825): an image drawn through an `ImageGcDrawer` with `setAntialias(OFF)`
has no pure `COLOR_BLUE` pixel in the first line's border range (the test needs an exact RGB match). The image holds ~730
distinct colors and the border/strikeout/underline pixels are blended or shifted (`[26 26 255]`, `[225 13 13]`, ...).
Zoom is 100. Hypotheses, not verified: the bitmap context ignores `setShouldAntialias(false)`, or a color-space
conversion between `NSColor.colorWithDeviceRed` and the image's bitmap rep.

## Fixed in TSK-051 (33 tests: 55 -> 22 failures, counted by primary cause)

| Cause | Tests | Fix |
|---|---:|---|
| `java.nio.file.Path/Files` + `@TempDir` | 8 Image (`getBoundsInPixels`, `getImageDataCurrentZoom`, `getImageData_100/125/150/200`, `..changingImageDataDoesNotAffectImage`, `equals`) | `jrt.Path/Files`, `TestEmitter` creates the directory |
| `Device_ImageDataProvider` | 1 | the above + `errorUndecodable` (`ERROR_INVALID_IMAGE` for a corrupt file with a known signature) |
| lambdas for `java.util.function.*`/`Comparator`, `Object.hashCode` | 3 Image (`fileNameProvider`, `bug566545_efficientGrayscaleImage`, `hashCode`) | `GoTypes.isJdkFunctional`, `JdkCalls` |
| `StringBuilder`, `Float.parseFloat`, `Locale`, `Thread`/`AtomicReference` | 4 (FontData `toString`, `ConstructorString`, `setLocale`; GC `bug1288`) | `jrt.StringBuilder/Locale/Thread/AtomicReference` |
| null String argument | 6 (Font x2, FontData `StringII`/`setName`, ImageLoader `load(String)`/`save(String)`) | `jrt.NullString` sentinel, guards check it |
| `TextStyle.equals(null)`: typed nil in `any` | 1 (TextLayout `setStyle`) | instanceof helpers reject a typed nil |
| `NSIntersectionRect`/`CGDisplayBounds`/`PtInRgn` by-value natives | 4 (GC `setClipping` x2, Region `contains` x2) | `internal/cocoa/rect_manual.go` |
| `a[i] = f(i++)` order + `String.length()` in bytes | 1 (TextLayout `getSegments`) | `pinIndex`, `jrt.StringLength/Substring` |
| stubs | 5 (ImageData `ConstructorInputStream`/`ConstructorString`, ImageLoader `addImageLoaderListener`/`save(OutputStream)`, ImageLoaderEvent) | `swt/graphics_imagecodec_manual.go`, `jrt.NewEventObject`, `Arrays.asList` |

Beyond the tests: the by-value natives also fed `Canvas.scroll`, Tree/Table expansion frames and `GC.copyArea`; `String.length()`
was the byte length, so every non-ASCII string handed to `NSString.stringWith` was padded with NULs (20 call sites in `swt`).

## Widget failures by cause (TSK-052, before the merge; 66 of all listed failures pass now)

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
prints how many and `make test-swt-update` records them. Refresh after `make gen`: see README "Round 14 widget
tests".
