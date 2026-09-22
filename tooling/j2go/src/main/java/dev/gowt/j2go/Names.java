package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

/** Naming: capitalization, overload suffixes, names.properties overrides (see AGENTS contract). */
class Names {

	// binaryName#method(erasures) -> overrideName, loaded from names.properties.
	private final Map<String, String> overrides = new HashMap<>();

	// (declaringTypeKey#methodName) -> ordered method keys, in declaration source order.
	private final Map<String, List<String>> overloadOrder = new HashMap<>();

	// method key -> its declared Java parameter names, for the overload-suffix rule.
	private final Map<String, List<String>> paramNamesByKey = new HashMap<>();

	void loadOverrides(Path propsFile) throws IOException {
		if (!Files.exists(propsFile)) return;
		Properties p = new Properties();
		try (var in = Files.newInputStream(propsFile)) {
			p.load(in);
		}
		for (String key : p.stringPropertyNames()) {
			overrides.put(key, p.getProperty(key));
		}
	}

	static String capitalize(String s) {
		if (s.isEmpty()) return s;
		return Character.toUpperCase(s.charAt(0)) + s.substring(1);
	}

	static String decapitalize(String s) {
		if (s.isEmpty()) return s;
		return Character.toLowerCase(s.charAt(0)) + s.substring(1);
	}

	/** Outer.Inner Go TYPE name: underscore joined, per contract. */
	static String goTypeName(ITypeBinding t) {
		ITypeBinding decl = t;
		LinkedList<String> parts = new LinkedList<>();
		while (decl != null) {
			parts.addFirst(decl.getName());
			decl = decl.getDeclaringClass();
		}
		return String.join("_", parts);
	}

	/** Outer.Inner Go function-name prefix: concatenated, no underscore (New<Class>, package statics). */
	static String goFuncPrefix(ITypeBinding t) {
		ITypeBinding decl = t;
		LinkedList<String> parts = new LinkedList<>();
		while (decl != null) {
			parts.addFirst(decl.getName());
			decl = decl.getDeclaringClass();
		}
		return String.join("", parts);
	}

	static String erasureKey(IMethodBinding m) {
		IMethodBinding decl = m.getMethodDeclaration();
		StringBuilder sb = new StringBuilder();
		sb.append(decl.getDeclaringClass().getErasure().getBinaryName()).append('#');
		sb.append(decl.isConstructor() ? "<init>" : decl.getName()).append('(');
		ITypeBinding[] params = decl.getParameterTypes();
		for (int i = 0; i < params.length; i++) {
			if (i > 0) sb.append(',');
			sb.append(params[i].getErasure().getBinaryName());
		}
		sb.append(')');
		return sb.toString();
	}

	/** Registers a declaration (method or constructor) in source order, with its Java parameter names. */
	void registerDeclaration(IMethodBinding m, List<String> paramNames) {
		String declKey = m.getDeclaringClass().getErasure().getBinaryName();
		String name = m.isConstructor() ? "<init>" : m.getName();
		String key = erasureKey(m);
		overloadOrder.computeIfAbsent(declKey + "#" + name, k -> new ArrayList<>()).add(key);
		paramNamesByKey.put(key, paramNames);
	}

	/**
	 * Go name for a resolved method/constructor use: base name for the first-declared overload,
	 * base+CapitalizedParamNames for the others, unless overridden via names.properties.
	 */
	String goMemberName(IMethodBinding m, String baseName) {
		String key = erasureKey(m);
		String override = overrides.get(key);
		if (override != null) return override;

		IMethodBinding decl = m.getMethodDeclaration();
		String declKey = decl.getDeclaringClass().getErasure().getBinaryName();
		String name = decl.isConstructor() ? "<init>" : decl.getName();
		List<String> order = overloadOrder.getOrDefault(declKey + "#" + name, List.of());
		int idx = order.indexOf(key);
		if (idx <= 0) return baseName; // first declared, or not registered (external) -> base name.

		List<String> params = paramNamesByKey.getOrDefault(key, List.of());
		StringBuilder sb = new StringBuilder(baseName);
		for (String p : params) {
			sb.append(capitalize(p));
		}
		// No contract rule for a later zero-param overload (nothing to append); fall back to arity.
		if (params.isEmpty()) sb.append(params.size());
		return sb.toString();
	}
}
