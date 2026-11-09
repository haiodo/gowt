package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/** Native methods: purego lazy binding, constant accessors, _sizeof, _stret, unsupported natives. */
final class NativeEmitter {

	private final Emitter emitter;
	private final WinNativeEmitter win;

	NativeEmitter(Emitter emitter) {
		this.emitter = emitter;
		this.win = new WinNativeEmitter(emitter);
	}

	/** static native OS.<Type>_sizeof(): a Go struct's size is already known at compile time. */
	private TypeModel.ClassInfo sizeofStructTarget(String javaName, IMethodBinding mb) {
		if (!javaName.endsWith("_sizeof") || mb.getParameterTypes().length != 0) return null;
		String typeName = javaName.substring(0, javaName.length() - "_sizeof".length());
		for (TypeModel.ClassInfo ci : emitter.model.all()) {
			if (ci.isStruct && ci.binding.getName().equals(typeName)) return ci;
		}
		return null;
	}

	// Every static native binds its C symbol lazily on first call (one lazyFunc[T] per native,
	// see the manual helper in the PI package), so a missing/wrong symbol panics when that
	// native is actually invoked, not at package init.
	void emitStaticNativeMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		if (dev.gowt.j2go.GoTypes.platform == dev.gowt.j2go.Platform.WIN32) {
			win.emit(md, ci, out);
			return;
		}
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String goName = ci.goFuncPrefix + emitter.names.goMemberName(mb, Names.capitalize(javaName));

