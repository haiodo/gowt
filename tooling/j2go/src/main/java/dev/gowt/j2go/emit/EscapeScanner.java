package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.*;

/** Pre-scans a try/catch's body + catches for control flow that must escape the recover()
 * closure: return always does; break/continue only outside a loop/switch scanned with it. */
final class EscapeScanner extends ASTVisitor {
	boolean hasReturn, hasBreak, hasContinue;
	int depth;

	@Override
	public boolean visit(ReturnStatement n) {
		hasReturn = true;
		return true;
	}

	@Override
	public boolean visit(BreakStatement n) {
		if (n.getLabel() == null && depth == 0) hasBreak = true;
		return true;
	}

	@Override
	public boolean visit(ContinueStatement n) {
		if (n.getLabel() == null && depth == 0) hasContinue = true;
		return true;
	}

	@Override
	public boolean visit(ForStatement n) {
		depth++;
		return true;
	}

	@Override
	public void endVisit(ForStatement n) {
		depth--;
	}

	@Override
	public boolean visit(EnhancedForStatement n) {
		depth++;
		return true;
	}

	@Override
	public void endVisit(EnhancedForStatement n) {
		depth--;
	}

	@Override
	public boolean visit(WhileStatement n) {
		depth++;
		return true;
	}

	@Override
	public void endVisit(WhileStatement n) {
		depth--;
	}

	@Override
	public boolean visit(DoStatement n) {
		depth++;
		return true;
	}

	@Override
	public void endVisit(DoStatement n) {
		depth--;
	}

	@Override
	public boolean visit(SwitchStatement n) {
		depth++;
		return true;
	}

	@Override
	public void endVisit(SwitchStatement n) {
		depth--;
	}

	// A nested anonymous class/lambda is its own scope - a return inside it is its own, not
	// the enclosing method's, so its subtree is not scanned.
	@Override
	public boolean visit(AnonymousClassDeclaration n) {
		return false;
	}

	@Override
	public boolean visit(LambdaExpression n) {
		return false;
	}
}
