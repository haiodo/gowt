package dev.gowt.j2go;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

/** Names for out-of-translated-set members hand-written in *_manual.go (see manual.txt). */
public class Manual {

	private static final String ROUNDING_MODE = "org.eclipse.swt.graphics.RoundingMode";
	private static final String PLATFORM = "org.eclipse.swt.internal.Platform";
	private static final String LIBRARY = "org.eclipse.swt.internal.Library";
	private static final String TOUCH = "org.eclipse.swt.widgets.Touch";
	private static final String EXCEPTION_STASH = "org.eclipse.swt.internal.ExceptionStash";
	private static final String CALLBACK = "org.eclipse.swt.internal.Callback";
	private static final String IME = "org.eclipse.swt.widgets.IME";
	private static final String WIDGET_SPY = "org.eclipse.swt.internal.WidgetSpy";
	private static final String AUTOSCALING_MODE = "org.eclipse.swt.graphics.AutoscalingMode";
	// checkWidget()'s thread-affinity check: no real Thread, both sides collapse to "any" nil.
	private static final String JAVA_THREAD = "java.lang.Thread";
	// Boxed Integer stays "any" (valueOf/intValue are JdkIntrinsics); Boolean unboxes to bool.
	private static final String JAVA_INTEGER = "java.lang.Integer";
	private static final String JAVA_BOOLEAN = "java.lang.Boolean";
	private static final String JAVA_CLASS = "java.lang.Class";
	private static final String JAVA_RUNNABLE = "java.lang.Runnable";
	private static final String SWT_LONG = "org.eclipse.swt.internal.LONG";
	private static final String DISPLAY_APPEARANCE = "org.eclipse.swt.widgets.Display.APPEARANCE";
	private static final String COCOA_PKG = "org.eclipse.swt.internal.cocoa.";
	public static final String GDK_RECTANGLE_TO_RECTANGLE = "GdkRectangleToRectangle";
	private static final String BROWSER_TEST = "org.eclipse.swt.tests.junit.Test_org_eclipse_swt_browser_Browser";
	private static final String BROWSER_TEST_GO = "Test_org_eclipse_swt_browser_Browser";
	private static final String CONTROL_EXAMPLE_PKG = "org.eclipse.swt.examples.controlexample.";

	// java.lang/java.util base types some translated classes extend (SWTException/SWTError,
	// TypedEvent - see README "Manual superclass embedding") or catch. Hand-written in internal/jrt.
	public static final String JAVA_THROWABLE = "java.lang.Throwable";
	public static final String JAVA_RUNTIME_EXCEPTION = "java.lang.RuntimeException";
	public static final String JAVA_ERROR = "java.lang.Error";
	public static final String JAVA_EXCEPTION = "java.lang.Exception";
	private static final String JAVA_ILLEGAL_ARGUMENT = "java.lang.IllegalArgumentException";
	private static final String JAVA_EVENT_OBJECT = "java.util.EventObject";
	private static final String JAVA_EVENT_LISTENER = "java.util.EventListener";
	private static final String SWT_EVENT_LISTENER = "org.eclipse.swt.internal.SWTEventListener";
	public static final String JRT_IMPORT = "github.com/haiodo/gowt/internal/jrt";

	// Round 9 images: java.io surface ImageLoader.java's own translated code needs (field/param
	// types and new FileInputStream/FileOutputStream(filename)) - README "Round 9 images". The
	// actual PNG/GIF/BMP/JPEG codec backend is a hand-written stdlib wrapper, not translated.
	private static final String JAVA_IO_INPUT_STREAM = "java.io.InputStream";
	private static final String JAVA_IO_OUTPUT_STREAM = "java.io.OutputStream";
	private static final String JAVA_IO_EXCEPTION = "java.io.IOException";
	private static final String JAVA_IO_FILE_INPUT_STREAM = "java.io.FileInputStream";
	// Only ever appears inside Image's own (already-panicking, HiDPI @2x) "new
	// BufferedInputStream(...)" - aliased to InputStream so that dead-code var declaration still
	// type-checks, not because a real BufferedInputStream (extra unread()-pushback) is ported.
	private static final String JAVA_IO_BUFFERED_INPUT_STREAM = "java.io.BufferedInputStream";
	private static final String JAVA_IO_FILE_OUTPUT_STREAM = "java.io.FileOutputStream";

