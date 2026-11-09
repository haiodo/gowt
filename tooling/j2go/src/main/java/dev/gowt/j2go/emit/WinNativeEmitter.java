package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/** Win32 natives: a lazy DLL proc called through syscall.SyscallN (internal/win32/dll_manual.go). */
final class WinNativeEmitter {

	// Java name -> exported symbol where they differ (MoveMemory is a macro over RtlMoveMemory).
	private static final Map<String, String> SYMBOLS = Map.of("MoveMemory", "RtlMoveMemory", "CopyMemory", "RtlMoveMemory");

	// os_custom.c natives with no DLL export of their own: constants, and hand-written custom_<name> (custom_manual.go).
	private static final Map<String, String> CONSTANTS = Map.of("DPI_AWARENESS_CONTEXT_UNAWARE", "-1", "DPI_AWARENESS_CONTEXT_SYSTEM_AWARE", "-2",
			"DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE", "-3", "DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2", "-4", "DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED", "-5");
	private static final java.util.Set<String> CUSTOM = java.util.Set.of("GetLibraryHandle", "AllowDarkModeForWindow", "SetPreferredAppMode", "IsDarkModeAvailable", "GID_ROTATE_ANGLE_FROM_ARGUMENT", "TreeView_GetItemRect", "PathToPIDL");

	// os.c stubs that dereference a pointer argument before the call: Java name -> index of that argument.
	private static final Map<String, Integer> DEREF = Map.of("ScriptStringOut", 0);

	// Natives that pump messages or call back into Go while they hold a pointer argument (modal loops, common dialogs,
	// message retrieval, OLE creation, enumerations): a callback can move the stack, so the pointer arguments must
	// not live on it (syscallEscaping, internal/win32/dll_manual.go). DoDragDrop returned a stale effect.
	// VtblCall, every COM method call, is escaping as a whole.
	private static final java.util.Set<String> ESCAPING = java.util.Set.of("DoDragDrop", "TrackPopupMenu", "MessageBox", "ChooseColor", "ChooseFont",
			"PrintDlg", "DocumentProperties", "DragDetect", "ShellExecuteEx", "EnumDisplayMonitors", "EnumFontFamilies", "EnumEnhMetaFile",
			"GetMessage", "PeekMessage", "DispatchMessage", "CoCreateInstance", "CoGetClassObject", "OleCreate", "OleCreateFromFile", "OleCreatePropertyFrame",
			"OleGetClipboard");
	// Escaping costs an allocation per call even without a pointer argument, so these escape only with one.
	private static final java.util.Set<String> ESCAPING_WITH_POINTER = java.util.Set.of("SendMessage");

	private final Emitter emitter;
	// Statements around the call for struct arguments: toC into a C-layout buffer before, fromC after.
	private final StringBuilder pre = new StringBuilder();
	private final StringBuilder post = new StringBuilder();
	// The default backend's generated wrappers (syscall.SyscallN) accumulate here, one
	// <unit>_native.go per compilation unit, instead of the unit's own files: the opt-in cgo
	// backend (<unit>_cgo.go, build tag gowtcgo) defines the same functions, so the two bodies
	// must live in mutually exclusive files while everything else the unit emits (structs,
	// sizeof accessors, constant wrappers) stays shared. Null while no native was emitted.
	StringBuilder nativeOut;
	private final java.util.Set<String> nativeImports = new java.util.LinkedHashSet<>();
	private final List<CgoNative> cgoNatives = new ArrayList<>();

	WinNativeEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	void clear() {
		nativeOut = null;
		nativeImports.clear();
		cgoNatives.clear();
	}

	boolean hasNatives() {
		return nativeOut != null;
	}

	java.util.Set<String> nativeImports() {
		return nativeImports;
	}

