package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/** Win32 natives: a lazy DLL proc called through syscall.SyscallN (internal/win32/dll_manual.go). */
final class WinNativeEmitter {

	// Java name -> exported symbol where they differ (MoveMemory is a macro over RtlMoveMemory).
	private static final Map<String, String> SYMBOLS = Map.of("MoveMemory", "RtlMoveMemory", "CopyMemory", "RtlMoveMemory");

	// os_custom.c natives with no DLL export of their own: constants, and hand-written custom_<name> (custom_manual.go).
	private static final Map<String, String> CONSTANTS = Map.of("DPI_AWARENESS_CONTEXT_UNAWARE", "-1", "DPI_AWARENESS_CONTEXT_SYSTEM_AWARE", "-2",
			"DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE", "-3", "DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2", "-4", "DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED", "-5");
	private static final java.util.Set<String> CUSTOM = java.util.Set.of("GetLibraryHandle", "AllowDarkModeForWindow", "SetPreferredAppMode", "IsDarkModeAvailable", "GID_ROTATE_ANGLE_FROM_ARGUMENT", "TreeView_GetItemRect");

	private final Emitter emitter;
	// Statements around the call for struct arguments: toC into a C-layout buffer before, fromC after.
	private final StringBuilder pre = new StringBuilder();
	private final StringBuilder post = new StringBuilder();

	WinNativeEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	private String paramName(MethodDeclaration md, int i) {
		return emitter.sanitizeIdent(((SingleVariableDeclaration) md.parameters().get(i)).getName().getIdentifier());
	}

	void emit(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String goName = ci.goFuncPrefix + emitter.names.goMemberName(mb, Names.capitalize(javaName));
		ITypeBinding[] types = mb.getParameterTypes();
		// GDI+ is C++ in os: the flat API is bound by hand (internal/win32/gdip_manual.go) once GC needs it.
		if (ci.binding.getName().equals("Gdip") && !javaName.endsWith("_sizeof")) return;
		if (javaName.endsWith("_sizeof") && types.length == 0) {
			String typeName = javaName.substring(0, javaName.length() - "_sizeof".length());
			Integer size = WinLayout.size(typeName);
			if (size != null) {
				out.append("func ").append(goName).append("() int32 { return ").append(size).append(" }\n\n");
				return;
			}
			for (TypeModel.ClassInfo c : emitter.model.all()) {
				if (c.binding.getName().equals(typeName)) {
					emitter.fileImports.add("unsafe");
					out.append("func ").append(goName).append("() int32 { return int32(unsafe.Sizeof(").append(c.goTypeName).append("{})) }\n\n");
					return;
				}
			}
		}
		if (javaName.equals("PTR_sizeof") && types.length == 0) {
			out.append("func ").append(goName).append("() int32 { return 8 }\n\n");
			return;
		}
		if (CONSTANTS.containsKey(javaName)) {
			out.append("func ").append(goName).append("() int64 { return ").append(CONSTANTS.get(javaName)).append(" }\n\n");
			return;
		}
		Integer constant = types.length == 0 ? WinLayout.size(javaName) : null;
		if (constant != null) {
			out.append("func ").append(goName).append("() int32 { return ").append(constant).append(" }\n\n");
			return;
		}
		if (CUSTOM.contains(javaName)) {
			List<String> cp = new ArrayList<>();
			List<String> ca = new ArrayList<>();
			for (int i = 0; i < types.length; i++) {
				cp.add(paramName(md, i) + " " + dev.gowt.j2go.GoTypes.map(types[i], emitter));
				ca.add(paramName(md, i));
			}
			String r = emitter.retType(mb);
			out.append("func ").append(goName).append('(').append(String.join(", ", cp)).append(") ").append(r).append(" { return custom_").append(javaName)
					.append('(').append(String.join(", ", ca)).append(") }\n\n");
			return;
		}
		// No Windows export behind these (os.c's own helpers, WebView2 glue): panic when called.
		if (java.util.Set.of("setenv", "PathToPIDL", "CreateSwtWebView2Options").contains(javaName)) {
			emitter.emitUnsupportedNative(md, ci, out);
			return;
		}
		boolean vtbl = javaName.startsWith("VtblCall") && types.length >= 2;
		List<String> ps = new ArrayList<>();
		List<String> args = new ArrayList<>();
		pre.setLength(0);
		post.setLength(0);
		for (int i = 0; i < types.length; i++) {
			String arg = argExpr(md, types[i], i);
			if (arg == null) {
				emitter.emitUnsupportedNative(md, ci, out);
				return;
			}
			ps.add(paramName(md, i) + " " + dev.gowt.j2go.GoTypes.map(types[i], emitter));
			if (!vtbl || i >= 1) args.add(arg);
		}
		String ret = emitter.retType(mb);
		String conv = retConv(mb.getReturnType().getName());
		if (conv == null) {
			emitter.emitUnsupportedNative(md, ci, out);
			return;
		}
		String symbol = SYMBOLS.getOrDefault(javaName, javaName);
		String dynamic = TypeModel.javadocTag(md, "@method", null).contains("flags=dynamic") ? ", true" : "";
		emitter.fileImports.add("syscall");
		emitter.fileImports.add("unsafe");
		String pv = "proc_" + goName;
		String target;
		if (vtbl) {
			target = "vtblFn(" + paramName(md, 1) + ", " + paramName(md, 0) + ")";
		} else {
			out.append("var ").append(pv).append(" = newProc(\"").append(symbol).append('"').append(dynamic).append(")\n");
			target = pv + ".addr()";
		}
		out.append("func ").append(goName).append('(').append(String.join(", ", ps)).append(") ").append(ret).append(" {\n").append(pre);
		// GOWT_TRACE_DISPATCH=1 keeps the last messages (internal/win32/trace_manual.go).
		if (javaName.equals("DispatchMessage")) out.append("\ttraceDispatch(").append(paramName(md, 0)).append(")\n");
		String call = "syscall.SyscallN(" + target + (args.isEmpty() ? "" : ", " + String.join(", ", args)) + ")";
		if (ret.isEmpty()) {
			out.append('\t').append(call).append('\n').append(post);
		} else if (ret.startsWith("float")) {
			emitter.fileImports.add("math");
			out.append("\t_, r2, _ := ").append(call).append('\n').append(post).append("\treturn math.Float").append(ret.substring(5)).append("frombits(")
					.append(ret.equals("float32") ? "uint32(r2)" : "uint64(r2)").append(")\n");
		} else {
			out.append("\tr, _, _ := ").append(call).append('\n').append(post).append("\treturn ").append(conv).append('\n');
		}
		out.append("}\n\n");
	}

