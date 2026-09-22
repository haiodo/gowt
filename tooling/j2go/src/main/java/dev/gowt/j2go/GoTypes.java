package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.ITypeBinding;

/** Java type -> Go type text (see contract's "Types" section). */
class GoTypes {

	static String map(ITypeBinding t, TypeModel model) {
		if (t.isArray()) {
			return "[]" + map(t.getComponentType(), model);
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
		TypeModel.ClassInfo ci = model.lookup(t);
		if (ci != null) return "*" + ci.goTypeName;
		if (Manual.isManual(qualified)) {
			return Manual.isValueType(qualified) ? Manual.goTypeName(qualified) : "*" + Manual.goTypeName(qualified);
		}
		return "unsupported_type_" + qualified;
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