	String nativeFileBody() {
		return nativeOut == null ? "" : nativeOut.toString();
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
		if (java.util.Set.of("setenv", "CreateSwtWebView2Options").contains(javaName)) {
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
			if (arg != null && DEREF.getOrDefault(javaName, -1) == i) arg = "derefPtr(" + paramName(md, i) + ")";
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
		boolean dynamicFlag = !dynamic.isEmpty();
		boolean escaping = ESCAPING.contains(javaName) || vtbl || ESCAPING_WITH_POINTER.contains(javaName) && hasPointer(types);
		// The SyscallN body is the default backend: it lives in <unit>_native.go (build tag
		// !gowtcgo) alongside the unit, the typed C body in <unit>_cgo.go (build tag gowtcgo).
		StringBuilder nat = nativeBuffer();
		nativeImports.add("syscall");
		nativeImports.add("unsafe");
		String pv = "proc_" + goName;
		String target;
		if (vtbl) {
			target = "vtblFn(" + paramName(md, 1) + ", " + paramName(md, 0) + ")";
		} else {
			nat.append("var ").append(pv).append(" = newProc(\"").append(symbol).append('"').append(dynamic).append(")\n");
			target = pv + ".addr()";
		}
		nat.append("func ").append(goName).append('(').append(String.join(", ", ps)).append(") ").append(ret).append(" {\n").append(pre);
		// GOWT_TRACE_DISPATCH=1 keeps the last messages (internal/win32/trace_manual.go).
		if (javaName.equals("DispatchMessage")) nat.append("\ttraceDispatch(").append(paramName(md, 0)).append(")\n");
		String call = (escaping ? "syscallEscaping(" : "syscall.SyscallN(") + target + (args.isEmpty() ? "" : ", " + String.join(", ", args)) + ")";
		if (ret.isEmpty()) {
			nat.append('\t').append(call).append('\n').append(post);
		} else if (ret.startsWith("float")) {
			nativeImports.add("math");
			nat.append("\t_, r2, _ := ").append(call).append('\n').append(post).append("\treturn math.Float").append(ret.substring(5)).append("frombits(")
					.append(ret.equals("float32") ? "uint32(r2)" : "uint64(r2)").append(")\n");
		} else {
			nat.append("\tr, _, _ := ").append(call).append('\n').append(post).append("\treturn ").append(conv).append('\n');
		}
		nat.append("}\n\n");
		registerCgo(md, mb, ci, javaName, goName, symbol, dynamicFlag, vtbl, types);
	}

	private static boolean hasPointer(ITypeBinding[] types) {
		for (ITypeBinding t : types) if (!t.isPrimitive()) return true;
		return false;
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
		nativeImports.add("math");
		return "uintptr(math.Float" + width + "bits(" + n + "))";
	}

	private StringBuilder nativeBuffer() {
		if (nativeOut == null) nativeOut = new StringBuilder();
		return nativeOut;
	}

	// ---- Opt-in cgo backend (windows && gowtcgo && cgo): one typed C wrapper per native,
	// collected per compilation unit and written as <unit>_cgo.go by Main. The wrappers call
	// through a lazily resolved function pointer (GetProcAddress across the package's DLL list,
	// W variant first, dynamic exports optional - dllcgo_manual.go's gowt_find); floats cross
	// the boundary typed, so the C compiler places them in the float registers SyscallN cannot
	// reach portably. VtblCall natives take the function pointer from the object's vtable as
	// their first C argument instead of resolving a DLL export.

	enum Kind { I32, I64, I16, I8, U16, F32, F64, BOOL, SLICE, STRUCT, DEREF }

	record CParam(String name, String goType, Kind kind, String elemC, int size, boolean byValue, boolean noIn, boolean noOut) {}

	record CgoNative(String binding, String symbol, boolean dynamic, boolean vtbl, boolean trace,
			List<CParam> params, String ret) {}

	private void registerCgo(MethodDeclaration md, IMethodBinding mb, TypeModel.ClassInfo ci,
			String javaName, String goName, String symbol, boolean dynamic, boolean vtbl, ITypeBinding[] types) {
		List<CParam> params = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			params.add(cgoParam(md, types[i], i, javaName));
		}
		cgoNatives.add(new CgoNative(goName, symbol, dynamic, vtbl, javaName.equals("DispatchMessage"), params, emitter.retType(mb)));
	}

