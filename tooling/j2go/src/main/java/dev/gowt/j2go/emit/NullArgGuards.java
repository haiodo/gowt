package dev.gowt.j2go.emit;

import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.HashSet;
import java.util.List;
import java.util.Set;

/** Does a callee treat a String parameter as a null-argument guard (see nullGuardedParam)?
 * Only then does a test's literal null become jrt.NullString; anywhere else it stays "". */
final class NullArgGuards {
	private NullArgGuards() {}

	/** True if any declaration of mb (or an override) guards parameter idx, directly or by
	 * passing it unchanged to a callee that does. */
	static boolean guards(TypeModel model, IMethodBinding mb, int idx) {
		return guards(model, mb, idx, new HashSet<>());
	}

	/** A local String initialized to null: sentinel only if it is later passed straight to a guarded parameter. */
	static boolean localReachesGuard(TypeModel model, VariableDeclarationFragment f) {
		return localReachesGuard(model, f.resolveBinding(), f);
	}

	/** Same for the String[] local that `arr[i] = null` fills (the callee's guard is on an element). */
	static boolean localReachesGuard(TypeModel model, IVariableBinding vb, ASTNode from) {
		ASTNode scope = from;
		while (scope != null && !(scope instanceof MethodDeclaration) && !(scope instanceof Initializer)) scope = scope.getParent();
		if (vb == null || scope == null) return false;
		boolean[] hit = {false};
		scope.accept(new ASTVisitor() {
			@Override
			public boolean visit(SimpleName n) {
				if (!hit[0] && n.resolveBinding() instanceof IVariableBinding b && b.isEqualTo(vb)) {
					hit[0] = passedToGuard(model, n, new HashSet<>());
				}
				return false;
			}
		});
		return hit[0];
	}

	private static boolean guards(TypeModel model, IMethodBinding mb, int idx, Set<String> seen) {
		if (!seen.add(mb.getKey() + "#" + idx)) return false;
		for (MethodDeclaration md : model.relatedDeclarations(mb)) {
			if (md.getBody() == null || idx >= md.parameters().size()) continue;
			IVariableBinding param = ((SingleVariableDeclaration) md.parameters().get(idx)).resolveBinding();
			if (param == null) continue;
			boolean[] hit = {false};
			md.getBody().accept(new ASTVisitor() {
				@Override
				public boolean visit(InfixExpression ie) {
					IVariableBinding g = nullGuardedParam(ie);
					if (g == null) g = elementGuardArray(ie);
					if (g != null && g.isEqualTo(param)) hit[0] = true;
					return !hit[0];
				}

				@Override
				public boolean visit(SimpleName n) {
					if (!hit[0] && n.resolveBinding() instanceof IVariableBinding b && b.isEqualTo(param)) {
						hit[0] = passedToGuard(model, n, seen);
					}
					return false;
				}
			});
			if (hit[0]) return true;
		}
		return false;
	}

	// n is a bare argument of a call/constructor whose matching parameter is guarded.
	private static boolean passedToGuard(TypeModel model, SimpleName n, Set<String> seen) {
		ASTNode p = n.getParent();
		IMethodBinding callee;
		List<?> args;
		if (p instanceof MethodInvocation mi) { callee = mi.resolveMethodBinding(); args = mi.arguments(); }
		else if (p instanceof ClassInstanceCreation c) { callee = c.resolveConstructorBinding(); args = c.arguments(); }
		else if (p instanceof ConstructorInvocation c) { callee = c.resolveConstructorBinding(); args = c.arguments(); }
		else if (p instanceof SuperConstructorInvocation c) { callee = c.resolveConstructorBinding(); args = c.arguments(); }
		else if (p instanceof SuperMethodInvocation c) { callee = c.resolveMethodBinding(); args = c.arguments(); }
		else return false;
		int i = args.indexOf(n);
		// An array passed as the varargs parameter is the parameter itself, not one of its elements.
		boolean element = callee != null && callee.isVarargs() && i >= callee.getParameterTypes().length - 1
				&& !(n.resolveTypeBinding() != null && n.resolveTypeBinding().isArray());
		return callee != null && i >= 0 && !element && guards(model, callee, i, seen);
	}

	/** `if (param == null) error(SWT.ERROR_NULL_ARGUMENT)` (or a throw) on a String parameter, or
	 * `if (items[i] == null) error(...)` on a String[] element (or its for-each variable): it fires only for jrt.NullString, which
	 * a test's literal null becomes; "" is a valid String (README "Round 11 null-string"). Every
	 * other String null check keeps `== ""`. Null when ie is neither. */
	static String stringNullCheck(Emitter emitter, InfixExpression ie, InfixExpression.Operator op) {
		Expression other = ie.getRightOperand() instanceof NullLiteral ? ie.getLeftOperand() : ie.getRightOperand();
		if (nullGuardedParam(ie) == null && !isElementGuard(ie, other)) return null;
		ITypeBinding t = other.resolveTypeBinding();
		if (t == null || !dev.gowt.j2go.GoTypes.map(t, emitter).equals("string")) return null;
		emitter.fileImports.add(dev.gowt.j2go.Manual.JRT_IMPORT);
		return "(" + emitter.expr(other) + (op == InfixExpression.Operator.EQUALS ? " == " : " != ") + "jrt.NullString)";
	}

