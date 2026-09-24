package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.MethodDeclaration;
import org.eclipse.jdt.core.dom.Modifier;
import org.eclipse.jdt.core.dom.SingleVariableDeclaration;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

/** Stateless text/binding helpers shared across the emit components - takes an Emitter param
 * explicitly where needed rather than holding one, so this stays free of per-file state. */
final class EmitUtil {
	private EmitUtil() {}

	// The one class whose static fields/methods drop the class-name prefix (README "Round 7
	// api") - SWT.java is a namespace of constants and utility methods, not a real object.
	static final String SWT_NO_PREFIX_CLASS = "org.eclipse.swt.SWT";

	static String ind(int n) {
		return "\t".repeat(n);
	}

	static String goStringLiteral(String s) {
		return "\"" + escapeGoQuoted(s) + "\"";
	}

	/** A decoded Java string literal can contain a raw newline/tab (from a source "\n"/"\t"
	 * escape) - Go's interpreted string literal needs those escaped too, not just \ and ". */
	static String escapeGoQuoted(String s) {
		return s.replace("\\", "\\\\").replace("\"", "\\\"")
				.replace("\n", "\\n").replace("\t", "\\t").replace("\r", "\\r");
	}

	static String escapeForFormat(String s) {
		return escapeGoQuoted(s).replace("%", "%%");
	}

	static String stripNumericSuffix(String token) {
		if (token.isEmpty()) return token;
		char last = token.charAt(token.length() - 1);
		// f/F/d/D are hex digits in 0x literals (0xFF): only the long suffix applies there.
		boolean hex = token.startsWith("0x") || token.startsWith("0X");
		if (hex) return last == 'l' || last == 'L' ? token.substring(0, token.length() - 1) : token;
		if (last == 'f' || last == 'F' || last == 'd' || last == 'D' || last == 'l' || last == 'L') {
			return token.substring(0, token.length() - 1);
		}
		return token;
	}

	/** Java allows a class to declare a static field and a static method with the same name. */
	static boolean staticFieldClashesWithMethod(ITypeBinding declaringType, String javaFieldName) {
		for (IMethodBinding m : declaringType.getDeclaredMethods()) {
			if (Modifier.isStatic(m.getModifiers()) && m.getName().equals(javaFieldName)) return true;
		}
		return false;
	}

	/** Whether mb implements an interface method - that signature is a fixed Go interface
	 * contract, so it keeps concrete *C params ("Round 7 api" only widens plain methods). */
	static boolean implementsInterfaceMethod(IMethodBinding mb) {
		for (ITypeBinding t = mb.getDeclaringClass(); t != null; t = t.getSuperclass()) {
			java.util.Deque<ITypeBinding> stack = new java.util.ArrayDeque<>(java.util.List.of(t.getInterfaces()));
			while (!stack.isEmpty()) {
				ITypeBinding iface = stack.pop();
				stack.addAll(java.util.List.of(iface.getInterfaces()));
				for (IMethodBinding im : iface.getDeclaredMethods()) {
					if (mb.overrides(im)) return true;
				}
			}
		}
		return false;
	}

	static boolean collidesWithTypeName(TypeModel model, String name) {
		for (TypeModel.ClassInfo c : model.all()) {
			if (c.goTypeName.equals(name)) return true;
		}
		return dev.gowt.j2go.Manual.ownPackageTypeNames().contains(name);
	}

	/** "<Class>Xxx", except SWT_NO_PREFIX_CLASS's bare "Xxx" (falls back to prefixed on a type-
	 * name collision). Shared by the declaration site and every reference site so they agree. */
	static String staticFieldGoName(Emitter emitter, TypeModel.ClassInfo ci, String javaName) {
		String bare = Names.capitalize(javaName);
		String prefixed = emitter.qualifiedFuncPrefix(ci) + bare;
		if (!ci.binaryName.equals(SWT_NO_PREFIX_CLASS)) return prefixed;
		return collidesWithTypeName(emitter.model, bare) ? prefixed : bare;
	}

	/** Same no-prefix exception for a static method, plus the pre-existing "Fn" collision
	 * fallback (a prefixed static method's name can coincidentally spell a translated type's own). */
	static String staticMethodGoName(Emitter emitter, TypeModel.ClassInfo ci, String bareMember) {
		if (!ci.binaryName.equals(SWT_NO_PREFIX_CLASS)) {
			String prefixed = emitter.qualifiedFuncPrefix(ci) + bareMember;
			return collidesWithTypeName(emitter.model, prefixed) ? prefixed + "Fn" : prefixed;
		}
		return collidesWithTypeName(emitter.model, bareMember) ? emitter.qualifiedFuncPrefix(ci) + bareMember : bareMember;
	}

	/** A translated-class parameter widens to "<Class>Like"; preludeOut collects the entry-
	 * conversion lines to emit after the opening brace (README "Round 7 api"). */
	static String publicParamList(Emitter emitter, IMethodBinding mb, MethodDeclaration mdOrNull, List<String> preludeOut) {
		ITypeBinding[] types = mb.getParameterTypes();
		List<String> javaNames = new ArrayList<>();
		if (mdOrNull != null) {
			for (Object p : mdOrNull.parameters()) javaNames.add(((SingleVariableDeclaration) p).getName().getIdentifier());
		}
		List<String> parts = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			String n = i < javaNames.size() ? emitter.sanitizeIdent(javaNames.get(i)) : "a" + i;
			TypeModel.ClassInfo ci = emitter.model.lookup(types[i]);
			if (ci != null && ci.likeInterfaceName != null) {
				parts.add(n + "Like " + ci.likeInterfaceName);
				preludeOut.add("var " + n + " *" + ci.goTypeName);
				preludeOut.add("if " + n + "Like != nil { " + n + " = " + n + "Like." + ci.asMethodName + "() }");
				preludeOut.add("_ = " + n); // a Java body that never reads this param still compiles
			} else {
				parts.add(n + " " + dev.gowt.j2go.GoTypes.map(types[i], emitter));
			}
		}
		return String.join(", ", parts);
	}

	static final Set<String> GO_KEYWORDS = Set.of(
			"break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct",
			"chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range", "type",
			"continue", "for", "import", "return", "var");

	// A field is always selector-qualified, so only a Go keyword (GC.GCTextData.range) needs renaming.
	static String fieldIdent(String javaName) {
		return GO_KEYWORDS.contains(javaName) ? javaName + "_" : javaName;
	}

	static final String OUTER_FIELD = "this_0";

	static boolean isInnerClass(ITypeBinding t) {
		return t.isMember() && t.isClass() && !Modifier.isStatic(t.getModifiers()) && !t.getDeclaringClass().isInterface();
	}
}