	private record Entry(String goType, String importPath, boolean isValueType) {}

	private static final Map<String, Entry> ENTRIES = new LinkedHashMap<>();

	private static void reg(String qualified, String goType, String importPath, boolean isValueType) {
		ENTRIES.put(qualified, new Entry(goType, importPath, isValueType));
	}

	static {
		reg(ROUNDING_MODE, "RoundingMode", null, true);
		reg(PLATFORM, "Platform", null, false);
		reg(LIBRARY, "Library", null, false);
		reg(TOUCH, "Touch", null, false);
		reg(EXCEPTION_STASH, "ExceptionStash", null, false);
		reg(CALLBACK, "Callback", null, false);
		reg(IME, "IME", null, false);
		reg(WIDGET_SPY, "WidgetSpy", null, false);
		reg(AUTOSCALING_MODE, "AutoscalingMode", null, true);
		reg(JAVA_THREAD, "any", null, true);
		reg(JAVA_INTEGER, "any", null, true);
		reg(JAVA_BOOLEAN, "bool", null, true);
		reg(JAVA_CLASS, "reflect.Type", "reflect", true);
		reg(JAVA_RUNNABLE, "jrt.Runnable", JRT_IMPORT, true);
		// java.util containers: hand-written in internal/jrt (util.go), erased to any elements.
		for (String q : new String[]{"java.util.Map", "java.util.HashMap", "java.util.concurrent.ConcurrentHashMap"}) reg(q, "jrt.Map", JRT_IMPORT, false);
		reg("java.util.Map.Entry", "jrt.MapEntry", JRT_IMPORT, false);
		for (String q : new String[]{"java.util.List", "java.util.ArrayList", "java.util.Set", "java.util.HashSet", "java.util.concurrent.ConcurrentLinkedQueue", "java.util.LinkedList", "java.util.LinkedHashSet", "java.util.AbstractCollection", "java.util.Collection", "java.util.stream.Stream"}) {
			reg(q, "jrt.List", JRT_IMPORT, false);
		}
		reg("java.util.Iterator", "jrt.Iterator", JRT_IMPORT, false);
		reg("java.util.TreeSet", "jrt.TreeSet", JRT_IMPORT, false);
		reg("java.util.stream.Collector", "jrt.Collector", JRT_IMPORT, false);
		reg(SWT_LONG, "LONG", null, false);
		// org.eclipse.swt.internal helpers referencing swt types (so not translatable into cocoa):
		// only their static members are used, hand-written in swt/internal_helpers_manual.go.
		for (String n : new String[]{"DPIUtil", "BidiUtil", "Compatibility", "DefaultExceptionHandler"}) {
			String q = "org.eclipse.swt.internal." + n;
			reg(q, q.substring(q.lastIndexOf('.') + 1), null, false);
		}
		// GC's text-layout cache: a record key and an LRU LinkedHashMap subclass, hand-written in
		// swt/graphics_gc_manual.go (no record rule; removeEldestEntry has no jrt.Map equivalent).
		reg("org.eclipse.swt.graphics.GC.GCTextData.Key", "GC_GCTextData_Key", null, false);
		// Browser: the Edge-unavailable dialog belongs to Browser.createWebBrowser, which is hand-written.
		reg("org.eclipse.swt.browser.Browser.WebViewUnavailableDialog", "BrowserWebViewUnavailableDialog", null, false);
		reg("org.eclipse.swt.graphics.GC.GCTextData.Cache", "GC_GCTextData_Cache", null, false);
		// Image loading (ImageLoader + codecs) is not ported: loaders/strict checks/disabled-image
		// colour transform are stubs in swt/graphics_stubs_manual.go.
		for (String n : new String[]{"graphics.ImageDataLoader", "internal.image.FileFormat", "internal.image.ImageColorTransformer",
				"internal.StrictChecks"}) {
			String q = "org.eclipse.swt." + n;
			reg(q, q.substring(q.lastIndexOf('.') + 1), null, false);
		}
		// A nested Java enum: no enum rule yet, hand-written as an int32 type + constants.
		reg(DISPLAY_APPEARANCE, "Display_APPEARANCE", null, true);
		// java.lang.Throwable itself is only ever used as a field/param/return TYPE (never a
		// manual superclass to embed) - Go's builtin error interface is exactly that role.
		reg(JAVA_THROWABLE, "error", null, true);
		reg(JAVA_RUNTIME_EXCEPTION, "jrt.RuntimeException", JRT_IMPORT, false);
		reg(JAVA_ERROR, "jrt.JavaError", JRT_IMPORT, false);
		reg(JAVA_ILLEGAL_ARGUMENT, "jrt.IllegalArgumentException", JRT_IMPORT, false);
		reg(JAVA_EVENT_OBJECT, "jrt.EventObject", JRT_IMPORT, false);
		// Both content-free marker interfaces in Java; nothing to embed on the Go side.
		reg(JAVA_EVENT_LISTENER, "any", null, true);
		reg(SWT_EVENT_LISTENER, "any", null, true);
		// isValueType=true: a bare Go interface (jrt.InputStream/OutputStream), never "*jrt.X" -
		// same reasoning as java.lang.Throwable -> error above.
		reg(JAVA_IO_INPUT_STREAM, "jrt.InputStream", JRT_IMPORT, true);
		reg(JAVA_IO_OUTPUT_STREAM, "jrt.OutputStream", JRT_IMPORT, true);
		reg(JAVA_IO_EXCEPTION, "jrt.IOException", JRT_IMPORT, false);
		// Names only feed ctorFuncName ("jrt.NewFileInputStream"/"...Output..."); no Go type by
		// these names actually exists, the constructor returns jrt.InputStream/OutputStream directly.
		reg(JAVA_IO_FILE_INPUT_STREAM, "jrt.FileInputStream", JRT_IMPORT, false);
		reg(JAVA_IO_FILE_OUTPUT_STREAM, "jrt.FileOutputStream", JRT_IMPORT, false);
		reg(JAVA_IO_BUFFERED_INPUT_STREAM, "jrt.InputStream", JRT_IMPORT, true);
		// Round 12: the in-memory streams the ImageLoader tests save to and load from.
		reg("java.io.ByteArrayInputStream", "jrt.ByteArrayInputStream", JRT_IMPORT, false);
		reg("java.io.ByteArrayOutputStream", "jrt.ByteArrayOutputStream", JRT_IMPORT, false);
		// Round 14: java.util.concurrent.atomic cells the widget tests capture from listeners.
		for (String n : new String[]{"AtomicBoolean", "AtomicInteger", "AtomicLong", "AtomicReference", "AtomicIntegerArray", "AtomicReferenceArray"}) {
			reg("java.util.concurrent.atomic." + n, "jrt." + n, JRT_IMPORT, false);
		}
		reg("java.net.URI", "jrt.URI", JRT_IMPORT, false);
		// EchoHttpServer (internal/jrt/http.go).
		for (String q : new String[]{"com.sun.net.httpserver.HttpExchange", "com.sun.net.httpserver.HttpServer", "com.sun.net.httpserver.Headers",
				"java.net.InetSocketAddress", "java.net.InetAddress", "java.net.URLEncoder", "java.net.URLDecoder", "java.nio.charset.Charset",
				"java.nio.charset.StandardCharsets"}) {
			String go = q.endsWith("Headers") ? "HttpHeaders" : q.endsWith("StandardCharsets") ? "StandardCharsets" : q.substring(q.lastIndexOf('.') + 1);
			reg(q, "jrt." + go, JRT_IMPORT, false);
		}
		reg("java.time.Instant", "jrt.Instant", JRT_IMPORT, false);
		reg("java.time.Duration", "jrt.Duration", JRT_IMPORT, false);
		// A record (no rule): swt/graphics_stubs_manual.go, element erased to any.
		reg("org.eclipse.swt.internal.DPIUtil.ElementAtZoom", "DPIUtilElementAtZoom", null, false);
		// TestInfo: internal/junit; Optional: internal/jrt/jdk.go.
		reg("org.junit.jupiter.api.TestInfo", "junit.TestInfo", "github.com/haiodo/gowt/internal/junit", false);
		reg("java.util.Optional", "jrt.Optional", JRT_IMPORT, false);
		reg("java.util.OptionalInt", "jrt.Optional", JRT_IMPORT, false);
		// internal/jrt/concurrent.go; TimeUnit is a value type (jrt.TimeUnitMILLISECONDS, ...).
		for (String q : new String[]{"java.util.concurrent.CountDownLatch", "java.util.concurrent.CompletableFuture",
				"java.lang.ref.WeakReference"}) {
			reg(q, "jrt." + q.substring(q.lastIndexOf('.') + 1), JRT_IMPORT, false);
		}
		// gtk DateTime: Calendar/DateFormat over time (internal/jrt/datetime.go); Field, Format.Field and Attribute share one Go type.
		for (String q : new String[]{"java.util.Calendar", "java.util.Date", "java.text.FieldPosition", "java.text.ParseException",
				"java.text.AttributedCharacterIterator", "java.text.CharacterIterator", "java.text.DateFormatSymbols", "java.util.Collections"}) {
			reg(q, "jrt." + q.substring(q.lastIndexOf('.') + 1), JRT_IMPORT, false);
		}
		for (String q : new String[]{"java.text.DateFormat", "java.text.SimpleDateFormat"}) reg(q, "jrt.DateFormat", JRT_IMPORT, false);
		for (String q : new String[]{"java.text.DateFormat.Field", "java.text.Format.Field", "java.text.AttributedCharacterIterator.Attribute"}) {
			reg(q, "jrt.DateFormatField", JRT_IMPORT, false);
		}
		// jface: the org.eclipse.core.runtime subset in internal/jrt/core.go (interfaces are bare Go interfaces).
		for (String n : new String[]{"Assert", "Status", "CoreException", "NullProgressMonitor", "ProgressMonitorWrapper", "SafeRunner", "ListenerList"}) {
			reg("org.eclipse.core.runtime." + n, "jrt." + n, JRT_IMPORT, false);
		}
		for (String n : new String[]{"IStatus", "IProgressMonitor", "ISafeRunnable"}) reg("org.eclipse.core.runtime." + n, "jrt." + n, JRT_IMPORT, true);
		reg("java.lang.NoSuchMethodException", "jrt.NoSuchMethodException", JRT_IMPORT, false);
		// jface/dialogs_manual.go: the statics layout needs of these slice-B classes.
		reg("org.eclipse.jface.dialogs.Dialog", "Dialog", null, false);
		reg("org.eclipse.jface.resource.JFaceResources", "JFaceResources", null, false);
		reg("java.util.concurrent.TimeUnit", "jrt.TimeUnit", JRT_IMPORT, true);
		reg("java.util.Properties", "jrt.Map", JRT_IMPORT, false);
		// Device (gtk) reads its CSS through these; one Go Reader serves all three reader classes (jrt/jdkio.go).
		for (String n : new String[]{"File", "BufferedReader", "InputStreamReader", "FileReader"}) reg("java.io." + n, "jrt." + n, JRT_IMPORT, false);
		reg("java.util.regex.Pattern", "jrt.Pattern", JRT_IMPORT, false);
		reg("java.util.regex.Matcher", "jrt.Matcher", JRT_IMPORT, false);
		reg("java.util.StringTokenizer", "jrt.StringTokenizer", JRT_IMPORT, false);
		// java.util.ResourceBundle over the registered resource FS, java.text.MessageFormat's
		// {n} substitution, and the exceptions they (and Integer.parseInt) throw - internal/jrt/text.go.
		reg("java.util.ResourceBundle", "jrt.ResourceBundle", JRT_IMPORT, false);
		reg("java.util.MissingResourceException", "jrt.MissingResourceException", JRT_IMPORT, false);
		reg("java.text.MessageFormat", "jrt.MessageFormat", JRT_IMPORT, false);
		reg("java.lang.NumberFormatException", "jrt.NumberFormatException", JRT_IMPORT, false);
		// Round 13: JDK types the swt/tests sources construct or pass around (internal/jrt/jdk.go, nio.go).
		for (String q : new String[]{"java.lang.StringBuilder", "java.util.Random", "java.util.Locale",
				"java.nio.file.Path", "java.nio.file.Files"}) {
			reg(q, "jrt." + q.substring(q.lastIndexOf('.') + 1), JRT_IMPORT, false);
		}
	}

