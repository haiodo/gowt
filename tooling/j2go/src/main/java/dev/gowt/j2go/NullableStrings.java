package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.util.*;

/**
 * String-returning internal methods that `return null` and whose result some caller tests against null
 * (Text.verifyText: null means "rejected", "" is a valid replacement). A Go string has no null, so there
 * the `return null` is jrt.NullString and a safe call site compares against it. A site is safe when the
 * result only reaches a null test: `call() == null`, or `v = call();` followed by `if (v == null ...)`
 * whose null branch neither uses v nor lets it be used after the if. Every other site normalizes the
 * sentinel back to "" (jrt.NullToEmpty), so it cannot leak. Public methods keep the "" convention.
 */
public final class NullableStrings {
	private final TypeModel model;
	private Set<String> safeKeys; // name(paramTypes) of methods with at least one safe call site
	private final Map<String, Boolean> memo = new HashMap<>();

	NullableStrings(TypeModel model) {
		this.model = model;
	}

	private static String key(IMethodBinding mb) {
		StringBuilder k = new StringBuilder(mb.getName()).append('(');
		for (ITypeBinding t : mb.getMethodDeclaration().getParameterTypes()) k.append(t.getErasure().getQualifiedName()).append(',');
		return k.toString();
	}

	private static boolean candidate(IMethodBinding mb) {
		return mb != null && mb.getReturnType().getQualifiedName().equals("java.lang.String") && !Modifier.isPublic(mb.getModifiers());
	}

	private static Expression strip(Expression e) {
		while (e instanceof ParenthesizedExpression p) e = p.getExpression();
		return e;
	}

	private static Expression nullOperand(InfixExpression ie) {
		InfixExpression.Operator op = ie.getOperator();
		if (op != InfixExpression.Operator.EQUALS && op != InfixExpression.Operator.NOT_EQUALS) return null;
		return ie.getRightOperand() instanceof NullLiteral ? ie.getLeftOperand() : ie.getLeftOperand() instanceof NullLiteral ? ie.getRightOperand() : null;
	}

	private static boolean refs(ASTNode n, IVariableBinding v) {
		boolean[] hit = {false};
		n.accept(new ASTVisitor() {
			@Override public boolean visit(SimpleName s) { hit[0] |= s.resolveBinding() instanceof IVariableBinding b && b.isEqualTo(v); return false; }
		});
		return hit[0];
	}

	private static boolean abrupt(Statement s) {
		if (s instanceof Block b && !b.statements().isEmpty()) s = (Statement) b.statements().get(b.statements().size() - 1);
		return s instanceof ReturnStatement || s instanceof ThrowStatement || s instanceof BreakStatement || s instanceof ContinueStatement;
	}

	/** The statement `v = call;` / `T v = call;` and the variable it fills, or null. */
	private static IVariableBinding assignedVar(MethodInvocation mi) {
		ASTNode p = mi.getParent();
		if (p instanceof VariableDeclarationFragment f && f.getInitializer() == mi && f.getParent() instanceof VariableDeclarationStatement) return f.resolveBinding();
		if (p instanceof Assignment a && a.getRightHandSide() == mi && a.getOperator() == Assignment.Operator.ASSIGN && a.getParent() instanceof ExpressionStatement
				&& a.getLeftHandSide() instanceof SimpleName n && n.resolveBinding() instanceof IVariableBinding v && !v.isField()) return v;
		return null;
	}

	/** The `if (v == null ...)` right after the statement holding `v = call`, as the if and v. */
	private static IfStatement guardAfter(MethodInvocation mi, IVariableBinding v) {
		ASTNode stmt = mi.getParent() instanceof Assignment a ? a.getParent() : mi.getParent().getParent();
		if (!(stmt.getParent() instanceof Block blk)) return null;
		int i = blk.statements().indexOf(stmt);
		if (i < 0 || i + 1 >= blk.statements().size() || !(blk.statements().get(i + 1) instanceof IfStatement is)) return null;
		return strip(is.getExpression()) instanceof InfixExpression ie && nullOperand(ie) instanceof Expression o && strip(o) instanceof SimpleName n
				&& n.resolveBinding() instanceof IVariableBinding b && b.isEqualTo(v) ? is : null;
	}

