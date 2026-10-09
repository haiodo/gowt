package dev.gowt.j2go;

import dev.gowt.j2go.emit.Emitter;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;

import java.util.ArrayList;
import java.util.List;

/** Java type -> Go type text (see contract's "Types" section). */
public class GoTypes {

	// Java package -> Go package routing: shared by Main (a file's own output dir) and the
	// cross-package qualification below (whether a referenced type needs "cocoa." + an import).
	/** Go package of the PI bindings being generated ("cocoa", "win32", "gtk"); set once by Main. */
	public static String piPackage = "cocoa";
	// Set once by Main: the platform being generated.
	public static Platform platform = Platform.COCOA;

	/** A Go package that holds a platform's PI bindings (bottom layer, no swt imports). */
	public static boolean isPiGoPackage(String goPackage) {
		return goPackage.equals("cocoa") || goPackage.equals("win32") || goPackage.equals("gtk");
	}

	public static boolean isCocoaPackage(String javaPackage) {
		return isPiJavaPackage(javaPackage) || javaPackage.equals("org.eclipse.swt.internal");
	}

	/** Go value types mirroring C structs: cocoa's and gtk's (Win32's Java structs are reference classes, packed by toC/fromC). */
	public static boolean isStructPackage(String javaPackage) {
		return platform != Platform.WIN32 && isCocoaPackage(javaPackage);
	}

	// gtk's PI is several Java packages (gtk, gtk3, gtk4, cairo), win32's is four (win32, win32.version, gdip, ole.win32): one Go package each.
	public static boolean isPiJavaPackage(String javaPackage) {
		return javaPackage.equals("org.eclipse.swt.internal." + piPackage)
				|| piPackage.equals("gtk") && javaPackage.matches("org\\.eclipse\\.swt\\.internal\\.(gtk3|gtk4|cairo)")
				|| piPackage.equals("win32") && javaPackage.matches("org\\.eclipse\\.swt\\.internal\\.(win32\\.version|gdip|ole\\.win32)");
	}

	private static final String EXAMPLES_PACKAGE = "org.eclipse.swt.examples.";
	// Round 12: the JUnit tests and their helper packages (tests.junit, tests.graphics) - one Go package.
	private static final String TESTS_PACKAGE = "org.eclipse.swt.tests.";
	// JFace has no platform code: one Go package for all of org.eclipse.jface.* (see README "Round 22").
	private static final String JFACE_TESTS_PACKAGE = "org.eclipse.jface.tests.";

	/** Repo-relative Go package dir of a top-level Java class. Of org.eclipse.swt.internal only
	 * PI's C belongs to cocoa; the common helpers there (TransparencyColorImageGcDrawer) use swt types. */
	public static String goPackageDir(String javaPackage, String topLevelName) {
		// DnD subclasses it (anonymously too) with the cascade of swt's own classes: same Go package as the subclasses (README "Round 24 dnd").
		if (javaPackage.equals("org.eclipse.swt.internal.ole.win32") && topLevelName.equals("COMObject")) return "swt";
		if (isPiJavaPackage(javaPackage)) return "internal/" + piPackage;
		if (javaPackage.equals("org.eclipse.swt.internal") && topLevelName.equals("C")) return "internal/" + piPackage;
		// Converter is PI's own (OS.getThemeName calls it) and needs nothing above it.
		if (piPackage.equals("gtk") && javaPackage.equals("org.eclipse.swt.internal") && topLevelName.equals("Converter")) return "internal/gtk";
		if (javaPackage.startsWith(TESTS_PACKAGE)) return "tests/swttests";
		if (javaPackage.startsWith(JFACE_TESTS_PACKAGE)) return "tests/jfacetests";
		if (javaPackage.equals("org.eclipse.swt.browser")) return "browser";
		if (javaPackage.equals("org.eclipse.jface") || javaPackage.startsWith("org.eclipse.jface.")) return "jface";
		if (javaPackage.startsWith(EXAMPLES_PACKAGE)) return "examples/" + javaPackage.substring(EXAMPLES_PACKAGE.length()).replace('.', '/');
		return "swt";
	}

	public static String goPackageOf(String javaPackage, String topLevelName) {
		String dir = goPackageDir(javaPackage, topLevelName);
		return dir.substring(dir.lastIndexOf('/') + 1);
	}

	/** Import path of a Go package produced by goPackageDir (package names are unique). */
	public static String importPath(String goPackage) {
		return "github.com/haiodo/gowt/" + switch (goPackage) {
			case "cocoa", "win32", "gtk" -> "internal/" + goPackage;
			case "swt" -> "swt";
			case "swttests" -> "tests/swttests";
			case "jfacetests" -> "tests/jfacetests";
			case "jface" -> "jface";
			case "browser" -> "browser";
			default -> "examples/" + goPackage;
		};
	}

	/** Import layering: cocoa < swt < examples; a package may only reference lower layers. */
	public static int layer(String goPackage) {
		return switch (goPackage) {
			case "cocoa", "win32", "gtk" -> 0;
			case "swt" -> 1;
			default -> 2;
		};
	}