	// Methods hand-written in *_manual.go instead of translated (key: Names.erasureKey).
	private static final Map<String, String> MANUAL_METHODS = Map.ofEntries(
			// id.objc_getClass()/toString(): need reflect over this.impl (see id_manual.go).
			Map.entry(COCOA_PKG + "id#objc_getClass()", "ObjcGetClass"),
			Map.entry(COCOA_PKG + "id#toString()", "String"),
			// JNI global refs: a Go handle table instead (internal/cocoa/jni_manual.go).
			Map.entry(COCOA_PKG + "OS#NewGlobalRef(java.lang.Object)", "OSNewGlobalRef"),
			Map.entry(COCOA_PKG + "OS#DeleteGlobalRef(J)", "OSDeleteGlobalRef"),
			Map.entry(COCOA_PKG + "OS#JNIGetObject(J)", "OSJNIGetObject"),
			// os.c wraps every native in @try/@catch and returns 0 on an NSException; Go can't
			// catch one, so the few selectors SWT relies on that for avoid the throw instead
			// (internal/cocoa/nsexception_manual.go).
			Map.entry(COCOA_PKG + "NSColor#colorSpace()", "ColorSpace"),
			// Creates the webview-backed WebBrowser (browser/browser_manual.go).
			Map.entry("org.eclipse.swt.browser.Browser#createWebBrowser(org.eclipse.swt.widgets.Composite,I)", "CreateWebBrowser"),
			// Test_org_eclipse_swt_browser_Browser: JVM diagnostics and the Linux fd listing, and the one test that needs a null Boolean
			// plus a page-level JavaScript switch (tests/swttests/browser_manual.go).
			Map.entry(BROWSER_TEST + "#printSystemEnv()", BROWSER_TEST_GO + "PrintSystemEnv"),
			Map.entry(BROWSER_TEST + "#printMemoryUse()", BROWSER_TEST_GO + "PrintMemoryUse"),
			Map.entry(BROWSER_TEST + "#printThreadsInfo()", BROWSER_TEST_GO + "PrintThreadsInfo"),
			Map.entry(BROWSER_TEST + "#getOpenedDescriptors()", BROWSER_TEST_GO + "GetOpenedDescriptors"),
			Map.entry(BROWSER_TEST + "#getPropertiesSafe()", BROWSER_TEST_GO + "GetPropertiesSafe"),
			Map.entry(BROWSER_TEST + "#test_setJavascriptEnabled()", "Test_setJavascriptEnabled"),
			// Draws through a bitmap NSGraphicsContext, not the deprecated NSImage.lockFocus (swt/widgets_taskitem_manual_darwin.go).
			Map.entry("org.eclipse.swt.widgets.TaskItem#updateImage()", "UpdateImage"),
			Map.entry("org.eclipse.swt.accessibility.Accessible#setNumberVARIANT(J,java.lang.Number)", "SetNumberVARIANT"),
			Map.entry("org.eclipse.swt.accessibility.Accessible#toString()", "String"),
			// os.c's by-pointer wrappers over functions that take and return an NSRect by value (internal/cocoa/rect_manual.go).
			Map.entry(COCOA_PKG + "OS#NSIntersectionRect(" + COCOA_PKG + "NSRect," + COCOA_PKG + "NSRect," + COCOA_PKG + "NSRect)",
					"OSNSIntersectionRect"),
			Map.entry(COCOA_PKG + "OS#CGDisplayBounds(I," + COCOA_PKG + "CGRect)", "OSCGDisplayBounds"),
			Map.entry(COCOA_PKG + "OS#PtInRgn([S,J)", "OSPtInRgn"),
			// An out-of-range index (-1: nothing selected) throws NSRangeException, which os.c swallows.
			Map.entry(COCOA_PKG + "NSComboBox#selectItemAtIndex(J)", "SelectItemAtIndex"),
			Map.entry(COCOA_PKG + "NSComboBox#itemObjectValueAtIndex(J)", "ItemObjectValueAtIndex"),
			Map.entry("org.eclipse.swt.internal.gtk.GdkRectangle#toRectangle()", GDK_RECTANGLE_TO_RECTANGLE),
			Map.entry("org.eclipse.swt.widgets.Display#dumpWidgetTableInfo()", "DumpWidgetTableInfoManual"),
			Map.entry("java.lang.Thread#sleep(J)", "jrt.Sleep"),
			Map.entry("java.lang.Thread#interrupted()", "jrt.Interrupted"),
			Map.entry("java.lang.Thread#yield()", "jrt.Yield"),
			// os.c's `((void (*)())proc)(id, sel)`: a call through a C function pointer.
			Map.entry(COCOA_PKG + "OS#call(J,J,J)", "OSCall"),
			// A fixed "return YES" IMP in os_custom.c, no Java/Go call (callback_manual.go).
			Map.entry(COCOA_PKG + "OS#isFlipped_CALLBACK()", "OSIsFlipped_CALLBACK"),
			// Compares a Java class name to "org.eclipse.swt.widgets." - a Go type's name can't
			// match that, so it checks the Go package instead (swt/widgets_stubs3_manual.go).
			Map.entry("org.eclipse.swt.widgets.Display#isValidClass(java.lang.Class)", "DisplayIsValidClass"),
			Map.entry("java.lang.Thread#currentThread()", "jrt.CurrentThread"),
			// Generic, so hand-written over func() any (swt/widgets_display_manual.go).
			Map.entry("org.eclipse.swt.widgets.Display#syncCall(org.eclipse.swt.SwtCallable)", "SyncCall"),
			Map.entry("java.lang.System#currentTimeMillis()", "jrt.CurrentTimeMillis"),
			// Eclipse's test harness screenshot helper: nothing to capture in cmd/swttest.
			Map.entry("org.eclipse.test.Screenshots#takeScreenshot(java.lang.Class,java.lang.String)", "jrt.TakeScreenshot"),
			// Round 9 images: loadByZoom's HiDPI @2x-variant dispatch (Stream/Optional/
			// ElementAtZoom<T>, no translator rule) is out of scope; NativeImageLoader.save's own
			// caller (ImageLoader.save) is plain, but NativeImageLoader itself was never translated
			// (cocoa PI-layer file) - both replaced by hand-written stubs/wrappers in
			// swt/graphics_imagecodec_manual.go, see README "Round 9 images".
			Map.entry("org.eclipse.swt.graphics.ImageLoader#loadByZoom(java.io.InputStream,I,I)", "LoadByZoomStub"),
			Map.entry("org.eclipse.swt.internal.NativeImageLoader#load(org.eclipse.swt.internal.DPIUtil$ElementAtZoom,org.eclipse.swt.graphics.ImageLoader,I)",
					"NativeImageLoaderLoad"),
			Map.entry("org.eclipse.swt.internal.NativeImageLoader#save(java.io.OutputStream,I,org.eclipse.swt.graphics.ImageLoader)",
					"NativeImageLoaderSave"),
			// Round 10: only the tabs translated so far (examples/controlexample/controlexample_manual.go).
			Map.entry(CONTROL_EXAMPLE_PKG + "ControlExample#createTabs()", "CreateTabs"));

