# SWT JUnit tests (TSK-2026-09-23-051)

`make test-swt` on macOS arm64, SWT `af630a9093`, translated set: the `graphics`, `layout` and
`events` test classes plus `SwtTestUtil`, `ImageTestUtil`, `CapturedOutput`,
`tests/graphics/ImageDataTestHelper`. Widget test classes are not translated (task 052).

**Total: 352 tests - 325 passed, 22 failed, 5 skipped** (first run, TSK-2026-09-23-051 seed: 292 / 55 / 5).

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
| graphics_Font | 9 | 0 | 0 |
| graphics_FontData | 11 | 0 | 0 |
| graphics_FontMetrics | 8 | 0 | 0 |
| graphics_GC | 63 | 10 | 1 |
| graphics_Image | 29 | 6 | 3 |
| graphics_ImageData | 17 | 4 | 0 |
| graphics_ImageLoader | 7 | 1 | 0 |
| graphics_ImageLoaderEvent | 2 | 0 | 0 |
| graphics_PaletteData | 5 | 0 | 0 |
| graphics_Path | 3 | 0 | 1 |
| graphics_Pattern | 16 | 0 | 0 |
| graphics_Point | 5 | 0 | 0 |
| graphics_Rectangle | 17 | 0 | 0 |
| graphics_Region | 21 | 0 | 0 |
| graphics_RGB | 6 | 0 | 0 |
| graphics_RGBA | 6 | 0 | 0 |
| graphics_TextLayout | 19 | 1 | 0 |
| graphics_Transform | 4 | 0 | 0 |
| layout_FormAttachment | 7 | 0 | 0 |
| layout_GridData | 3 | 0 | 0 |

Skips are the tests' own `assumeTrue/assumeFalse` for cocoa (GC `bug493455`, Image
`imageDataIsCached`/`imageDataSameVia*` x2, Path `testClonePath`).

## Failures by cause (22)

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
| `NSIntersectionRect`/`CGDisplayBounds`/`PtInRgn` by-value natives | 4 (GC `setClipping` x2, Region `contains` x2) | `internal/cocoa/os_custom_manual.go` |
| `a[i] = f(i++)` order + `String.length()` in bytes | 1 (TextLayout `getSegments`) | `pinIndex`, `jrt.StringLength/Substring` |
| stubs | 5 (ImageData `ConstructorInputStream`/`ConstructorString`, ImageLoader `addImageLoaderListener`/`save(OutputStream)`, ImageLoaderEvent) | `swt/graphics_imagecodec_manual.go`, `jrt.NewEventObject`, `Arrays.asList` |

Beyond the tests: the by-value natives also fed `Canvas.scroll`, Tree/Table expansion frames and `GC.copyArea`; `String.length()`
was the byte length, so every non-ASCII string handed to `NSString.stringWith` was padded with NULs (20 call sites in `swt`).
