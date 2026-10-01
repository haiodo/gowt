package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

import static dev.gowt.j2go.emit.EmitUtil.ind;

/** for/enhanced-for/while/do statements and their loop-head/prelude-hoisting helpers. */
final class LoopEmitter {

	private final Emitter emitter;

	LoopEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// `for cond; post {`; a condition with hoisted statements (`(bit >>= b) != 0`) is re-evaluated
	// at the top of every iteration instead.
	private String loopHead(Expression cond, String post, int indent) {
		List<String> hoisted = new ArrayList<>();
		String c = cond == null ? "" : withPrelude(() -> emitter.expr(cond), hoisted);
		String inHead = hoisted.isEmpty() && !c.equals("true") ? c : ""; // `for {` is a terminating statement, `for true {` is not
		StringBuilder b = new StringBuilder(ind(indent)).append("for ");
		b.append(post.isEmpty() ? inHead : "; " + inHead + "; " + post).append(" {\n");
		if (hoisted.isEmpty()) return b.toString();
		for (String p : hoisted) b.append(ind(indent + 1)).append(p).append('\n');
		b.append(ind(indent + 1)).append("if !(").append(c).append(") {\n").append(ind(indent + 2)).append("break\n")
				.append(ind(indent + 1)).append("}\n");
		return b.toString();
	}

	private String withPrelude(java.util.function.Supplier<String> emit, List<String> hoisted) {
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		String text = emit.get();
		hoisted.addAll(emitter.prelude);
		emitter.prelude = saved;
		return text;
	}


	// Bare text (no newline) for a for-loop init/update clause: prefers Go's native x++/x--
	// and plain assignment over the general temp-var expression forms.
	private String exprAsSimpleStmt(Expression e) {
		if (e instanceof PostfixExpression pf) return emitter.expr(pf.getOperand()) + pf.getOperator().toString();
		String opText = e instanceof PrefixExpression pf ? pf.getOperator().toString() : "";
		if (e instanceof PrefixExpression pf && (opText.equals("++") || opText.equals("--"))) {
			return emitter.expr(pf.getOperand()) + opText;
		}
		if (e instanceof Assignment a) {
			String lhs = emitter.expr(a.getLeftHandSide());
			String rhs = emitter.adaptNumeric(emitter.expr(a.getRightHandSide()), a.getRightHandSide().resolveTypeBinding(), a.getLeftHandSide().resolveTypeBinding());
			String boolOp = emitter.booleanCompoundOp(a, lhs, rhs);
			if (boolOp != null) return boolOp;
			return emitter.compoundAssign(a, lhs, rhs);
		}
		return emitter.expr(e);
	}

	String emitFor(ForStatement fs, int indent) {
		StringBuilder b = new StringBuilder();
		List<?> inits = fs.initializers();
		List<?> updaters = fs.updaters();
		boolean singleVarInit = inits.size() == 1 && inits.get(0) instanceof VariableDeclarationExpression vde
				&& vde.fragments().size() == 1;
		boolean singlePlainInit = inits.size() == 1 && !(inits.get(0) instanceof VariableDeclarationExpression);
		if ((singleVarInit || singlePlainInit) && updaters.size() <= 1) {
			String initText;
			if (singleVarInit) {
				VariableDeclarationExpression vde = (VariableDeclarationExpression) inits.get(0);
				VariableDeclarationFragment f = (VariableDeclarationFragment) vde.fragments().get(0);
				String goType = dev.gowt.j2go.GoTypes.map(vde.getType().resolveBinding(), emitter);
				// Short var decl always infers from the RHS; force the Go type explicitly so an
				// untyped literal (`int i = 0`) doesn't silently default to plain int.
				initText = emitter.sanitizeIdent(f.getName().getIdentifier()) + " := " + goType + "(" + emitter.expr(f.getInitializer()) + ")";
			} else {
				initText = exprAsSimpleStmt((Expression) inits.get(0));
			}
			List<String> hoisted = new ArrayList<>();
			String condText = fs.getExpression() == null ? "" : withPrelude(() -> emitter.expr(fs.getExpression()), hoisted);
			String updText = updaters.isEmpty() ? "" : withPrelude(() -> exprAsSimpleStmt((Expression) updaters.get(0)), hoisted);
			// A hoisted update (`sp = spr += d`) must run every iteration: only the general form can.
			if (!hoisted.isEmpty()) return emitForGeneral(fs, indent);
			b.append(ind(indent)).append("for ").append(initText).append("; ").append(condText).append("; ").append(updText).append(" {\n");
			emitter.loopSwitchDepth++;
			b.append(emitter.asBlock(fs.getBody(), indent + 1));
			emitter.loopSwitchDepth--;
			b.append(ind(indent)).append("}\n");
			return b.toString();
		}
		return emitForGeneral(fs, indent);
	}