	// Dropped: Selector.java's own bookkeeping is elided (see README). Matched by name only.
	private static final Set<String> SKIP_METHOD_NAMES = Set.of(
			// jface: BrowserFactory is not translated (swt Browser is not ported).
			"org.eclipse.jface.widgets.WidgetFactory#browser",
			COCOA_PKG + "OS#registerSelector", COCOA_PKG + "OS#getSelector",
			// Takes a Display and Colors: a PI package cannot reference swt (dark-theme tweaks, not ported).
			"org.eclipse.swt.internal.win32.OS#setTheme",
			// Exits the process on an old Windows build; a no-op in internal/win32/custom_manual.go.
			"org.eclipse.swt.internal.win32.version.OsVersion#checkCompatibleWindowsVersion");

	// Cocoa stands in for these by hand (swt/*_manual_darwin.go); win32 translates the real sources.
	private static final Set<String> WIN32_TRANSLATED = Set.of("org.eclipse.swt.internal.BidiUtil", IME);

	// Hand-written for cocoa only: gtk translates its own IME.
	private static final Set<String> COCOA_ONLY = Set.of(IME);

	public static boolean isManual(String qualifiedTypeName) {
		return ENTRIES.containsKey(qualifiedTypeName) && !(GoTypes.platform == Platform.WIN32 && WIN32_TRANSLATED.contains(qualifiedTypeName))
				&& !(COCOA_ONLY.contains(qualifiedTypeName) && !GoTypes.piPackage.equals("cocoa"));
	}