	public static String map(ITypeBinding t, Emitter emitter) {
		TypeModel model = emitter.model;
		if (t.isArray()) {
			return "[]" + map(t.getComponentType(), emitter);
		}
		if (t.isPrimitive()) {
			return switch (t.getName()) {
				case "int" -> "int32";
				case "long" -> "int64";
				case "short" -> "int16";
				case "byte" -> "int8";
				case "char" -> "uint16";
				case "float" -> "float32";
				case "double" -> "float64";
				case "boolean" -> "bool";
				case "void" -> "";
				default -> "unsupported_primitive_" + t.getName();
			};
		}
		// `var s = new Base() {...}`: Go has no anonymous types; the variable is typed by what the class extends.
		if (t.isAnonymous() && t.getSuperclass() != null && !t.getSuperclass().getQualifiedName().equals("java.lang.Object")) t = t.getSuperclass();
		String qualified = t.getErasure().getQualifiedName();
		if (qualified.equals("java.lang.String")) return "string";
		if (qualified.equals("java.lang.Object")) return "any";
		// Thrown values are Go panics recovered as error (ControlFlowEmitter's catch dispatch), so
		// every Java exception type used as a value type is error - SWTException embeds the jrt struct.
		if (qualified.equals(Manual.JAVA_RUNTIME_EXCEPTION) || qualified.equals(Manual.JAVA_ERROR)
				|| qualified.equals(Manual.JAVA_EXCEPTION) || qualified.equals(Manual.JAVA_THROWABLE)) return "error";
		// java.util.function.Consumer<T>: no lambda/method-ref support yet (see README), but a
		// functional-interface PARAMETER type still needs a real Go type to compile at all.
		if (qualified.equals("java.util.function.Consumer")) {
			ITypeBinding[] args = t.getTypeArguments();
			String argType = args.length > 0 ? map(args[0], emitter) : "any";
			return "func(" + argType + ")";
		}
		if (isJdkFunctional(qualified) && t.getFunctionalInterfaceMethod() != null) return funcType(t.getFunctionalInterfaceMethod(), emitter);
		TypeModel.ClassInfo ci = model.lookup(t);
		if (ci != null) {
			String name = emitter.qualifiedTypeName(ci);
			return (ci.isStruct || ci.isInterface) ? name : "*" + name;
		}
		if (Manual.isManual(qualified)) {
			emitter.addManualImport(qualified);
			String goType = emitter.qualifyManual(Manual.goTypeName(qualified), t);
			return Manual.isValueType(qualified) ? goType : "*" + goType;
		}
		// Any other JDK type (Locale, Cleaner, StringBuilder, ...) degrades to any: every member
		// access on it is already an unresolved-call marker, so only on-path uses need a mapping.
		if (qualified.startsWith("java.")) return "any";
		// Tests reference SWT API not translated yet: it degrades like the JDK, a test using it fails.
		if (emitter.degradesUnresolvedTypes()) return "any";
		emitter.checkNoForeignPackageLeak(qualified);
		// Not a real Go identifier (still undefined - go vet reports it plainly instead of gofmt
		// choking on a qualified-name-shaped parse error).
		// A local class has no qualified name: the caller's own panic marker needs only some type.
		if (qualified.isEmpty()) return "any";
		return "unsupported_type_" + qualified.replace('.', '_');
	}

	/** java.util.function.* and Comparator are bare Go funcs (lambdas need no adapter, a call is a Go call). */
	public static boolean isJdkFunctional(String qualified) {
		return qualified.startsWith("java.util.function.") || qualified.equals("java.util.Comparator")
				|| qualified.equals("java.util.concurrent.Callable") || qualified.equals("com.sun.net.httpserver.HttpHandler") || qualified.equals("org.eclipse.swt.SwtCallable");
	}

	private static String funcType(IMethodBinding sam, Emitter emitter) {
		List<String> params = new ArrayList<>();
		for (ITypeBinding p : sam.getParameterTypes()) params.add(map(p, emitter));
		String ret = sam.getReturnType().getName().equals("void") ? "" : " " + map(sam.getReturnType(), emitter);
		return "func(" + String.join(", ", params) + ")" + ret;
	}

	/** Zero-width bit size of a Go primitive integer/float type, for >>> and cast rules. */
	static int bits(String goPrimitive) {
		return switch (goPrimitive) {
			case "int8" -> 8;
			case "int16", "uint16" -> 16;
			case "int32", "float32" -> 32;
			case "int64", "float64" -> 64;
			default -> 0;
		};
	}

	static boolean isFloat(String goPrimitive) {
		return goPrimitive.equals("float32") || goPrimitive.equals("float64");
	}

	static boolean isInteger(String goPrimitive) {
		return switch (goPrimitive) {
			case "int8", "int16", "uint16", "int32", "int64" -> true;
			default -> false;
		};
	}
}