	// General form (0/multiple initializers or updaters): hoist inits before the loop, append
	// updaters at the end of the body; braced so the hoisted vars keep the loop's scope.
	private String emitForGeneral(ForStatement fs, int indent) {
		StringBuilder b = new StringBuilder(ind(indent) + "{\n");
		indent++;
		for (Object o : fs.initializers()) {
			if (o instanceof VariableDeclarationExpression vde) {
				for (Object fo : vde.fragments()) {
					VariableDeclarationFragment f = (VariableDeclarationFragment) fo;
					String goType = dev.gowt.j2go.GoTypes.map(vde.getType().resolveBinding(), emitter);
					String init = emitter.adaptNumeric(emitter.exprInto(f.getInitializer(), b, indent), f.getInitializer().resolveTypeBinding(), vde.getType().resolveBinding());
					b.append(ind(indent)).append("var ").append(emitter.sanitizeIdent(f.getName().getIdentifier())).append(' ').append(goType).append(" = ").append(init).append('\n');
				}
			} else {
				b.append(ind(indent)).append(exprAsSimpleStmt((Expression) o)).append('\n');
			}
		}
		// Updaters run as the post statement (a closure, so a `continue` still reaches them).
		StringBuilder post = new StringBuilder();
		for (Object o : fs.updaters()) {
			List<String> hoisted = new ArrayList<>();
			String upd = withPrelude(() -> exprAsSimpleStmt((Expression) o), hoisted);
			for (String p : hoisted) post.append(ind(indent + 1)).append(p).append('\n');
			post.append(ind(indent + 1)).append(upd).append('\n');
		}
		b.append(loopHead(fs.getExpression(), post.length() == 0 ? "" : "func() {\n" + post + ind(indent) + "}()", indent));
		emitter.loopSwitchDepth++;
		b.append(emitter.asBlock(fs.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		b.append(ind(indent)).append("}\n");
		b.append(ind(indent - 1)).append("}\n");
		return b.toString();
	}

	String emitEnhancedFor(EnhancedForStatement efs, int indent) {
		ITypeBinding collType = efs.getExpression().resolveTypeBinding();
		String varName = emitter.sanitizeIdent(efs.getParameter().getName().getIdentifier());
		StringBuilder b = new StringBuilder();
		String collText = emitter.exprInto(efs.getExpression(), b, indent);
		if (collType != null && collType.isArray()) {
			b.append(ind(indent)).append("for _, ").append(varName).append(" := range ").append(collText).append(" {\n");
			emitter.loopSwitchDepth++;
			b.append(emitter.asBlock(efs.getBody(), indent + 1));
			emitter.loopSwitchDepth--;
			b.append(ind(indent)).append("}\n");
			return b.toString();
		}
		// A java.util collection is a jrt.List of erased elements: cast each back to the loop type.
		if (collType != null && dev.gowt.j2go.GoTypes.map(collType, emitter).equals("*jrt.List")) {
			String elemType = dev.gowt.j2go.GoTypes.map(efs.getParameter().getType().resolveBinding(), emitter);
			String tmp = "elem" + (++emitter.tempCounter);
			b.append(ind(indent)).append("for _, ").append(tmp).append(" := range ").append(collText).append(".ToArray() {\n");
			b.append(ind(indent + 1)).append(varName).append(" := jrt.Cast[").append(elemType).append("](").append(tmp).append(")\n");
			emitter.loopSwitchDepth++;
			b.append(emitter.asBlock(efs.getBody(), indent + 1));
			emitter.loopSwitchDepth--;
			b.append(ind(indent)).append("}\n");
			return b.toString();
		}
		emitter.unsupported.add("EnhancedForStatement: non-array Iterable " + efs);
		// Not a terminating statement: code after the loop stays reachable for go vet.
		b.append(ind(indent)).append("func() { panic(\"j2go: unsupported EnhancedForStatement over non-array\") }()\n");
		return b.toString();
	}

	String emitWhile(WhileStatement ws, int indent) {
		StringBuilder b = new StringBuilder(loopHead(ws.getExpression(), "", indent));
		emitter.loopSwitchDepth++;
		b.append(emitter.asBlock(ws.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	String emitDo(DoStatement ds, int indent) {
		StringBuilder b = new StringBuilder();
		b.append(ind(indent)).append("for {\n");
		emitter.loopSwitchDepth++;
		b.append(emitter.asBlock(ds.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		String cond = emitter.exprInto(ds.getExpression(), b, indent + 1);
		b.append(ind(indent + 1)).append("if !(").append(cond).append(") {\n");
		b.append(ind(indent + 2)).append("break\n");
		b.append(ind(indent + 1)).append("}\n");
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}
}