		if (emitter.natives.isManual(javaName)) return;
		TypeModel.ClassInfo sizeofTarget = sizeofStructTarget(javaName, mb);
		if (sizeofTarget != null) {
			emitter.fileImports.add("unsafe");
			out.append("func ").append(goName).append("() int32 { return int32(unsafe.Sizeof(")
					.append(sizeofTarget.goTypeName).append("{})) }\n\n");
			return;
		}
		// C.PTR_sizeof(): SWT's own native (sizeof(void*)), not a real libSystem/framework symbol.
		if (javaName.equals("PTR_sizeof") && mb.getParameterTypes().length == 0) {
			emitter.fileImports.add("unsafe");
			out.append("func ").append(goName).append("() int32 { return int32(unsafe.Sizeof(uintptr(0))) }\n\n");
			return;
		}
		// A zero-arg native shadowed by a same-named field (kUTTypeFileURL(), ~165 of these) is
		// a constant-global accessor, not a real function - calling it can SIGBUS (see README).
		// flags=const, or Apple's kName convention (os_custom.c's no_gen kTISPropertyUnicodeKeyLayoutData).
		boolean isConst = !emitter.natives.isFunction(javaName) && (TypeModel.javadocTag(md, "@method", null).contains("flags=const")
				|| javaName.matches("k[A-Z].*"));
		if (mb.getParameterTypes().length == 0 && (isConst || isShadowedByField(ci.binding, javaName))) {
			emitConstantAccessor(md, mb, javaName, goName, out);
			return;
		}
		String callbackTypes = TypeModel.javadocTag(md, "@method", null);
		if (callbackTypes.contains("callback_types=")) {
			emitStructCallback(mb, goName, callbackTypes, out);
			return;
		}
		if (javaName.endsWith("_stret") && emitter.retType(mb).isEmpty() && mb.getParameterTypes().length > 0) {
			TypeModel.ClassInfo resultStruct = structParamTarget(mb.getParameterTypes()[0]);
			if (resultStruct != null) {
				emitStretNative(md, mb, javaName, goName, resultStruct, out);
				return;
			}
		}
		emitLazyNative(md, mb, javaName, goName, emitter.retType(mb), out);
	}

	TypeModel.ClassInfo structParamTarget(ITypeBinding t) {
		TypeModel.ClassInfo target = emitter.model.lookup(t);
		return target != null && target.isStruct ? target : null;
	}

	// arm64 has one objc_msgSend entry point regardless of return shape: bind without the _stret
	// suffix, typed to return the struct, and write it through result (now a pointer, see README).
	private void emitStretNative(MethodDeclaration md, IMethodBinding mb, String javaName, String goName,
			TypeModel.ClassInfo resultStruct, StringBuilder out) {
		ITypeBinding[] types = mb.getParameterTypes();
		List<?> mdParams = md.parameters();
		List<String> restParams = new ArrayList<>();
		List<String> restArgs = new ArrayList<>();
		List<String> restNames = new ArrayList<>();
		List<String> restTypes = new ArrayList<>();
		for (int i = 1; i < types.length; i++) {
			String n = i < mdParams.size() ? emitter.sanitizeIdent(((SingleVariableDeclaration) mdParams.get(i)).getName().getIdentifier()) : "a" + i;
			restParams.add(n + " " + paramType(mb, i));
			restArgs.add(n);
			restNames.add(n);
			restTypes.add(paramType(mb, i));
		}
		String resultParamName = mdParams.isEmpty() ? "result" : emitter.sanitizeIdent(((SingleVariableDeclaration) mdParams.get(0)).getName().getIdentifier());
		String symbol = emitter.natives.symbolFor(stripSuffix(javaName, "_stret"));
		// One lazyFunc[T] per native (manual helper in the PI package) instead of a per-native
		// _impl/_once pair; sync/purego imports live in the helper.
		out.append("var ").append(goName).append("_lazy lazyFunc[func(").append(String.join(", ", restParams)).append(") ")
				.append(resultStruct.goTypeName).append("]\n");
		out.append("func ").append(goName).append('(').append(resultParamName).append(" *").append(resultStruct.goTypeName);
		if (!restParams.isEmpty()) out.append(", ").append(String.join(", ", restParams));
		out.append(") {\n\t*").append(resultParamName).append(" = ").append(goName).append("_lazy.get(\"").append(goName)
				.append("\", \"").append(symbol).append("\")(").append(String.join(", ", restArgs)).append(")\n}\n\n");
		registerCgo(goName, symbol, restNames, restTypes, resultStruct.goTypeName);
	}

	private String stripSuffix(String s, String suffix) {
		return s.substring(0, s.length() - suffix.length());
	}

	/** A struct param JNI passes by pointer (no `flags=struct`, see TypeModel) is a Go pointer. */
	private String paramType(IMethodBinding mb, int i) {
		String t = dev.gowt.j2go.GoTypes.map(mb.getParameterTypes()[i], emitter);
		return emitter.model.isNativeStructPointerParam(mb, i) ? "*" + t : t;
	}

	private void emitLazyNative(MethodDeclaration md, IMethodBinding mb, String javaName, String goName, String ret, StringBuilder out) {
		String symbol = emitter.natives.symbolFor(javaName);
		ITypeBinding[] types = mb.getParameterTypes();
		// purego's darwin/arm64 stack packing copies a slice arg's whole header (ptr, len, cap)
		// instead of its pointer, so past 8 integer args an array goes as *elem.
		int intArgs = 0;
		for (ITypeBinding t : types) if (!t.getName().equals("double") && !t.getName().equals("float")) intArgs++;
		List<String> ps = new ArrayList<>();
		List<String> backingPs = new ArrayList<>();
		List<String> backingTypes = new ArrayList<>();
		List<String> args = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			String n = emitter.sanitizeIdent(((SingleVariableDeclaration) md.parameters().get(i)).getName().getIdentifier());
			String t = paramType(mb, i);
			ps.add(n + " " + t);
			boolean byPointer = intArgs > 8 && types[i].isArray();
			String backing = byPointer ? "*" + t.substring(2) : t;
			backingPs.add(n + " " + backing);
			backingTypes.add(backing);
			args.add(byPointer ? "unsafe.SliceData(" + n + ")" : n);
			if (byPointer) emitter.fileImports.add("unsafe");
		}
		String params = String.join(", ", ps);
		out.append("var ").append(goName).append("_lazy lazyFunc[func(").append(String.join(", ", backingPs)).append(") ").append(ret).append("]\n");
		out.append("func ").append(goName).append('(').append(params).append(") ").append(ret).append(" {\n\t")
				.append(ret.isEmpty() ? "" : "return ").append(goName).append("_lazy.get(\"").append(goName)
				.append("\", \"").append(symbol).append("\")(").append(String.join(", ", args)).append(")\n}\n\n");
		registerCgo(goName, symbol, paramNames(md), backingTypes, ret);
	}

	// The generated wrapper's own parameter names, parallel to the backingTypes passed to registerCgo.
	private List<String> paramNames(MethodDeclaration md) {
		List<String> names = new ArrayList<>();
		for (Object p : md.parameters()) names.add(emitter.sanitizeIdent(((SingleVariableDeclaration) p).getName().getIdentifier()));
		return names;
	}

	// A constant-global accessor (see caller): Dlsym the address and dereference it directly,
	// instead of calling it as code - 0 if missing, SWT itself treats 0 as "not present".
	private void emitConstantAccessor(MethodDeclaration md, IMethodBinding mb, String javaName, String goName, StringBuilder out) {
		String ret = emitter.retType(mb);
		String symbol = emitter.natives.symbolFor(javaName);
		// unsafe.Add, not unsafe.Pointer(addr) directly: go vet's unsafeptr check flags a bare
		// uintptr->Pointer conversion, unsafe.Add is the vet-clean spelling of the same thing.
		String deref = switch (ret) {
			case "float64" -> "*(*float64)(unsafe.Add(unsafe.Pointer(nil), addr))";
			case "float32" -> "float32(*(*float64)(unsafe.Add(unsafe.Pointer(nil), addr)))";
			default -> ret + "(*(*int64)(unsafe.Add(unsafe.Pointer(nil), addr)))";
		};
		emitter.fileImports.add("unsafe");
		emitter.fileImports.add("github.com/ebitengine/purego");
		out.append("func ").append(goName).append("() ").append(ret).append(" {\n")
				.append("\taddr, _ := purego.Dlsym(purego.RTLD_DEFAULT, \"").append(symbol).append("\")\n")
				.append("\tif addr == 0 {\n\t\treturn 0\n\t}\n")
				.append("\treturn ").append(deref).append('\n')
				.append("}\n\n");
	}

	private static final java.util.Set<String> CALLBACK_STRUCTS = java.util.Set.of("NSRect", "NSPoint", "NSSize", "NSRange");

	/** CALLBACK_x(func): os.c's by-value-struct IMP trampoline, generated from the Javadoc's
	 * callback_types/flags (README "Callback design"). */
	private void emitStructCallback(IMethodBinding mb, String goName, String javadoc, StringBuilder out) {
		String[] types = field(javadoc, "callback_types=").split(";");
		String[] flags = field(javadoc, "callback_flags=").split(";");
		List<String> params = new ArrayList<>();
		List<String> args = new ArrayList<>();
		boolean pins = false;
		for (int i = 1; i < types.length; i++) {
			String st = structType(types[i], flags[i]);
			params.add("arg" + (i - 1) + " " + (st != null ? st : "uintptr"));
			args.add(st != null ? "pinArg(&pin, &arg" + (i - 1) + ")" : "int64(arg" + (i - 1) + ")");
			pins |= st != null;
		}
		String retStruct = structType(types[0], flags[0]);
		String call = "fn([]int64{" + String.join(", ", args) + "})";
		out.append("func ").append(goName).append('(').append(emitter.paramList(mb, null)).append(") int64 {\n");
		out.append("\tfn := callbackFunc(a0)\n");
		out.append("\treturn int64(NewCallback(func(").append(String.join(", ", params)).append(") ")
				.append(retStruct != null ? retStruct : "uintptr").append(" {\n");
		if (pins) {
			emitter.fileImports.add("runtime");
			out.append("\t\tvar pin runtime.Pinner\n\t\tdefer pin.Unpin()\n");
		}
		out.append("\t\treturn ").append(retStruct != null ? "structResult[" + retStruct + "](" + call + ")" : "uintptr(" + call + ")")
				.append("\n\t}))\n}\n\n");
	}

	private static String field(String javadoc, String key) {
		String rest = javadoc.substring(javadoc.indexOf(key) + key.length()).trim();
		int end = rest.indexOf(',');
		return (end < 0 ? rest : rest.substring(0, end)).trim();
	}

	private static String structType(String type, String flag) {
		return flag.trim().equals("struct") && CALLBACK_STRUCTS.contains(type.trim()) ? type.trim() : null;
	}

	/** Native with no verified purego binding (see natives.properties): unsupported-marker stub. */
	void emitUnsupportedNative(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String goName = ci.goFuncPrefix + emitter.names.goMemberName(mb, Names.capitalize(javaName));
		emitter.unsupported.add("NativeMethod: " + ci.binaryName + "." + javaName);
		out.append("func ").append(goName).append('(').append(emitter.paramList(mb, md)).append(") ").append(emitter.retType(mb))
				.append(" {\n\tpanic(\"j2go: unsupported native ").append(javaName).append("\")\n}\n\n");
	}

	/** The reverse of EmitUtil.staticFieldClashesWithMethod: a field with this exact name also exists. */
	private boolean isShadowedByField(ITypeBinding declaringType, String javaMethodName) {
		for (IVariableBinding f : declaringType.getDeclaredFields()) {
			if (f.getName().equals(javaMethodName)) return true;
		}
		return false;
	}

	// ---- Opt-in cgo backend (darwin, build tag gowtcgo): the generated lazy wrappers call
	// get("<name>", "<symbol>"); the default build ignores name and binds symbol via purego,
	// the gowtcgo build ignores symbol and resolves name in the generated registry of typed
	// C wrappers (cgoNativeRegistry, see lazyfunccgo_manual.go in the PI package). Entries are
	// collected per compilation unit and written as <unit>_cgo.go by Main.

	record CgoNative(String binding, String symbol, List<String> paramNames, List<String> paramTypes, String ret) {}

	private final List<CgoNative> cgoNatives = new ArrayList<>();

	void clearCgoNatives() {
		cgoNatives.clear();
		win.clear();
	}

	// ---- Opt-in cgo backend for win32 (build tag windows && gowtcgo && cgo), parallel to the
	// darwin one above: the default backend's SyscallN wrappers go to <unit>_native.go, the
	// typed C wrappers to <unit>_cgo.go; Main writes both next to the unit's own files.

	boolean winHasNatives() {
		return win.hasNatives();
	}

	java.util.Set<String> winNativeImports() {
		return win.nativeImports();
	}

	String winNativeFileBody() {
		return win.nativeFileBody();
	}

	String winCgoFileText(String goPackage) {
		return win.cgoFileText(goPackage);
	}

	private void registerCgo(String binding, String symbol, List<String> names, List<String> types, String ret) {
		if (dev.gowt.j2go.GoTypes.platform != dev.gowt.j2go.Platform.COCOA) return;
		cgoNatives.add(new CgoNative(binding, symbol, names, types, ret));
	}

	// Go scalar -> C spelling (inside the C preamble and of the gowt_ wrapper params).
	private static final java.util.Map<String, String> C_SCALARS = java.util.Map.of(
			"int8", "int8_t", "int16", "int16_t", "int32", "int32_t", "int64", "int64_t",
			"uint16", "uint16_t", "float32", "float", "float64", "double");

	private static String cScalar(String goType) {
		return C_SCALARS.get(goType);
	}

	// The generated <unit>_cgo.go file for this unit's natives, or null when it has none.
	String cgoFileText(String goPackage) {
		if (cgoNatives.isEmpty()) return null;
		boolean usesUnsafe = false;
		for (CgoNative n : cgoNatives) {
			for (String t : n.paramTypes()) {
				usesUnsafe |= t.startsWith("[]") || t.startsWith("*") || t.equals("string");
			}
		}
		StringBuilder c = new StringBuilder();
		c.append("#include <stdint.h>\n");
		c.append("#include <stdio.h>\n");
		c.append("#include <stdlib.h>\n");
		c.append("#include <dlfcn.h>\n\n");
		appendCStructs(c);
		for (CgoNative n : cgoNatives) appendCWrapper(n, c);
		StringBuilder go = new StringBuilder();
		for (CgoNative n : cgoNatives) appendGoFunc(n, go);
		// Package-level registration, not func init(): variable initializers run before any
		// init() regardless of file order, and other files' init()s (cocoa_os.go's deferred
		// static fields) call natives - the registry must be populated by then.
		go.append("var _ = func() int {\n");
		for (CgoNative n : cgoNatives) go.append("\tcgoNativeRegistry[\"").append(n.binding()).append("\"] = cgo_")
				.append(n.binding()).append('\n');
		go.append("\treturn 0\n}()\n");
		StringBuilder f = new StringBuilder();
		f.append("// Code generated by j2go. DO NOT EDIT.\n");
		f.append("//go:build darwin && gowtcgo && cgo\n\n");
		f.append("package ").append(goPackage).append("\n\n");
		f.append("/*\n").append(c).append("*/\n");
		f.append("import \"C\"\n\n");
		if (usesUnsafe) f.append("import \"unsafe\"\n\n");
		f.append(go);
		return f.toString();
	}

	// Struct value types used by this unit's natives, as C typedefs mirroring the Go structs
	// field-for-field (same names and layout - cgo converts between them by value). Emitted
	// dependency-first: a struct whose field is another struct waits for that one's typedef.
	private void appendCStructs(StringBuilder c) {
		Set<String> used = new LinkedHashSet<>();
		for (CgoNative n : cgoNatives) {
			for (String t : n.paramTypes()) if (!t.equals("bool") && !t.equals("string") && !t.startsWith("[]") && cScalar(t) == null) collectStruct(t, used);
			String r = n.ret();
			if (!r.isEmpty() && cScalar(r) == null && !r.equals("bool") && !r.equals("string")) used.add(r);
		}
		// A field's struct type needs its own typedef even when no native takes it directly.
		Set<String> more = new LinkedHashSet<>();
		for (String s : used) collectFieldStructs(s, more);
		used.addAll(more);
		Set<String> emitted = new LinkedHashSet<>();
		boolean progress = true;
		while (!emitted.containsAll(used) && progress) {
			progress = false;
			for (String s : used) {
				if (emitted.contains(s) || !structDepsEmitted(s, emitted)) continue;
				appendCStruct(s, c);
				emitted.add(s);
				progress = true;
			}
		}
		if (!emitted.containsAll(used)) {
			StringBuilder d = new StringBuilder();
			for (String s : used) {
				if (emitted.contains(s)) continue;
				d.append(' ').append(s).append(":[");
				for (CField field : structFields(structByGoName(s))) d.append(field.goName()).append('=').append(field.goType()).append(' ');
				d.append(']');
			}
			throw new RuntimeException("j2go: cgo registry: cyclic struct layout in " + used + ":" + d);
		}
	}

	private void collectStruct(String t, Set<String> used) {
		String name = t.startsWith("*") ? t.substring(1) : t;
		if (cScalar(name) != null) return;
		used.add(name);
	}

	// A struct's fields in Go emission (AST declaration) order with their Go types - the C
	// typedef must mirror the Go struct exactly for cgo's by-value struct conversion.
	private record CField(String goName, String goType) {}

	private List<CField> structFields(TypeModel.ClassInfo ci) {
		AbstractTypeDeclaration td = emitter.model.typeDeclarationOf(ci.binding);
		if (td == null) throw new RuntimeException("j2go: cgo registry: no declaration for struct " + ci.goTypeName);
		List<CField> fields = new ArrayList<>();
		for (Object o : td.bodyDeclarations()) {
			if (!(o instanceof FieldDeclaration fd) || Modifier.isStatic(fd.getModifiers())) continue;
			for (Object fo : fd.fragments()) {
				VariableDeclarationFragment f = (VariableDeclarationFragment) fo;
				String javaName = f.getName().getIdentifier();
				if (javaName.equals("serialVersionUID")) continue;
				String goName = Modifier.isPublic(fd.getModifiers()) ? Names.capitalize(javaName) : EmitUtil.fieldIdent(javaName);
				ITypeBinding t = f.resolveBinding() != null ? f.resolveBinding().getType() : fd.getType().resolveBinding();
				fields.add(new CField(goName, dev.gowt.j2go.GoTypes.map(t, emitter)));
			}
		}
		return fields;
	}

	private void collectFieldStructs(String goName, Set<String> more) {
		for (CField field : structFields(structByGoName(goName))) {
			if (cScalar(field.goType()) == null && !field.goType().startsWith("[]") && !field.goType().equals("string")) more.add(field.goType());
		}
	}

	private boolean structDepsEmitted(String goName, Set<String> emitted) {
		for (CField field : structFields(structByGoName(goName))) {
			if (cScalar(field.goType()) == null && !emitted.contains(field.goType())) return false;
		}
		return true;
	}

	private void appendCStruct(String goName, StringBuilder c) {
		c.append("typedef struct {\n");
		for (CField field : structFields(structByGoName(goName))) {
			String ct = cScalar(field.goType());
			if (ct == null) ct = "gowt_" + field.goType(); // a nested struct, typedef emitted first
			c.append('\t').append(ct).append(' ').append(field.goName()).append(";\n");
		}
		c.append("} gowt_").append(goName).append(";\n\n");
	}

	private TypeModel.ClassInfo structByGoName(String goTypeName) {
		for (TypeModel.ClassInfo ci : emitter.model.all()) {
			if (ci.isStruct && goTypeName.equals(ci.goTypeName)) return ci;
		}
		throw new RuntimeException("j2go: cgo registry: unknown struct type " + goTypeName);
	}

	// One C wrapper per native: a lazily dlsym'd, exactly typed function pointer through which
	// the native is called directly - no purego/libffi on this path. The static pointer is
	// resolved on first call; the Go side's sync.Once has already run ensureFrameworks, so
	// RTLD_DEFAULT sees every framework.
	private void appendCWrapper(CgoNative n, StringBuilder c) {
		List<String> ctypes = new ArrayList<>();
		List<String> cparams = new ArrayList<>();
		for (int i = 0; i < n.paramTypes().size(); i++) {
			String ct = cParamType(n.paramTypes().get(i));
			ctypes.add(ct);
			cparams.add(ct + " " + n.paramNames().get(i));
		}
		String cret = cRetType(n.ret());
		c.append("static ").append(cret).append(" (*gowt_ptr_").append(n.binding()).append(")(")
				.append(String.join(", ", ctypes)).append(");\n");
		c.append("static ").append(cret).append(" gowt_").append(n.binding()).append("(")
				.append(String.join(", ", cparams)).append(") {\n");
		c.append("\tif (gowt_ptr_").append(n.binding()).append(" == 0) {\n");
		c.append("\t\t*(void **)&gowt_ptr_").append(n.binding()).append(" = dlsym(RTLD_DEFAULT, \"").append(n.symbol()).append("\");\n");
		c.append("\t\tif (gowt_ptr_").append(n.binding()).append(" == 0) {\n");
		c.append("\t\t\tfprintf(stderr, \"gowt: native symbol not found: %s\\n\", \"").append(n.symbol()).append("\");\n");
		c.append("\t\t\tabort();\n\t\t}\n\t}\n");
		c.append('\t');
		if (!n.ret().isEmpty()) c.append("return ");
		c.append("gowt_ptr_").append(n.binding()).append('(').append(String.join(", ", n.paramNames())).append(");\n");
		c.append("}\n\n");
	}

	private String cParamType(String goType) {
		String scalar = cScalar(goType);
		if (scalar != null) return scalar;
		if (goType.equals("bool")) return "uint8_t";
		if (goType.equals("string")) return "char*";
		if (goType.startsWith("[]")) return cScalar(goType.substring(2)) + "*";
		if (goType.startsWith("*")) return cPointeeType(goType.substring(1)) + "*";
		return "gowt_" + goType;
	}

	private String cPointeeType(String goType) {
		String scalar = cScalar(goType);
		return scalar != null ? scalar : "gowt_" + goType;
	}

	private String cRetType(String ret) {
		if (ret.isEmpty()) return "void";
		return cParamType(ret);
	}

	// The registry's Go entry for one native: converts Go args to the C wrapper's types and
	// its result back (struct results and args convert by value, slices to element pointers).
	private void appendGoFunc(CgoNative n, StringBuilder go) {
		List<String> gparams = new ArrayList<>();
		List<String> cargs = new ArrayList<>();
		List<String> stmts = new ArrayList<>();
		for (int i = 0; i < n.paramTypes().size(); i++) {
			String name = n.paramNames().get(i);
			String t = n.paramTypes().get(i);
			gparams.add(name + " " + t);
			cargs.add(cArgExpr(name, t, stmts));
		}
		String ret = n.ret();
		go.append("func cgo_").append(n.binding()).append('(').append(String.join(", ", gparams)).append(')');
		if (!ret.isEmpty()) go.append(' ').append(ret);
		go.append(" {\n");
		for (String s : stmts) go.append('\t').append(s).append('\n');
		String call = "C.gowt_" + n.binding() + "(" + String.join(", ", cargs) + ")";
		go.append('\t');
		if (ret.isEmpty()) {
			go.append(call).append('\n');
		} else if (ret.equals("bool")) {
			go.append("return ").append(call).append(" != 0\n");
		} else if (ret.equals("string")) {
			go.append("return C.GoString(").append(call).append(")\n");
		} else if (cScalar(ret) != null) {
			go.append("return ").append(ret).append('(').append(call).append(")\n");
		} else {
			go.append("return cgoStruct[").append(ret).append("](").append(call).append(")\n");
		}
		go.append("}\n\n");
	}

	private String cArgExpr(String name, String t, List<String> stmts) {
		String scalar = cScalar(t);
		if (scalar != null) return "C." + scalar + "(" + name + ")";
		if (t.equals("bool")) return "cgoBool(" + name + ")";
		if (t.equals("string")) {
			stmts.add("cstr_" + name + " := C.CString(" + name + ")");
			stmts.add("defer C.free(unsafe.Pointer(cstr_" + name + "))");
			return "cstr_" + name;
		}
		if (t.startsWith("[]")) return "(*C." + cScalar(t.substring(2)) + ")(unsafe.Pointer(unsafe.SliceData(" + name + ")))";
		if (t.startsWith("*")) return "(*C." + cPointeeType(t.substring(1)) + ")(unsafe.Pointer(" + name + "))";
		// Struct by value: Go and C structs are layout-identical but not type-identical for
		// the compiler (cgo field types/tags differ), so reinterpret instead of converting.
		return "cgoStruct[C.gowt_" + t + "](" + name + ")";
	}
}