	private CParam cgoParam(MethodDeclaration md, ITypeBinding t, int i, String javaName) {
		String n = paramName(md, i);
		String goType = dev.gowt.j2go.GoTypes.map(t, emitter);
		if (DEREF.getOrDefault(javaName, -1) == i) return new CParam(n, goType, Kind.DEREF, null, 0, false, false, false);
		if (t.isArray()) {
			if (!t.getComponentType().isPrimitive()) throw new RuntimeException("j2go: win32 cgo: non-primitive array param " + javaName);
			String elemC = C_SCALARS.get(dev.gowt.j2go.GoTypes.map(t.getComponentType(), emitter));
			if (elemC == null) throw new RuntimeException("j2go: win32 cgo: unmapped array element " + t.getComponentType().getName() + " in " + javaName);
			return new CParam(n, goType, Kind.SLICE, elemC, 0, false, false, false);
		}
		if (t.isPrimitive()) {
			return switch (t.getName()) {
				case "int" -> new CParam(n, goType, Kind.I32, null, 0, false, false, false);
				case "long" -> new CParam(n, goType, Kind.I64, null, 0, false, false, false);
				case "short" -> new CParam(n, goType, Kind.I16, null, 0, false, false, false);
				case "byte" -> new CParam(n, goType, Kind.I8, null, 0, false, false, false);
				case "char" -> new CParam(n, goType, Kind.U16, null, 0, false, false, false);
				case "boolean" -> new CParam(n, goType, Kind.BOOL, null, 0, false, false, false);
				case "float" -> new CParam(n, goType, Kind.F32, null, 0, false, false, false);
				case "double" -> new CParam(n, goType, Kind.F64, null, 0, false, false, false);
				default -> throw new RuntimeException("j2go: win32 cgo: unmapped primitive " + t.getName() + " in " + javaName);
			};
		}
		TypeModel.ClassInfo target = emitter.model.lookup(t);
		Integer size = target == null || target.isInterface ? null : WinLayout.size(target.binding.getName());
		if (size == null) throw new RuntimeException("j2go: win32 cgo: unmapped param type " + t.getName() + " in " + javaName);
		String javaParam = ((SingleVariableDeclaration) md.parameters().get(i)).getName().getIdentifier();
		String in = TypeModel.javadocTag(md, "@param", javaParam);
		return new CParam(n, goType, Kind.STRUCT, null, size, in.contains("struct"), in.contains("no_in"),
				in.contains("no_out") || in.contains("struct"));
	}

	// Go scalar -> C spelling, both of wrapper parameters and of slice element types.
	private static final java.util.Map<String, String> C_SCALARS = java.util.Map.of(
			"int8", "int8_t", "int16", "int16_t", "int32", "int32_t", "int64", "int64_t",
			"uint16", "uint16_t", "float32", "float", "float64", "double");

	private static String cType(CParam p) {
		return switch (p.kind()) {
			case I32 -> "int32_t";
			case I64 -> "int64_t";
			// Win32 takes BOOL/byte/short parameters as full-width int registers: the upper
			// bits are not guaranteed on amd64, so the wrapper widens instead of matching
			// the Go type's width. Returns stay narrow, as the default build's r&0xff does.
			case I16 -> "int32_t";
			case I8 -> "int32_t";
			case U16 -> "uint16_t";
			case BOOL -> "int32_t";
			case F32 -> "float";
			case F64 -> "double";
			case SLICE -> p.elemC() + "*";
			case STRUCT -> "void*";
			case DEREF -> "uintptr_t";
		};
	}

	private static String cRet(String ret) {
		if (ret.isEmpty()) return "void";
		String c = C_SCALARS.get(ret);
		if (c != null) return c;
		if (ret.equals("bool")) return "uint8_t";
		throw new RuntimeException("j2go: win32 cgo: unmapped return type " + ret);
	}

