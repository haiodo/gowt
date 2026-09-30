package dev.gowt.j2go.emit;

import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.HashSet;
import java.util.List;
import java.util.Set;

/** Does a callee treat a String parameter as a null-argument guard (see NumericEmitter.nullGuardedParam)?
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
		IVariableBinding vb = f.resolveBinding();
		ASTNode scope = f;
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
					IVariableBinding g = NumericEmitter.nullGuardedParam(ie);
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
		return callee != null && i >= 0 && !(callee.isVarargs() && i >= callee.getParameterTypes().length - 1)
				&& guards(model, callee, i, seen);
	}
}