	private String retConv(String javaType) {
		return switch (javaType) {
			case "void", "float", "double" -> "-";
			case "int" -> "int32(r)";
			case "long" -> "int64(r)";
			case "short" -> "int16(r)";
			case "byte" -> "int8(r)";
			case "char" -> "uint16(r)";
			case "boolean" -> "r&0xff != 0";
			default -> null;
		};
	}

	/** The uintptr argument text, or null for a parameter type no Win32 call can take. */
	private String argExpr(MethodDeclaration md, ITypeBinding t, int i) {
		String n = paramName(md, i);
		if (t.isArray()) {
			return t.getComponentType().isPrimitive() ? "uintptr(unsafe.Pointer(unsafe.SliceData(" + n + ")))" : null;
		}
		if (t.isPrimitive()) {
			return switch (t.getName()) {
				case "boolean" -> "boolToUintptr(" + n + ")";
				case "float" -> floatBits(n, "32");
				case "double" -> floatBits(n, "64");
				default -> "uintptr(" + n + ")";
			};
		}
		TypeModel.ClassInfo target = emitter.model.lookup(t);
		Integer size = target == null || target.isInterface ? null : WinLayout.size(target.binding.getName());
		if (size == null) return null;
		String javaName = ((SingleVariableDeclaration) md.parameters().get(i)).getName().getIdentifier();
		String in = TypeModel.javadocTag(md, "@param", javaName);
		pre.append("\tvar b_").append(n).append(" [").append((size + 7) / 8).append("]uint64\n\tvar p_").append(n).append(" unsafe.Pointer\n\tif ")
				.append(n).append(" != nil {\n\t\tp_").append(n).append(" = unsafe.Pointer(&b_").append(n).append(")\n");
		if (!in.contains("no_in")) pre.append("\t\t").append(n).append(".toC(p_").append(n).append(")\n");
		pre.append("\t}\n");
		if (!in.contains("no_out") && !in.contains("struct")) {
			post.append("\tif ").append(n).append(" != nil {\n\t\t").append(n).append(".fromC(p_").append(n).append(")\n\t}\n");
		}
		return in.contains("struct") ? "structArg(p_" + n + ", " + size + ")" : "uintptr(p_" + n + ")";
	}

	private String floatBits(String n, String width) {
		emitter.fileImports.add("math");
		return "uintptr(math.Float" + width + "bits(" + n + "))";
	}
}