	/** A value type with no Go method surface mirroring Java's (a bare "any", or java.lang.Class
	 * as reflect.Type) - method calls on it can't dispatch through Manual.instanceMember and
	 * fall back to the unresolved-call degrade instead (see InvocationEmitter.emitMethodInvocation). */
	public static boolean isBareAny(String qualifiedTypeName) {
		Entry e = ENTRIES.get(qualifiedTypeName);
		return e != null && e.isValueType() && (e.goType().equals("any") || qualifiedTypeName.equals(JAVA_CLASS));
	}

	/** Go func name for a manual.txt method-level entry (erasureKey format), or null. */
	public static String manualMethod(String erasureKey) {
		return MANUAL_METHODS.get(erasureKey);
	}

	public static boolean isSkippedMethod(String declaringClassQualifiedName, String javaMethodName) {
		return SKIP_METHOD_NAMES.contains(declaringClassQualifiedName + "#" + javaMethodName);
	}

	/** OS.java's own SELECTORS bookkeeping field - see SKIP_METHOD_NAMES, same reasoning. */
	public static boolean isSkippedField(String declaringClassQualifiedName, String javaFieldName) {
		// jface: the button labels read JFaceResources' bundle (slice B); util.Util's empty sorted set needs Collections wrappers.
		if (declaringClassQualifiedName.equals("org.eclipse.jface.dialogs.IDialogConstants") && javaFieldName.endsWith("_LABEL")) return true;
		if (declaringClassQualifiedName.equals("org.eclipse.jface.util.Util") && javaFieldName.equals("EMPTY_SORTED_SET")) return true;
		return declaringClassQualifiedName.equals(COCOA_PKG + "OS") && javaFieldName.equals("SELECTORS");
	}

