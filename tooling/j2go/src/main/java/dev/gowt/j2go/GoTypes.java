package dev.gowt.j2go;

import dev.gowt.j2go.emit.Emitter;
import org.eclipse.jdt.core.dom.ITypeBinding;

/** Java type -> Go type text (see contract's "Types" section). */
public class GoTypes {

	// Java package -> Go package routing: shared by Main (a file's own output dir) and the
	// cross-package qualification below (whether a referenced type needs "cocoa." + an import).
	public static boolean isCocoaPackage(String javaPackage) {
		return javaPackage.equals("org.eclipse.swt.internal.cocoa") || javaPackage.equals("org.eclipse.swt.internal");
	}

	private static final String EXAMPLES_PACKAGE = "org.eclipse.swt.examples.";

	/** Repo-relative Go package dir of a top-level Java class. Of org.eclipse.swt.internal only
	 * PI's C belongs to cocoa; the common helpers there (TransparencyColorImageGcDrawer) use swt types. */
	public static String goPackageDir(String javaPackage, String topLevelName) {
		if (javaPackage.equals("org.eclipse.swt.internal.cocoa")) return "internal/cocoa";
		if (javaPackage.equals("org.eclipse.swt.internal") && topLevelName.equals("C")) return "internal/cocoa";
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
			case "cocoa" -> "internal/cocoa";
			case "swt" -> "swt";
			default -> "examples/" + goPackage;
		};
	}

	/** Import layering: cocoa < swt < examples; a package may only reference lower layers. */
	public static int layer(String goPackage) {
		return switch (goPackage) {
			case "cocoa" -> 0;
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
		TypeModel.ClassInfo ci = model.lookup(t);
		if (ci != null) {
			String name = emitter.qualifiedTypeName(ci);
			return (ci.isStruct || ci.isInterface) ? name : "*" + name;
		}
		if (Manual.isManual(qualified)) {
			emitter.addManualImport(qualified);
			return Manual.isValueType(qualified) ? Manual.goTypeName(qualified) : "*" + Manual.goTypeName(qualified);
		}
		// Any other JDK type (Locale, Cleaner, StringBuilder, ...) degrades to any: every member
		// access on it is already an unresolved-call marker, so only on-path uses need a mapping.
		if (qualified.startsWith("java.")) return "any";
		emitter.checkNoForeignPackageLeak(qualified);
		// Not a real Go identifier (still undefined - go vet reports it plainly instead of gofmt
		// choking on a qualified-name-shaped parse error).
		return "unsupported_type_" + qualified.replace('.', '_');
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
