package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Platform;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

/**
 * The toC/fromC methods of a Win32 PI struct: what os_structs.c's get/set<Struct>Fields do in JNI. The Go
 * struct keeps SWT's Java shape; a native copies it into a C-layout buffer and back around the call.
 */
final class WinPack {

	private WinPack() {}

	// C bitfield structs: packed by hand (internal/win32/bitfields_manual.go).
	static final java.util.Set<String> MANUAL = java.util.Set.of("MENUBARINFO", "SCRIPT_ANALYSIS", "SCRIPT_CONTROL",
			"SCRIPT_LOGATTR", "SCRIPT_PROPERTIES", "SCRIPT_STATE");

	static String emit(Emitter emitter, TypeModel.ClassInfo ci) {
		String cls = ci.binding.getName();
		if (GoTypes.platform != Platform.WIN32 || !GoTypes.isPiJavaPackage(ci.javaPackage) || WinLayout.size(cls) == null
				|| MANUAL.contains(cls)) return "";
		StringBuilder to = new StringBuilder("func (this *" + ci.goTypeName + ") toC(p unsafe.Pointer) {\n");
		StringBuilder from = new StringBuilder("func (this *" + ci.goTypeName + ") fromC(p unsafe.Pointer) {\n");
		if (ci.superclass != null && WinLayout.size(ci.superclass.binding.getName()) != null) {
			to.append("\tthis.").append(ci.superclass.goTypeName).append(".toC(p)\n");
			from.append("\tthis.").append(ci.superclass.goTypeName).append(".fromC(p)\n");
		}
		for (IVariableBinding vb : ci.binding.getDeclaredFields()) {
			if (Modifier.isStatic(vb.getModifiers())) continue;
			WinLayout.Field lf = WinLayout.field(cls, vb.getName());
			if (lf == null) continue;
			field(emitter, vb, lf, to, from);
		}
		emitter.fileImports.add("unsafe");
		return to + "}\n\n" + from + "}\n\n";
	}

	private static void field(Emitter emitter, IVariableBinding vb, WinLayout.Field lf, StringBuilder to, StringBuilder from) {
		String f = "this." + emitter.fieldGoName(vb);
		ITypeBinding t = vb.getType();
		String at = "unsafe.Add(p, " + lf.offset() + ")";
		if (t.isArray()) {
			String elem = t.getComponentType().getName();
			int w = width(elem);
			String ct = cType(elem, w, w);
			String goElem = GoTypes.map(t.getComponentType(), emitter);
			String loop = "\tfor i := 0; i < " + lf.size() / w + " && i < len(" + f + "); i++ {\n";
			to.append(loop).append("\t\t*(*").append(ct).append(")(unsafe.Add(").append(at).append(", i*").append(w).append(")) = ").append(ct).append('(').append(f).append("[i])\n\t}\n");
			from.append(loop).append("\t\t").append(f).append("[i] = ").append(goElem).append("(*(*").append(ct).append(")(unsafe.Add(").append(at).append(", i*").append(w).append(")))\n\t}\n");
			return;
		}
		if (t.isPrimitive()) {
			String name = t.getName();
			if (name.equals("boolean")) {
				String ct = cType("int", lf.size(), 4);
				to.append("\t*(*").append(ct).append(")(").append(at).append(") = 0\n\tif ").append(f).append(" {\n\t\t*(*").append(ct).append(")(").append(at).append(") = 1\n\t}\n");
				from.append('\t').append(f).append(" = *(*").append(ct).append(")(").append(at).append(") != 0\n");
				return;
			}
			String ct = cType(name, lf.size(), width(name));
			String goT = GoTypes.map(t, emitter);
			to.append("\t*(*").append(ct).append(")(").append(at).append(") = ").append(ct).append('(').append(f).append(")\n");
			from.append('\t').append(f).append(" = ").append(goT).append("(*(*").append(ct).append(")(").append(at).append("))\n");
			return;
		}
		TypeModel.ClassInfo nested = emitter.model.lookup(t);
		if (nested == null || WinLayout.size(nested.binding.getName()) == null) return;
		to.append("\tif ").append(f).append(" != nil {\n\t\t").append(f).append(".toC(").append(at).append(")\n\t}\n");
		from.append("\tif ").append(f).append(" == nil {\n\t\t").append(f).append(" = New").append(nested.goTypeName).append("()\n\t}\n\t")
				.append(f).append(".fromC(").append(at).append(")\n");
	}

	private static int width(String javaType) {
		return switch (javaType) {
			case "byte" -> 1;
			case "short", "char" -> 2;
			case "int", "float" -> 4;
			default -> 8;
		};
	}

	/** Go type to store/load a C field of cWidth bytes from a Java type: zero-extended when Java is wider. */
	private static String cType(String javaType, int cWidth, int javaWidth) {
		if (javaType.equals("float")) return "float32";
		if (javaType.equals("double")) return "float64";
		boolean unsigned = javaType.equals("char") || javaWidth > cWidth;
		return (unsigned ? "uint" : "int") + cWidth * 8;
	}
}
