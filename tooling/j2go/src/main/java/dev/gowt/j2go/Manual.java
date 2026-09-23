package dev.gowt.j2go;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

/** Names for out-of-translated-set members hand-written in *_manual.go (see manual.txt). */
public class Manual {

	private static final String MONITOR = "org.eclipse.swt.widgets.Monitor";
	private static final String ROUNDING_MODE = "org.eclipse.swt.graphics.RoundingMode";
	private static final String PLATFORM = "org.eclipse.swt.internal.Platform";
	private static final String LIBRARY = "org.eclipse.swt.internal.Library";
	private static final String DISPLAY = "org.eclipse.swt.widgets.Display";
	private static final String GRAPHICS_GC = "org.eclipse.swt.graphics.GC";
	private static final String TOUCH = "org.eclipse.swt.widgets.Touch";
	private static final String EXCEPTION_STASH = "org.eclipse.swt.internal.ExceptionStash";
	private static final String CALLBACK = "org.eclipse.swt.internal.Callback";
	private static final String COMPOSITE = "org.eclipse.swt.widgets.Composite";
	private static final String SHELL = "org.eclipse.swt.widgets.Shell";
	private static final String DECORATIONS = "org.eclipse.swt.widgets.Decorations";
	private static final String CANVAS = "org.eclipse.swt.widgets.Canvas";
	private static final String MENU = "org.eclipse.swt.widgets.Menu";
	private static final String SCROLL_BAR = "org.eclipse.swt.widgets.ScrollBar";
	private static final String FONT = "org.eclipse.swt.graphics.Font";
	private static final String COLOR = "org.eclipse.swt.graphics.Color";
	private static final String IMAGE = "org.eclipse.swt.graphics.Image";
	private static final String CURSOR = "org.eclipse.swt.graphics.Cursor";
	private static final String REGION = "org.eclipse.swt.graphics.Region";
	private static final String DRAWABLE = "org.eclipse.swt.graphics.Drawable";
	private static final String ACCESSIBLE = "org.eclipse.swt.accessibility.Accessible";
	private static final String ACC = "org.eclipse.swt.accessibility.ACC";
	private static final String WIDGET_SPY = "org.eclipse.swt.internal.WidgetSpy";
	private static final String GC_DATA = "org.eclipse.swt.graphics.GCData";
	// Referenced only as an inherited-method declaring class (display.getSystemFont(),
	// someFont.dispose()), never as their own declared variable type in this file set - aliased
	// to an existing stub rather than adding a distinct, unused Go type.
	private static final String DEVICE = "org.eclipse.swt.graphics.Device";
	private static final String RESOURCE = "org.eclipse.swt.graphics.Resource";
	private static final String DIALOG = "org.eclipse.swt.widgets.Dialog";
	private static final String TOUCH_SOURCE = "org.eclipse.swt.widgets.TouchSource";
	private static final String AUTOSCALING_MODE = "org.eclipse.swt.graphics.AutoscalingMode";
	// checkWidget()'s thread-affinity check: no real Thread, both sides collapse to "any" nil.
	private static final String JAVA_THREAD = "java.lang.Thread";
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
	private static final String JRT_IMPORT = "github.com/haiodo/gowt/internal/jrt";

	private record Entry(String goType, String importPath, boolean isValueType) {}

	private static final Map<String, Entry> ENTRIES = new LinkedHashMap<>();

	private static void reg(String qualified, String goType, String importPath, boolean isValueType) {
		ENTRIES.put(qualified, new Entry(goType, importPath, isValueType));
	}

