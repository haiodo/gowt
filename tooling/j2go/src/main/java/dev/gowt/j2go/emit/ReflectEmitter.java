package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

/** Reflect registry (README "Round 10 reflection", "Round 11 null-string and reflect opt-in"):
 * public methods of widgets-package classes, registered from the separate generated package
 * swt/swtreflect so only programs importing it keep them. */
final class ReflectEmitter {

	private static final String WIDGETS_PACKAGE = "org.eclipse.swt.widgets";
	static final String REFLECT_PACKAGE = "swtreflect";

	private final Emitter emitter;
	private final List<String> registrations = new ArrayList<>();
	private final Set<String> imports = new TreeSet<>();

	ReflectEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	/** Registers one public instance method (own declaration, or a split cascade's override-point
	 * wrapper) of a org.eclipse.swt.widgets class for java.lang.Class#getMethod/Method#invoke -
	 * see internal/jrt/reflect.go. Types are spelled as seen from package swtreflect. Skipped for
	 * anything GoTypes couldn't map to a real Go type - simply not reflectable. */
	void registerReflectMethod(TypeModel.ClassInfo ci, IMethodBinding mb, String javaName, String goName, boolean widen) {
		if (!ci.javaPackage.equals(WIDGETS_PACKAGE) || !Modifier.isPublic(mb.getModifiers()) || mb.isVarargs()) return;
		String savedPackage = emitter.currentGoPackage;
		Set<String> savedImports = emitter.fileImports;
		emitter.currentGoPackage = REFLECT_PACKAGE;
		emitter.fileImports = imports;
		try {
			String line = reflectRegistration(ci, mb, javaName, goName, widen);
			if (line != null) registrations.add(line);
		} finally {
			emitter.currentGoPackage = savedPackage;
			emitter.fileImports = savedImports;
		}
	}

	private String reflectRegistration(TypeModel.ClassInfo ci, IMethodBinding mb, String javaName, String goName, boolean widen) {
		String ret = emitter.retType(mb);
		if (ret.contains("unsupported_") || ret.contains("func(")) return null;
		ITypeBinding[] paramTypes = mb.getParameterTypes();
		List<String> paramTypeExprs = new ArrayList<>();
		List<String> callArgs = new ArrayList<>();
		for (int i = 0; i < paramTypes.length; i++) {
			String t = regParamType(paramTypes[i], widen);
			if (t.contains("unsupported_") || t.contains("func(")) return null;
			paramTypeExprs.add("reflect.TypeFor[" + t + "]()");
			callArgs.add("jrt.ArgAs[" + t + "](args[" + i + "])");
		}
		String self = "*" + emitter.qualifiedTypeName(ci);
		String paramTypesLit = paramTypeExprs.isEmpty() ? "nil" : "[]reflect.Type{" + String.join(", ", paramTypeExprs) + "}";
		String returnTypeExpr = ret.isEmpty() ? "nil" : "reflect.TypeFor[" + ret + "]()";
		String call = "jrt.Narrow[" + self + "](target)." + goName + "(" + String.join(", ", callArgs) + ")";
		String body = ret.isEmpty() ? call + "; return nil" : "return " + call;
		return "jrt.RegisterMethod(reflect.TypeFor[" + self + "](), \"" + javaName + "\", "
				+ paramTypesLit + ", " + returnTypeExpr + ", func(target any, args []any) any { " + body + " })";
	}

	/** Same per-parameter widening publicParamList uses (README "Round 7 api"): a translated-class
	 * param widens to its "<Class>Like" interface so the registered closure accepts any subclass. */
	private String regParamType(ITypeBinding t, boolean widen) {
		if (widen) {
			TypeModel.ClassInfo pci = emitter.model.lookup(t);
			if (pci != null && pci.likeInterfaceName != null) return emitter.qualify(pci.likeInterfaceName, pci);
		}
		return dev.gowt.j2go.GoTypes.map(t, emitter);
	}

	/** Package swtreflect's source (without the file header), or null when nothing registered. */
	String registryFile() {
		if (registrations.isEmpty()) return null;
		StringBuilder b = new StringBuilder();
		for (String line : registrations) b.append('\t').append(line).append('\n');
		String init = b.toString();
		imports.add("reflect");
		imports.add(Manual.JRT_IMPORT);
		imports.removeIf(imp -> !init.contains(imp.substring(imp.lastIndexOf('/') + 1) + "."));
		StringBuilder out = new StringBuilder("// Package swtreflect registers the public methods of package swt's widgets for\n"
				+ "// java.lang.reflect-style lookup (internal/jrt/reflect.go). Import it for its side effect.\n"
				+ "package " + REFLECT_PACKAGE + "\n\nimport (\n");
		for (String imp : imports) out.append("\t\"").append(imp).append("\"\n");
		return out.append(")\n\nfunc init() {\n").append(init).append("}\n").toString();
	}
}
