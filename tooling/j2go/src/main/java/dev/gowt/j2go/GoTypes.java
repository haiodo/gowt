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

	public static String goPackageOf(String javaPackage) {
		return isCocoaPackage(javaPackage) ? "cocoa" : "swt";
	}

	public static final String COCOA_IMPORT = "github.com/haiodo/gowt/internal/cocoa";

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
			return Manual.isValueType(qualified) ? Manual.goTypeName(qualified) : "*" + Manual.goTypeName(qualified);
		}
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