	/** `x == null` against jrt.NullString, for an x that came from a NullableStrings method. */
	static String nullSentinelCompare(Emitter emitter, InfixExpression ie, InfixExpression.Operator op) {
		Expression other = ie.getRightOperand() instanceof NullLiteral ? ie.getLeftOperand() : ie.getRightOperand();
		emitter.fileImports.add(dev.gowt.j2go.Manual.JRT_IMPORT);
		return "(" + emitter.expr(other) + (op == InfixExpression.Operator.EQUALS ? " == " : " != ") + "jrt.NullString)";
	}

	private static boolean isElementGuard(InfixExpression ie, Expression other) {
		InfixExpression.Operator op = ie.getOperator();
		return (op == InfixExpression.Operator.EQUALS || op == InfixExpression.Operator.NOT_EQUALS)
				&& (ie.getLeftOperand() instanceof NullLiteral || ie.getRightOperand() instanceof NullLiteral)
				&& (other instanceof ArrayAccess || isForEachVariable(other)) && isGuardCondition(ie, false);
	}

	/** The array parameter whose element is null-checked by an error guard (`items[i] == null`, or the for-each variable over it), or null. */
	static IVariableBinding elementGuardArray(InfixExpression ie) {
		InfixExpression.Operator op = ie.getOperator();
		if (op != InfixExpression.Operator.EQUALS && op != InfixExpression.Operator.NOT_EQUALS) return null;
		Expression l = ie.getLeftOperand(), r = ie.getRightOperand();
		Expression other = r instanceof NullLiteral ? l : l instanceof NullLiteral ? r : null;
		if (other == null || !isGuardCondition(ie, false)) return null;
		Expression arr = null;
		if (other instanceof ArrayAccess aa) arr = aa.getArray();
		else if (other instanceof SimpleName n && n.resolveBinding() instanceof IVariableBinding vb) {
			for (ASTNode p = n.getParent(); p != null && arr == null; p = p.getParent()) {
				if (p instanceof EnhancedForStatement f && f.getParameter().resolveBinding() != null && f.getParameter().resolveBinding().isEqualTo(vb)) arr = f.getExpression();
			}
		}
		return arr instanceof SimpleName a && a.resolveBinding() instanceof IVariableBinding av && av.isParameter() ? av : null;
	}

	// `for (String item : items)`: the loop variable stands for an element like items[i].
	private static boolean isForEachVariable(Expression e) {
		if (!(e instanceof SimpleName n) || !(n.resolveBinding() instanceof IVariableBinding vb)) return false;
		for (ASTNode p = e.getParent(); p != null; p = p.getParent()) {
			if (p instanceof EnhancedForStatement f && f.getParameter().resolveBinding() != null && f.getParameter().resolveBinding().isEqualTo(vb)) return true;
		}
		return false;
	}

	/** The parameter of a null-argument guard `param ==/!= null`, or null if ie is not one. Shared
	 * with guards(), which asks the same question of a callee's body. */
	static IVariableBinding nullGuardedParam(InfixExpression ie) {
		InfixExpression.Operator op = ie.getOperator();
		if (op != InfixExpression.Operator.EQUALS && op != InfixExpression.Operator.NOT_EQUALS) return null;
		Expression l = ie.getLeftOperand(), r = ie.getRightOperand();
		Expression other = r instanceof NullLiteral ? l : l instanceof NullLiteral ? r : null;
		if (!(other instanceof SimpleName n) || !(n.resolveBinding() instanceof IVariableBinding vb)) return null;
		return vb.isParameter() && isGuardCondition(ie, true) ? vb : null;
	}

	// e is the condition of an if (alone or inside a || chain) whose then-branch is a throw or
	// error(...): ERROR_NULL_ARGUMENT only when nullArgumentOnly, any error code otherwise.
	private static boolean isGuardCondition(Expression e, boolean nullArgumentOnly) {
		ASTNode p = e.getParent();
		while (p instanceof ParenthesizedExpression || p instanceof InfixExpression ie && ie.getOperator() == InfixExpression.Operator.CONDITIONAL_OR) {
			p = p.getParent();
		}
		if (!(p instanceof IfStatement is)) return false;
		Statement then = is.getThenStatement();
		if (then instanceof Block b && b.statements().size() == 1) then = (Statement) b.statements().get(0);
		if (then instanceof ThrowStatement) return true;
		return then instanceof ExpressionStatement es && es.getExpression() instanceof MethodInvocation mi
				&& mi.getName().getIdentifier().equals("error") && !mi.arguments().isEmpty()
				&& (!nullArgumentOnly || mi.arguments().get(0) instanceof Name arg && arg.getFullyQualifiedName().endsWith("ERROR_NULL_ARGUMENT"));
	}
}
