package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.Modifier;

import java.util.Set;

/** Stateless text/binding helpers shared across the emit components - no Emitter state. */
final class EmitUtil {
	private EmitUtil() {}

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