	static {
		reg(MONITOR, "Monitor", null, false);
		reg(ROUNDING_MODE, "RoundingMode", null, true);
		reg(PLATFORM, "Platform", null, false);
		reg(LIBRARY, "Library", null, false);
		reg(DISPLAY, "Display", null, false);
		reg(GRAPHICS_GC, "GC", null, false);
		reg(TOUCH, "Touch", null, false);
		reg(EXCEPTION_STASH, "ExceptionStash", null, false);
		reg(CALLBACK, "Callback", null, false);
		reg(COMPOSITE, "Composite", null, false);
		reg(SHELL, "Shell", null, false);
		reg(DECORATIONS, "Decorations", null, false);
		reg(CANVAS, "Canvas", null, false);
		reg(MENU, "Menu", null, false);
		reg(SCROLL_BAR, "ScrollBar", null, false);
		reg(FONT, "Font", null, false);
		reg(COLOR, "Color", null, false);
		reg(IMAGE, "Image", null, false);
		reg(CURSOR, "Cursor", null, false);
		reg(REGION, "Region", null, false);
		// A content-free marker interface in the translated set (Control implements it): Go's
		// structural typing needs no explicit "implements", so nothing to embed - see emitClass.
		reg(DRAWABLE, "any", null, true);
		reg(ACCESSIBLE, "Accessible", null, false);
		reg(ACC, "ACC", null, false);
		reg(WIDGET_SPY, "WidgetSpy", null, false);
		reg(GC_DATA, "GCData", null, false);
		reg(DEVICE, "Display", null, false);
		reg(RESOURCE, "any", null, true);
		reg(DIALOG, "Dialog", null, false);
		reg(TOUCH_SOURCE, "any", null, true);
		reg(AUTOSCALING_MODE, "AutoscalingMode", null, true);
		reg(JAVA_THREAD, "any", null, true);
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

	// OS's <Type>_sizeof() natives, hand-written until OS.java itself translates (see
	// internal/cocoa/sizeof_manual.go). Key is Names.erasureKey's format.
	private static final Map<String, String> MANUAL_METHODS = Map.ofEntries(
			manualSizeof("NSPoint"), manualSizeof("NSSize"), manualSizeof("NSRect"), manualSizeof("NSRange"),
			manualSizeof("objc_super"), manualSizeof("CGPoint"), manualSizeof("CGSize"), manualSizeof("CGRect"),
			manualSizeof("NSAffineTransformStruct"), manualSizeof("NSOperatingSystemVersion"),
			manualSizeof("CGPathElement"),
			// id.objc_getClass()/toString(): need reflect over this.impl (see id_manual.go).
			Map.entry(COCOA_PKG + "id#objc_getClass()", "ObjcGetClass"),
			Map.entry(COCOA_PKG + "id#toString()", "String"),
			// Display.map has 2 real overloads called from Control.java; Manual.instanceMember has
			// no overload awareness, so each needs its own erasureKey-keyed name here instead.
			Map.entry("org.eclipse.swt.widgets.Display#map(org.eclipse.swt.widgets.Control,org.eclipse.swt.widgets.Control,int,int)", "Map"),
			Map.entry("org.eclipse.swt.widgets.Display#map(org.eclipse.swt.widgets.Control,org.eclipse.swt.widgets.Control,org.eclipse.swt.graphics.Rectangle)", "MapRect"),
			Map.entry("java.lang.Thread#currentThread()", "ThreadCurrentThread"));

	// Dropped: Selector.java's own usages are elided (see README), and the 3 dark-mode helpers
	// pull in untranslated cocoa types not worth it. Matched by name only, no overloads exist.
	private static final Set<String> SKIP_METHOD_NAMES = Set.of(
			COCOA_PKG + "OS#registerSelector", COCOA_PKG + "OS#getSelector",
			COCOA_PKG + "OS#setTheme", COCOA_PKG + "OS#isAppDarkAppearance",
			COCOA_PKG + "OS#isSystemDarkAppearance");

	private static Map.Entry<String, String> manualSizeof(String type) {
		String key = COCOA_PKG + "OS#" + type + "_sizeof()";
		return Map.entry(key, "OS" + Names.capitalize(type + "_sizeof"));
	}

	public static boolean isManual(String qualifiedTypeName) {
		return ENTRIES.containsKey(qualifiedTypeName);
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

	// The real SWT widget hierarchy above our manual stubs (Composite extends Scrollable, a
	// translated class; every other link here is manual-to-manual): Control.java climbs this
	// chain (e.g. `control = control.parent`) and needs the matching Go embedding + upcast - see
	// Emitter.upcastObject. The stub .go files embed these same types as their first field.
	private static final Map<String, String> WIDGET_SUPER = Map.of(
			COMPOSITE, "org.eclipse.swt.widgets.Scrollable",
			CANVAS, COMPOSITE,
			DECORATIONS, CANVAS,
			SHELL, DECORATIONS,
			MENU, "org.eclipse.swt.widgets.Widget",
			SCROLL_BAR, "org.eclipse.swt.widgets.Widget");

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
}