	// The generated <unit>_cgo.go: the C preamble (export resolution + one wrapper per native),
	// then the Go functions with the same signatures as the default build's <unit>_native.go.
	String cgoFileText(String goPackage) {
		if (cgoNatives.isEmpty()) return null;
		boolean usesUnsafe = false;
		for (CgoNative n : cgoNatives) {
			for (CParam p : n.params()) usesUnsafe |= p.kind() == Kind.SLICE || p.kind() == Kind.STRUCT;
		}
		StringBuilder c = new StringBuilder();
		c.append("#include <stdint.h>\n");
		c.append("#include <stdio.h>\n");
		c.append("#include <stdlib.h>\n\n");
		c.append("// gowt_find resolves an export across the package's DLL list (W variant first);\n");
		c.append("// NULL when no DLL has it. dynamic=1 marks an optional export. Defined in dllcgo_manual.go.\n");
		c.append("extern void *gowt_find(const char *name, int dynamic);\n\n");
		appendCStructTypedefs(c);
		for (CgoNative n : cgoNatives) appendCWrapper(n, c);
		StringBuilder go = new StringBuilder();
		for (CgoNative n : cgoNatives) appendGoFunc(n, go);
		// Probe (dll_manual.go) must see the same export set the default build's newProc vars
		// register: one metadata entry per native with a DLL export (VtblCall has none). No
		// binding here - the wrappers resolve lazily through gowt_find.
		go.append("var _ = func() int {\n");
		for (CgoNative n : cgoNatives) {
			if (n.vtbl()) continue;
			go.append("\tregisterProcMeta(\"").append(n.symbol()).append("\", ").append(n.dynamic()).append(")\n");
		}
		go.append("\treturn 0\n}()\n");
		StringBuilder f = new StringBuilder();
		f.append("// Code generated by j2go. DO NOT EDIT.\n");
		f.append("//go:build windows && gowtcgo && cgo\n\n");
		f.append("package ").append(goPackage).append("\n\n");
		f.append("/*\n").append(c).append("*/\n");
		f.append("import \"C\"\n\n");
		if (usesUnsafe) f.append("import \"unsafe\"\n\n");
		f.append(go);
		return f.toString();
	}

	// C twins of the structs a native passes by value (ABI), mirroring the Go struct's fields;
	// a wrapper copies the marshaled buffer into one before the typed call.
	private void appendCStructTypedefs(StringBuilder c) {
		Set<String> emitted = new LinkedHashSet<>();
		for (CgoNative n : cgoNatives) {
			for (CParam p : n.params()) {
				if (p.kind() != Kind.STRUCT || !p.byValue()) continue;
				String name = structName(n, p);
				if (!emitted.add(name)) continue;
				c.append("typedef struct {\n");
				for (CField field : structFields(name)) {
					String ct = C_SCALARS.get(field.goType());
					if (ct == null) throw new RuntimeException("j2go: win32 cgo: struct " + name + " field " + field.goName() + " has non-scalar type " + field.goType());
					c.append('\t').append(ct).append(' ').append(field.goName()).append(";\n");
				}
				c.append("} gowt_").append(name).append(";\n\n");
			}
		}
	}

	private record CField(String goName, String goType) {}

	private List<CField> structFields(String goTypeName) {
		TypeModel.ClassInfo ci = null;
		for (TypeModel.ClassInfo x : emitter.model.all()) {
			if (goTypeName.equals(x.goTypeName)) { ci = x; break; }
		}
		if (ci == null) throw new RuntimeException("j2go: win32 cgo: unknown struct " + goTypeName);
		AbstractTypeDeclaration td = emitter.model.typeDeclarationOf(ci.binding);
		if (td == null) throw new RuntimeException("j2go: win32 cgo: no declaration for struct " + goTypeName + " (" + ci.binaryName + ")");
		List<CField> fields = new ArrayList<>();
		for (Object o : td.bodyDeclarations()) {
			if (!(o instanceof FieldDeclaration fd) || Modifier.isStatic(fd.getModifiers())) continue;
			for (Object fo : fd.fragments()) {
				VariableDeclarationFragment f = (VariableDeclarationFragment) fo;
				String javaName = f.getName().getIdentifier();
				ITypeBinding t = f.resolveBinding() != null ? f.resolveBinding().getType() : fd.getType().resolveBinding();
				fields.add(new CField(Modifier.isPublic(fd.getModifiers()) ? Names.capitalize(javaName) : EmitUtil.fieldIdent(javaName),
						dev.gowt.j2go.GoTypes.map(t, emitter)));
			}
		}
		return fields;
	}

