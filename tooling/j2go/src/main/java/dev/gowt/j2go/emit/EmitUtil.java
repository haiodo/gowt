package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

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
		String q = s.replace("\\", "\\\\").replace("\"", "\\\"")
				.replace("\n", "\\n").replace("\t", "\\t").replace("\r", "\\r");
		// Other control characters (\0, \b) would land raw in the Go source; NUL is illegal there.
		StringBuilder sb = new StringBuilder(q.length());
		for (char c : q.toCharArray()) {
			if (c < 0x20 || c == 0x7f) sb.append(String.format("\\u%04x", (int) c));
			else sb.append(c);
		}
		return sb.toString();
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
		return collidesWithTypeName(emitter.model, bare) ? prefixed : emitter.qualify(bare, ci);
	}

	/** Same no-prefix exception for a static method, plus the pre-existing "Fn" collision
	 * fallback (a prefixed static method's name can coincidentally spell a translated type's own). */
	static String staticMethodGoName(Emitter emitter, TypeModel.ClassInfo ci, String bareMember) {
		if (!ci.binaryName.equals(SWT_NO_PREFIX_CLASS)) {
			String prefixed = emitter.qualifiedFuncPrefix(ci) + bareMember;
			return collidesWithTypeName(emitter.model, prefixed) ? prefixed + "Fn" : prefixed;
		}
		return collidesWithTypeName(emitter.model, bareMember) ? emitter.qualifiedFuncPrefix(ci) + bareMember : emitter.qualify(bareMember, ci);
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
				parts.add(n + "Like " + emitter.qualify(ci.likeInterfaceName, ci));
				preludeOut.add("var " + n + " *" + emitter.qualifiedTypeName(ci));
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

	/** A local Java only assigns: Go rejects it as unused. */
	static boolean neverRead(VariableDeclarationFragment f, ASTNode scope) {
		IVariableBinding vb = f.resolveBinding();
		if (vb == null) return false;
		boolean[] read = {false};
		scope.accept(new ASTVisitor() {
			@Override
			public boolean visit(SimpleName n) {
				if (n == f.getName() || !vb.isEqualTo(n.resolveBinding())) return false;
				boolean assigned = n.getParent() instanceof Assignment a && a.getLeftHandSide() == n
						&& a.getOperator() == Assignment.Operator.ASSIGN;
				read[0] |= !assigned;
				return false;
			}
		});
		return !read[0];
	}

	/** func init() running each deferred static initializer under its own recover(). */
	static String deferredInitFunc(List<String> inits, List<String> labels) {
		StringBuilder out = new StringBuilder("func init() {\n");
		// Some deferred fields (e.g. kUTType*) resolve only inside SWT's own native lib,
		// which this port lacks - recover per-entry so one bad symbol doesn't sink the rest.
		for (int i = 0; i < inits.size(); i++) {
			out.append("\tfunc() {\n\t\tdefer func() {\n\t\t\tif r := recover(); r != nil {\n")
					.append("\t\t\t\tfmt.Fprintln(os.Stderr, \"gowt/internal/cocoa: deferred init ")
					.append(labels.get(i)).append(":\", r)\n")
					.append("\t\t\t}\n\t\t}()\n").append(inits.get(i)).append("\t}()\n");
		}
		return out.append("}\n\n").toString();
	}

	/** Java null as Go text. A test's literal null String argument or initializer becomes jrt.NullString,
	 * which the ported null-argument guards check for (NumericEmitter.stringParamNullCheck); elsewhere
	 * String null stays "" via adaptNumeric. */
	static String nullLiteral(Emitter emitter, Expression e) {
		if (!emitter.degradesUnresolvedTypes()) return "nil";
		ASTNode child = e;
		ASTNode p = e.getParent();
		while (p instanceof CastExpression || p instanceof ParenthesizedExpression) {
			child = p;
			p = p.getParent();
		}
		ITypeBinding target = null;
		if (p instanceof VariableDeclarationFragment f && f.getInitializer() == child && f.resolveBinding() != null) {
			target = f.resolveBinding().getType();
		} else if (p instanceof MethodInvocation mi && mi.resolveMethodBinding() != null) {
			target = paramType(mi.resolveMethodBinding(), mi.arguments().indexOf(child));
		} else if (p instanceof ClassInstanceCreation cic && cic.resolveConstructorBinding() != null) {
			target = paramType(cic.resolveConstructorBinding(), cic.arguments().indexOf(child));
		}
		if (target == null || !target.getQualifiedName().equals("java.lang.String")) return "nil";
		emitter.fileImports.add(dev.gowt.j2go.Manual.JRT_IMPORT);
		return "jrt.NullString";
	}

	private static ITypeBinding paramType(IMethodBinding mb, int i) {
		ITypeBinding[] pt = mb.getParameterTypes();
		return i >= 0 && i < pt.length && !(mb.isVarargs() && i == pt.length - 1) ? pt[i] : null;
	}

	/** A right operand of &&/|| runs only when reached: prelude lines it needs (an inline
	 * assignment) go inside a closure, not before the whole condition. */
	static String lazyOperand(Emitter emitter, Expression e, String goOp) {
		if (!goOp.equals("&&") && !goOp.equals("||")) return emitter.expr(e);
		List<String> saved = emitter.prelude;
		emitter.prelude = new java.util.ArrayList<>();
		String text = emitter.expr(e);
		List<String> own = emitter.prelude;
		emitter.prelude = saved;
		return own.isEmpty() ? text : "func() bool { " + String.join("; ", own) + "; return " + text + " }()";
	}

	/** `x == null` on an Object (Go any): a typed nil pointer inside the interface is null too,
	 * `== nil` misses it. Null when ie is not such a comparison. */
	static String anyNullCompare(Emitter emitter, InfixExpression ie, String left, String right) {
		InfixExpression.Operator op = ie.getOperator();
		if ((op != InfixExpression.Operator.EQUALS && op != InfixExpression.Operator.NOT_EQUALS) || !ie.extendedOperands().isEmpty()
				|| !(right.equals("nil") ^ left.equals("nil"))) return null;
		ITypeBinding t = (right.equals("nil") ? ie.getLeftOperand() : ie.getRightOperand()).resolveTypeBinding();
		if (t == null || t.isNullType() || !dev.gowt.j2go.GoTypes.map(t, emitter).equals("any")) return null;
		emitter.fileImports.add("github.com/haiodo/gowt/internal/jrt");
		return (op == InfixExpression.Operator.NOT_EQUALS ? "!" : "") + "jrt.IsNil(" + (right.equals("nil") ? left : right) + ")";
	}
}
