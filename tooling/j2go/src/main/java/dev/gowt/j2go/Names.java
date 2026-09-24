package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.Modifier;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

/** Naming: capitalization, overload suffixes, names.properties overrides (see AGENTS contract). */
public class Names {

	// binaryName#method(erasures) -> overrideName, loaded from names.properties.
	private final Map<String, String> overrides = new HashMap<>();

	// (declaringTypeKey#methodName) -> ordered method keys, in declaration source order.
	private final Map<String, List<String>> overloadOrder = new HashMap<>();

	// method key -> its declared Java parameter names, for the overload-suffix rule.
	private final Map<String, List<String>> paramNamesByKey = new HashMap<>();

	/** Registers a computed (not user-configured) override - see TypeModel's cross-family
	 * instance-method collision pass. */
	void addOverride(String erasureKey, String goName) {
		overrides.put(erasureKey, goName);
	}

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

	public static String capitalize(String s) {
		if (s.isEmpty()) return s;
		return Character.toUpperCase(s.charAt(0)) + s.substring(1);
	}

	/** Go base name for a Java method: toString -> String, else Capitalize(name). Shared by a
	 * plain call/declaration and the override cascade, so both land on the same Go name. */
	public static String javaMethodBaseGoName(String javaMethodName) {
		return switch (javaMethodName) {
			case "equals" -> "Equals";
			case "hashCode" -> "HashCode";
			case "toString" -> "String";
			default -> capitalize(javaMethodName);
		};
	}

	public static String decapitalize(String s) {
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

	public static String erasureKey(IMethodBinding m) {
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
	public String goMemberName(IMethodBinding m, String baseName) {
		String key = erasureKey(m);
		String override = overrides.get(key);
		if (override != null) return override;

		IMethodBinding decl = m.getMethodDeclaration();
		String declKey = decl.getDeclaringClass().getErasure().getBinaryName();
		String name = decl.isConstructor() ? "<init>" : decl.getName();
		List<String> order = overloadOrder.getOrDefault(declKey + "#" + name, List.of());
		int idx = order.indexOf(key);
		if (idx <= 0) return withTypeNameGuard(decl, baseName); // first declared, or external -> base name.

		if (!nameBasedSuffixesUnique(order)) {
			// JNIGen natives reuse arg0/arg1/... across overloads that differ only by type.
			return baseName + "Overload" + idx;
		}

		List<String> params = paramNamesByKey.getOrDefault(key, List.of());
		StringBuilder sb = new StringBuilder();
		for (String p : params) {
			sb.append(capitalize(p));
		}
		// No contract rule for a later zero-param overload (nothing to append); fall back to arity.
		if (params.isEmpty()) sb.append(params.size());
		// setBackground(Color color) must not become the natural name of setBackgroundColor(NSColor).
		String name_ = baseName + sb;
		if (!decl.isConstructor() && isNaturalNameOfOther(decl, name_)) name_ = baseName + "With" + sb;
		return withTypeNameGuard(decl, name_);
	}

	// Unsuffixed Go name of a differently-named Java method of the same kind (instance: declared
	// or inherited below Object; static: same class) - from bindings, so it never depends on the
	// translated set.
	private static boolean isNaturalNameOfOther(IMethodBinding decl, String goName) {
		boolean isStatic = Modifier.isStatic(decl.getModifiers());
		for (ITypeBinding t = decl.getDeclaringClass(); t != null && !t.getQualifiedName().equals("java.lang.Object");
				t = isStatic ? null : t.getSuperclass()) {
			for (IMethodBinding o : t.getDeclaredMethods()) {
				if (o.isConstructor() || Modifier.isStatic(o.getModifiers()) != isStatic || o.getName().equals(decl.getName())) continue;
				if (javaMethodBaseGoName(o.getName()).equals(goName)) return true;
			}
		}
		return false;
	}

	// Layout.layout() -> "Layout" is also the embedded field every subclass reaches it through;
	// an instance method named like its own or an ancestor's type gets "Fn".
	private static String withTypeNameGuard(IMethodBinding decl, String goName) {
		if (decl.isConstructor() || Modifier.isStatic(decl.getModifiers())) return goName;
		for (ITypeBinding t = decl.getDeclaringClass(); t != null && !t.getQualifiedName().equals("java.lang.Object");
				t = t.getSuperclass()) {
			if (goTypeName(t.getErasure()).equals(goName)) return goName + "Fn";
		}
		return goName;
	}

	private final Map<String, Boolean> uniqueCache = new HashMap<>();

	// Suffixes for overloads after the first (which always keeps the bare base name and so
	// can't collide) - unique means name-based suffixing is fine, else fall back to position.
	private boolean nameBasedSuffixesUnique(List<String> order) {
		return uniqueCache.computeIfAbsent(String.join(",", order), k -> {
			Set<String> seen = new HashSet<>();
			for (int i = 1; i < order.size(); i++) {
				List<String> params = paramNamesByKey.getOrDefault(order.get(i), List.of());
				StringBuilder sb = new StringBuilder();
				for (String p : params) sb.append(capitalize(p));
				if (params.isEmpty()) sb.append(0);
				if (!seen.add(sb.toString())) return false;
			}
			return true;
		});
	}
}