	private void appendCWrapper(CgoNative n, StringBuilder c) {
		List<CParam> ps = n.params();
		int first = n.vtbl() ? 2 : 0;
		List<String> ctypes = new ArrayList<>();
		List<String> cparams = new ArrayList<>();
		List<String> callArgs = new ArrayList<>();
		if (n.vtbl()) {
			cparams.add("uintptr_t fn");
			// The first argument of a COM method is the object the vtable dispatches on.
			CParam self = ps.get(1);
			ctypes.add("uintptr_t");
			cparams.add("int64_t " + self.name());
			callArgs.add(self.name());
		}
		for (int i = first; i < ps.size(); i++) {
			CParam p = ps.get(i);
			// The function pointer takes a by-value struct; the wrapper receives the marshaled
			// buffer and copies it into one before the call.
			ctypes.add(p.kind() == Kind.STRUCT && p.byValue() ? "gowt_" + structName(n, p) : cType(p));
			cparams.add(cType(p) + " " + p.name());
			callArgs.add(p.kind() == Kind.STRUCT && p.byValue() ? "v_" + p.name() : p.name());
		}
		String cret = cRet(n.ret());
		c.append("static ").append(cret).append(' ').append("gowt_").append(n.binding()).append('(')
				.append(String.join(", ", cparams)).append(") {\n");
		if (!n.vtbl()) {
			c.append("\tstatic ").append(cret).append(" (*p)(").append(String.join(", ", ctypes)).append(");\n");
			c.append("\tif (p == 0) {\n");
			c.append("\t\t*(void **)&p = gowt_find(\"").append(n.symbol()).append("\", ").append(n.dynamic() ? 1 : 0).append(");\n");
			c.append("\t\tif (p == 0) {\n");
			if (n.dynamic()) {
				// An absent dynamic export stands in as 0; a void native just returns.
				c.append(n.ret().isEmpty() ? "\t\t\treturn;\n" : "\t\t\treturn 0;\n");
			} else {
				c.append("\t\t\tfprintf(stderr, \"win32: no DLL exports %s\\n\", \"").append(n.symbol()).append("\");\n");
				c.append("\t\t\tabort();\n");
			}
			c.append("\t\t}\n\t}\n");
		}
		for (int i = first; i < ps.size(); i++) {
			CParam p = ps.get(i);
			if (p.kind() == Kind.STRUCT && p.byValue())
				c.append('\t').append("gowt_").append(structName(n, p)).append(" v_").append(p.name())
						.append(" = *(").append("const gowt_").append(structName(n, p)).append(" *)").append(p.name()).append(";\n");
		}
		c.append('\t');
		if (!n.ret().isEmpty()) c.append("return ");
		// The pointer cast must parenthesize the pointer: "(cast)p(args)" parses as a cast of
		// p(args) in C.
		c.append("((").append(cret).append("(*)(").append(String.join(", ", ctypes)).append("))");
		c.append(n.vtbl() ? "fn" : "p").append(')').append('(').append(String.join(", ", callArgs)).append(");\n");
		c.append("}\n\n");
	}

