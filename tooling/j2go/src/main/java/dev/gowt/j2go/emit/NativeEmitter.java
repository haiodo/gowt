package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

/** Native methods: purego lazy binding, constant accessors, _sizeof, _stret, unsupported natives. */
final class NativeEmitter {

	private final Emitter emitter;

	NativeEmitter(Emitter emitter) {
		this.emitter = emitter;
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

	// Every static native binds its C symbol lazily on first call (sync.Once + RTLD_DEFAULT),
	// so a missing/wrong symbol panics when that native is actually invoked, not at package init.
	void emitStaticNativeMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String goName = ci.goFuncPrefix + emitter.names.goMemberName(mb, Names.capitalize(javaName));

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
		if (mb.getParameterTypes().length == 0 && isShadowedByField(ci.binding, javaName)) {
			emitConstantAccessor(md, mb, javaName, goName, out);
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
		for (int i = 1; i < types.length; i++) {
			String n = i < mdParams.size() ? emitter.sanitizeIdent(((SingleVariableDeclaration) mdParams.get(i)).getName().getIdentifier()) : "a" + i;
			restParams.add(n + " " + dev.gowt.j2go.GoTypes.map(types[i], emitter));
			restArgs.add(n);
		}
		String resultParamName = mdParams.isEmpty() ? "result" : emitter.sanitizeIdent(((SingleVariableDeclaration) mdParams.get(0)).getName().getIdentifier());
		String symbol = emitter.natives.symbolFor(stripSuffix(javaName, "_stret"));
		String backing = goName + "_impl";
		String once = goName + "_once";
		emitter.fileImports.add("sync");
		emitter.fileImports.add("github.com/ebitengine/purego");
		out.append("var ").append(backing).append(" func(").append(String.join(", ", restParams)).append(") ")
				.append(resultStruct.goTypeName).append('\n');
		out.append("var ").append(once).append(" sync.Once\n");
		out.append("func ").append(goName).append('(').append(resultParamName).append(" *").append(resultStruct.goTypeName);
		if (!restParams.isEmpty()) out.append(", ").append(String.join(", ", restParams));
		out.append(") {\n\t").append(once).append(".Do(func() { ensureFrameworks(); purego.RegisterLibFunc(&").append(backing)
				.append(", purego.RTLD_DEFAULT, \"").append(symbol).append("\") })\n");
		out.append("\t*").append(resultParamName).append(" = ").append(backing).append('(')
				.append(String.join(", ", restArgs)).append(")\n}\n\n");
	}

	private String stripSuffix(String s, String suffix) {
		return s.substring(0, s.length() - suffix.length());
	}

	private void emitLazyNative(MethodDeclaration md, IMethodBinding mb, String javaName, String goName, String ret, StringBuilder out) {
		String symbol = emitter.natives.symbolFor(javaName);
		String params = emitter.paramList(mb, md);
		String args = emitter.argNames(md);
		String backing = goName + "_impl";
		String once = goName + "_once";
		emitter.fileImports.add("sync");
		emitter.fileImports.add("github.com/ebitengine/purego");
		out.append("var ").append(backing).append(" func(").append(params).append(") ").append(ret).append('\n');
		out.append("var ").append(once).append(" sync.Once\n");
		out.append("func ").append(goName).append('(').append(params).append(") ").append(ret).append(" {\n\t")
				.append(once).append(".Do(func() { ensureFrameworks(); purego.RegisterLibFunc(&").append(backing)
				.append(", purego.RTLD_DEFAULT, \"").append(symbol).append("\") })\n\t")
				.append(ret.isEmpty() ? "" : "return ").append(backing).append('(').append(args).append(")\n}\n\n");
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
}
