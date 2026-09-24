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
	private static final String TOOL_BAR = "org.eclipse.swt.widgets.ToolBar";
	private static final String CARET = "org.eclipse.swt.widgets.Caret";
	private static final String IME = "org.eclipse.swt.widgets.IME";
	private static final String ACCESSIBLE = "org.eclipse.swt.accessibility.Accessible";
	private static final String ACC = "org.eclipse.swt.accessibility.ACC";
	private static final String WIDGET_SPY = "org.eclipse.swt.internal.WidgetSpy";
	private static final String DIALOG = "org.eclipse.swt.widgets.Dialog";
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
		reg(TOOL_BAR, "ToolBar", null, false);
		reg(CARET, "Caret", null, false);
		reg(IME, "IME", null, false);
		reg(ACCESSIBLE, "Accessible", null, false);
		reg(ACC, "ACC", null, false);
		reg(WIDGET_SPY, "WidgetSpy", null, false);
		reg(DIALOG, "Dialog", null, false);
		reg(AUTOSCALING_MODE, "AutoscalingMode", null, true);
		reg(JAVA_THREAD, "any", null, true);
		reg(JAVA_INTEGER, "any", null, true);
		reg(JAVA_BOOLEAN, "bool", null, true);
		reg(JAVA_CLASS, "reflect.Type", "reflect", true);
		reg(JAVA_RUNNABLE, "jrt.Runnable", JRT_IMPORT, true);
		// java.util containers: hand-written in internal/jrt (util.go), erased to any elements.
		for (String q : new String[]{"java.util.Map", "java.util.HashMap"}) reg(q, "jrt.Map", JRT_IMPORT, false);
		for (String q : new String[]{"java.util.List", "java.util.ArrayList", "java.util.concurrent.ConcurrentLinkedQueue"}) {
			reg(q, "jrt.List", JRT_IMPORT, false);
		}
		reg(SWT_LONG, "LONG", null, false);
		// org.eclipse.swt.internal helpers referencing swt types (so not translatable into cocoa):
		// only their static members are used, hand-written in swt/internal_helpers_manual.go.
		for (String n : new String[]{"DPIUtil", "BidiUtil", "graphics.ImageUtil", "Compatibility", "DefaultExceptionHandler"}) {
			String q = "org.eclipse.swt.internal." + n;
			reg(q, q.substring(q.lastIndexOf('.') + 1), null, false);
		}
		// GC's text-layout cache: a record key and an LRU LinkedHashMap subclass, hand-written in
		// swt/graphics_gc_manual.go (no record rule; removeEldestEntry has no jrt.Map equivalent).
		reg("org.eclipse.swt.graphics.GC.GCTextData.Key", "GC_GCTextData_Key", null, false);
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
		// Widgets/graphics Display only references off the Shell+Button path (dialogs, tray,
		// dock menu, combo tracking, font metrics): opaque stubs in swt/widgets_stubs3_manual.go.
		for (String n : new String[]{"widgets.FontDialog", "widgets.FileDialog", "widgets.ColorDialog", "widgets.Combo",
				"widgets.TaskBar", "widgets.TaskItem", "widgets.Tray", "widgets.TrayItem"}) {
			String q = "org.eclipse.swt." + n;
			reg(q, q.substring(q.lastIndexOf('.') + 1), null, false);
		}
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
			// os.c's `((void (*)())proc)(id, sel)`: a call through a C function pointer.
			Map.entry(COCOA_PKG + "OS#call(J,J,J)", "OSCall"),
			// A fixed "return YES" IMP in os_custom.c, no Java/Go call (callback_manual.go).
			Map.entry(COCOA_PKG + "OS#isFlipped_CALLBACK()", "OSIsFlipped_CALLBACK"),
			// Compares a Java class name to "org.eclipse.swt.widgets." - a Go type's name can't
			// match that, so it checks the Go package instead (swt/widgets_stubs3_manual.go).
			Map.entry("org.eclipse.swt.widgets.Display#isValidClass(java.lang.Class)", "DisplayIsValidClass"),
			Map.entry("java.lang.Thread#currentThread()", "ThreadCurrentThread"));

	// Dropped: Selector.java's own bookkeeping is elided (see README). Matched by name only.
	private static final Set<String> SKIP_METHOD_NAMES = Set.of(
			COCOA_PKG + "OS#registerSelector", COCOA_PKG + "OS#getSelector");

	public static boolean isManual(String qualifiedTypeName) {
		return ENTRIES.containsKey(qualifiedTypeName);
	}

	/** A value type with no real Go method surface (java.util.Map, ...) - method calls on it
	 * can't dispatch through Manual.instanceMember and must fall back to the unresolved-call
	 * degrade instead (see InvocationEmitter.emitMethodInvocation). */
	public static boolean isBareAny(String qualifiedTypeName) {
		Entry e = ENTRIES.get(qualifiedTypeName);
		return e != null && e.isValueType() && e.goType().equals("any");
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
	static boolean isManualSuper(String qualifiedTypeName) {
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

	/** Static field/method reference on a manual type: always Class+Name (no special-casing left). */
	public static String staticMember(String qualifiedTypeName, String javaMemberName) {
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
		for (Entry e : ENTRIES.values()) {
			if (e.importPath() == null) s.add(e.goType());
		}
		return s;
	}
}
