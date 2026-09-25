# SWT JUnit tests: first run (TSK-2026-09-23-051 seed)

`make test-swt` on macOS arm64, SWT `af630a9093`, translated set: the `graphics`, `layout` and
`events` test classes plus `SwtTestUtil`, `ImageTestUtil`, `CapturedOutput`,
`tests/graphics/ImageDataTestHelper`. Widget test classes are not translated (task 052).

**Total: 352 tests - 292 passed, 55 failed, 5 skipped.**

Not translated from this set: `Test_org_eclipse_swt_layout_BorderLayout` - `BorderLayout`/`BorderData`
are not in `swt`, and its `MockControl extends Canvas` subclasses a widget from another Go package
(the impl cascade's `impl` field and `init<X>` are unexported).

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
| graphics_GC | 60 | 13 | 1 |
| graphics_Image | 17 | 18 | 3 |
| graphics_ImageData | 15 | 6 | 0 |
| graphics_ImageLoader | 3 | 5 | 0 |
| graphics_ImageLoaderEvent | 1 | 1 | 0 |
| graphics_PaletteData | 5 | 0 | 0 |
| graphics_Path | 3 | 0 | 1 |
| graphics_Pattern | 16 | 0 | 0 |
| graphics_Point | 5 | 0 | 0 |
| graphics_Rectangle | 17 | 0 | 0 |
| graphics_Region | 19 | 2 | 0 |
| graphics_RGB | 6 | 0 | 0 |
| graphics_RGBA | 6 | 0 | 0 |
| graphics_TextLayout | 17 | 3 | 0 |
| graphics_Transform | 4 | 0 | 0 |
| layout_FormAttachment | 7 | 0 | 0 |
| layout_GridData | 3 | 0 | 0 |

Skips are the tests' own `assumeTrue/assumeFalse` for cocoa (GC `bug493455`, Image
`imageDataIsCached`/`imageDataSameVia*` x2, Path `testClonePath`).

## Failures by cause

| Cause | Count |
|---|---:|
| not translated: JDK surface | 23 |
| not translated: HiDPI image path in `swt` (TSK-048) | 12 |
| translator contract: null String is `""` | 8 |
| stub (hand-written codec / jrt) | 4 |
| translator bug | 3 |
| port bug, not yet diagnosed | 5 |

### not translated: JDK surface (23)

- `java.nio.file.Path/Files` + `@TempDir` (`SwtTestUtil.getPath` -> `unresolved call resolve`): Image
  x15 (`fileNameProvider`, `inputStream`, `Device_ImageDataProvider`, `Device_ImageFileNameProvider`,
  `drawImageAtSize_*` x2, `equals`, `getBoundsInPixels`, `getImageDataCurrentZoom`, `getImageData_100/125/150/200`,
  `changingImageDataDoesNotAffectImage`, `hashCode`).
- `StringBuilder` in swt's `FontData.toString`/`FontData(String)`: FontData `toString`, `ConstructorLjava_lang_String`.
- `java.util.Random` in `ImageDataTestHelper`: ImageData `blit`, `blit_MsbLsb`.
- `Locale.ENGLISH`: FontData `setLocale`. `AtomicReference`/`Thread`: GC `bug1288_createGCFromImageFromNonDisplayThread`.
  `WeakReference`/`System.gc`: GC `noMemoryLeakAfterDispose`. `BiFunction` lambda in `ImageTestUtil`: Image
  `bug566545_efficientGrayscaleImage`.

### not translated: HiDPI image path in `swt` (12, TSK-2026-09-23-048)

`Optional`/`Stream` in `Image`'s provider path (`unresolved call empty`/`of` in `swt/graphics_image.go`):
GC `drawImage..IIII`, `..IIII_ImageDataProvider`, `..IIII_ImageDataAtSizeProvider[1..4]`. `ImageDataLoaderLoadByZoom`
stub (`stub: ImageLoader not ported`): GC `drawImage..IIII_withTransform`, `.._zeroTargetSize`, Image
`DeviceLjava_io_InputStream`, ImageData `getTransparencyMask`, `getTransparencyType`. `DPIUtil.ElementAtZoom`:
ImageLoader `loadSingleFrameGifReportedAsAnimation_bug3404`.

### translator contract: null String is `""` (8)

The tests pass `null` for a String and expect `ERROR_NULL_ARGUMENT`; the port has no null String
(README "Round 11 null-string"), the guard is `if false`: Font `DeviceLjava_lang_StringII`,
`DeviceLorg_eclipse_swt_graphics_FontData`; FontData `StringII`, `setName`; Image `DeviceLjava_lang_String`
("Argument not valid" instead of "Argument cannot be null"); ImageData `ConstructorLjava_lang_String`,
ImageLoader `load(String)`, `save(String)` (`IOException` from opening "" instead of
`IllegalArgumentException`).

### stub (4)

- `ImageDataLoaderLoad(nil)` panics a string instead of `ERROR_NULL_ARGUMENT`: ImageData `ConstructorLjava_io_InputStream`.
- The stdlib codec never fires `ImageLoaderListener` (known, README "Round 9 images"): ImageLoader
  `addImageLoaderListener`.
- `save` to an unsupported format raises `IllegalArgumentException`, SWT `SWTException`: ImageLoader `saveLjava_io_OutputStreamI`.
- `jrt.NewEventObject` has no null-source check: ImageLoaderEvent `Constructor...`.

### translator bug (3)

- `object == null` on a Java `Object` (Go `any`) holding a typed nil pointer is false in Go:
  `TextStyle.equals(null style)` dereferences nil (`swt/graphics_textstyle.go:118`): TextLayout `setStyle`,
  `bug568740_multilineTextStyle`.
- Anonymous subclass of another package's cascade class (`new Canvas(shell, 0) {...}` in a test) is a
  marker: GC `drawImage_nonAutoScalableGC_bug_2504`.

### port bug, not yet diagnosed (5)

- GC `setClippingIIII`, `setClippingLorg_eclipse_swt_graphics_Rectangle`: `getClipping()` after
  `setClipping(5,10,20,...)` returns `0,200,200` (the full image).
- Region `containsII`, `containsLorg_eclipse_swt_graphics_Point`: a region with one rectangle does not
  contain a point inside it.
- TextLayout `getSegments`: `index out of range [17] with length 17` in `swt/graphics_textlayout.go`.