	private static boolean safeSite(MethodInvocation mi) {
		ASTNode p = mi.getParent();
		while (p instanceof ParenthesizedExpression) p = p.getParent();
		if (p instanceof InfixExpression ie && nullOperand(ie) != null) return true;
		IVariableBinding v = assignedVar(mi);
		IfStatement is = v == null ? null : guardAfter(mi, v);
		if (is == null) return false;
		boolean isEq = ((InfixExpression) strip(is.getExpression())).getOperator() == InfixExpression.Operator.EQUALS;
		Statement nullBranch = isEq ? is.getThenStatement() : is.getElseStatement();
		if (nullBranch != null && refs(nullBranch, v)) return false;
		if (nullBranch != null && abrupt(nullBranch)) return true;
		Block blk = (Block) is.getParent();
		for (int j = blk.statements().indexOf(is) + 1; j < blk.statements().size(); j++) if (refs((ASTNode) blk.statements().get(j), v)) return false;
		return true;
	}

	private void scan() {
		safeKeys = new HashSet<>();
		for (MethodDeclaration md : model.allDeclarations()) {
			if (md.getBody() == null) continue;
			md.getBody().accept(new ASTVisitor() {
				@Override public boolean visit(MethodInvocation mi) {
					IMethodBinding mb = mi.resolveMethodBinding();
					if (candidate(mb) && safeSite(mi)) safeKeys.add(key(mb));
					return true;
				}
			});
		}
	}

	private boolean nullable(IMethodBinding mb) {
		if (safeKeys == null) scan();
		if (!candidate(mb) || !safeKeys.contains(key(mb))) return false;
		return memo.computeIfAbsent(key(mb) + mb.getDeclaringClass().getQualifiedName(), k -> {
			for (MethodDeclaration md : model.relatedDeclarations(mb)) {
				if (md.getBody() == null) continue;
				boolean[] hit = {false};
				md.getBody().accept(new ASTVisitor() {
					@Override public boolean visit(ReturnStatement rs) { hit[0] |= rs.getExpression() instanceof NullLiteral; return false; }
					@Override public boolean visit(LambdaExpression l) { return false; }
					@Override public boolean visit(AnonymousClassDeclaration a) { return false; }
				});
				if (hit[0]) return true;
			}
			return false;
		});
	}

	/** `return null;` in a nullable method. */
	public boolean isNullReturn(ReturnStatement rs) {
		if (!(rs.getExpression() instanceof NullLiteral)) return false;
		ASTNode p = rs.getParent();
		while (p != null && !(p instanceof MethodDeclaration) && !(p instanceof LambdaExpression) && !(p instanceof AnonymousClassDeclaration)) p = p.getParent();
		return p instanceof MethodDeclaration md && md.resolveBinding() != null && nullable(md.resolveBinding());
	}

	/** A call of a nullable method whose result may flow anywhere: the sentinel must become "" there. */
	public boolean needsNormalizing(MethodInvocation mi) {
		IMethodBinding mb = mi.resolveMethodBinding();
		return mb != null && nullable(mb) && !safeSite(mi);
	}

	/** `call() == null`, or `v == null` right after `v = call()` at a safe site of a nullable method. */
	public boolean isNullCompare(InfixExpression ie) {
		Expression other = nullOperand(ie);
		if (other == null) return false;
		other = strip(other);
		if (other instanceof MethodInvocation mi) return mi.resolveMethodBinding() != null && nullable(mi.resolveMethodBinding()) && safeSite(mi);
		if (!(other instanceof SimpleName n) || !(n.resolveBinding() instanceof IVariableBinding v)) return false;
		ASTNode is = ie.getParent();
		while (is instanceof ParenthesizedExpression) is = is.getParent();
		if (!(is instanceof IfStatement ifs) || !(ifs.getParent() instanceof Block blk)) return false;
		int i = blk.statements().indexOf(ifs) - 1;
		if (i < 0) return false;
		Object prev = blk.statements().get(i);
		Expression call = prev instanceof VariableDeclarationStatement vds && vds.fragments().size() == 1 ? ((VariableDeclarationFragment) vds.fragments().get(0)).getInitializer()
				: prev instanceof ExpressionStatement es && es.getExpression() instanceof Assignment a ? a.getRightHandSide() : null;
		return call instanceof MethodInvocation mi && assignedVar(mi) != null && assignedVar(mi).isEqualTo(v) && isNullCompare0(mi);
	}

	private boolean isNullCompare0(MethodInvocation mi) {
		return mi.resolveMethodBinding() != null && nullable(mi.resolveMethodBinding()) && safeSite(mi);
	}
}