	public static boolean isValueType(String qualifiedTypeName) {
		Entry e = ENTRIES.get(qualifiedTypeName);
		return e != null && e.isValueType();
	}

	/** Go type name, package-qualified when the manual type lives outside swt (e.g. "jrt.Error"). */
	public static String goTypeName(String qualifiedTypeName) {
		Entry e = ENTRIES.get(qualifiedTypeName);
		return e != null ? e.goType() : "unsupported_manual_type_" + qualifiedTypeName;
	}

	/** Import path a manual type's Go spelling needs (e.g. "jrt.Error" needs internal/jrt), or null. */
	public static String importPath(String qualifiedTypeName) {
		Entry e = ENTRIES.get(qualifiedTypeName);
		return e == null ? null : e.importPath();
	}

	/** Whether qualifiedTypeName may serve as an embedded Go base for a translated subclass. */
	public static boolean isManualSuper(String qualifiedTypeName) {
		return qualifiedTypeName.equals(JAVA_RUNTIME_EXCEPTION) || qualifiedTypeName.equals(JAVA_ERROR)
				|| qualifiedTypeName.equals(JAVA_EVENT_OBJECT);
	}

	// Manual superclass of a manual widget stub (Emitter.upcastObject climbs it). Empty since
	// ScrollBar became a real ClassInfo (Round 8), Menu in Round 7.
	private static final Map<String, String> WIDGET_SUPER = Map.of();