	// The Go twin of one native: same signature and struct marshaling as the default build
	// (<unit>_native.go), the call itself a typed C call. A pointer passed to cgo escapes, so
	// the struct buffers are heap-allocated automatically - the default build's
	// syscallEscaping (uintptrescapes) exists only because SyscallN's uintptr args hide them.
	private void appendGoFunc(CgoNative n, StringBuilder go) {
		List<String> ps = new ArrayList<>();
		List<String> cargs = new ArrayList<>();
		for (CParam p : n.params()) ps.add(p.name() + " " + goParamType(n, p));
		if (n.vtbl()) {
			// The wrapper takes the resolved function pointer first and the COM object (its
			// first method argument) second; the remaining Go parameters start at index 2.
			cargs.add("C.uintptr_t(vtblFn(" + n.params().get(1).name() + ", " + n.params().get(0).name() + "))");
			cargs.add("C.int64_t(" + n.params().get(1).name() + ")");
		}
		for (int i = n.vtbl() ? 2 : 0; i < n.params().size(); i++) cargs.add(cArgExpr(n, n.params().get(i)));
		go.append("func ").append(n.binding()).append('(').append(String.join(", ", ps)).append(')');
		if (!n.ret().isEmpty()) go.append(' ').append(n.ret());
		go.append(" {\n");
		for (CParam p : n.params()) {
			if (p.kind() != Kind.STRUCT) continue;
			go.append("\tvar b_").append(p.name()).append(" [").append((p.size() + 7) / 8).append("]uint64\n");
			go.append("\tvar p_").append(p.name()).append(" unsafe.Pointer\n");
			go.append("\tif ").append(p.name()).append(" != nil {\n");
			go.append("\t\tp_").append(p.name()).append(" = unsafe.Pointer(&b_").append(p.name()).append(")\n");
			if (!p.noIn()) go.append("\t\t").append(p.name()).append(".toC(p_").append(p.name()).append(")\n");
			go.append("\t}\n");
		}
		if (n.trace()) go.append("\ttraceDispatch(").append(n.params().get(0).name()).append(")\n");
		String call = "C.gowt_" + n.binding() + "(" + String.join(", ", cargs) + ")";
		if (n.ret().isEmpty()) {
			go.append('\t').append(call).append('\n');
		} else {
			String conv = switch (n.ret()) {
				case "int32" -> "int32(";
				case "int64" -> "int64(";
				case "int16" -> "int16(";
				case "int8" -> "int8(";
				case "uint16" -> "uint16(";
				case "float32" -> "float32(";
				case "float64" -> "float64(";
				default -> null;
			};
			if (n.ret().equals("bool")) {
				go.append("\tr := ").append(call).append(" != 0\n");
			} else if (conv != null) {
				go.append("\tr := ").append(conv).append(call).append(")\n");
			} else {
				throw new RuntimeException("j2go: win32 cgo: unmapped return type " + n.ret());
			}
		}
		for (CParam p : n.params()) {
			if (p.kind() != Kind.STRUCT || p.noOut()) continue;
			go.append("\tif ").append(p.name()).append(" != nil {\n\t\t").append(p.name()).append(".fromC(p_").append(p.name()).append(")\n\t}\n");
		}
		if (!n.ret().isEmpty()) go.append("\treturn r\n");
		go.append("}\n\n");
	}

	private String structName(CgoNative n, CParam p) {
		// STRUCT params are Go pointers (*RECT); the typedef twin drops the star.
		return p.goType().substring(1);
	}

	// The Go type of a wrapper parameter, as in the default build's signature.
	private String goParamType(CgoNative n, CParam p) {
		return p.goType();
	}

	private String cArgExpr(CgoNative n, CParam p) {
		return switch (p.kind()) {
			case I32 -> "C.int32_t(" + p.name() + ")";
			case I64 -> "C.int64_t(" + p.name() + ")";
			// The wrapper parameter is a full-width int (see cType).
			case I16 -> "C.int32_t(" + p.name() + ")";
			case I8 -> "C.int32_t(" + p.name() + ")";
			case U16 -> "C.uint16_t(" + p.name() + ")";
			case BOOL -> "C.int32_t(boolToUintptr(" + p.name() + "))";
			case F32 -> "C.float(" + p.name() + ")";
			case F64 -> "C.double(" + p.name() + ")";
			case SLICE -> "(*C." + p.elemC() + ")(unsafe.Pointer(unsafe.SliceData(" + p.name() + ")))";
			case STRUCT -> "p_" + p.name();
			case DEREF -> "C.uintptr_t(derefPtr(" + p.name() + "))";
		};
	}
}