	/** qualifiedTypeName's superclass in the manual widget-hierarchy chain above, or null. */
	public static String manualSuperclassOf(String qualifiedTypeName) {
		return WIDGET_SUPER.get(qualifiedTypeName);
	}

	/** Embedded field name for a manual superclass: its Go type name, without a package qualifier. */
	public static String manualSuperFieldName(String qualifiedTypeName) {
		String t = goTypeName(qualifiedTypeName);
		int dot = t.indexOf('.');
		return dot < 0 ? t : t.substring(dot + 1);
	}

	/** Go constructor function for `new T(...)` or a manual-superclass super(...) call. */
	public static String ctorFuncName(String qualifiedTypeName) {
		String t = goTypeName(qualifiedTypeName);
		int dot = t.indexOf('.');
		return dot < 0 ? ("New" + t) : (t.substring(0, dot) + ".New" + t.substring(dot + 1));
	}

	/** Static method of a manual type: its name plus a tag for the overloads the hand-written stand-in tells apart by type. */
	public static String staticMethod(String qualifiedTypeName, org.eclipse.jdt.core.dom.IMethodBinding mb) {
		String name = staticMember(qualifiedTypeName, mb.getName());
		if (!qualifiedTypeName.equals("org.eclipse.swt.internal.DPIUtil") || mb.getParameterTypes().length == 0) return name;
		String first = mb.getParameterTypes()[0].getName();
		if (first.equals("float") || first.equals("double")) return name + Names.capitalize(first);
		if (mb.getName().equals("scaleImageData") && mb.getParameterTypes().length == 3) return name + "Element";
		if (mb.getName().equals("getScalingFactor") && mb.getParameterTypes().length == 2) return name + "Zooms";
		return name;
	}

	/** Static field/method reference on a manual type: always Class+Name (no special-casing left). */
	public static String staticMember(String qualifiedTypeName, String javaMemberName) {
		if (qualifiedTypeName.equals(JAVA_BOOLEAN) && (javaMemberName.equals("TRUE") || javaMemberName.equals("FALSE"))) return javaMemberName.toLowerCase();
		return goTypeName(qualifiedTypeName) + Names.capitalize(javaMemberName);
	}

	/** Instance method on a manual type value, e.g. RoundingMode.round(float) or stash.Close(). */
	public static String instanceMember(String javaMemberName) {
		return Names.capitalize(javaMemberName);
	}

	/** Every manual type's own (same-Go-package) type name - manual types in another package
	 * (jrt.*, reflect.Type) can't collide with an swt-package identifier. */
	public static Set<String> ownPackageTypeNames() {
		Set<String> s = new java.util.HashSet<>();
		for (var e : ENTRIES.entrySet()) {
			if (e.getValue().importPath() == null && isManual(e.getKey())) s.add(e.getValue().goType());
		}
		return s;
	}
}
