package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;

import static dev.gowt.j2go.emit.EmitUtil.ind;

/** switch (statement and expression), throw, try/catch/finally, catch dispatch. */
final class ControlFlowEmitter {

	private final Emitter emitter;

	ControlFlowEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- switch expression (RGB HSB only)

	/** Handles `target = switch (e) { case C -> { stmts; yield v; } ... };` lowering. */
	String emitSwitchExpressionAssign(String targetLhs, SwitchExpression se, int indent) {
		StringBuilder b = new StringBuilder();
		String subject = emitter.exprInto(se.getExpression(), b, indent);
		b.append(ind(indent)).append("switch ").append(subject).append(" {\n");
		List<Object> stmts = se.statements();
		for (int i = 0; i < stmts.size(); i++) {
			SwitchCase sc = (SwitchCase) stmts.get(i);
			Statement body = (Statement) stmts.get(i + 1);
			i++;
			if (sc.isDefault()) {
				b.append(ind(indent)).append("default:\n");
			} else {
				List<String> labels = new ArrayList<>();
				for (Object ex : sc.expressions()) labels.add(emitter.expr((Expression) ex));
				b.append(ind(indent)).append("case ").append(String.join(", ", labels)).append(":\n");
			}
			b.append(emitSwitchCaseBody(body, targetLhs, se.resolveTypeBinding(), indent + 1));
		}
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private String emitSwitchCaseBody(Statement body, String targetLhs, ITypeBinding targetType, int indent) {
		List<Statement> stmts = body instanceof Block bl ? bl.statements() : List.of(body);
		StringBuilder b = new StringBuilder();
		for (Statement s : stmts) {
			if (s instanceof YieldStatement ys) {
				b.append(emitStatement0AsAssign(targetLhs, ys.getExpression(), targetType, indent));
			} else {
				b.append(emitter.stmt(s, indent));
			}
		}
		return b.toString();
	}

	private String emitStatement0AsAssign(String lhs, Expression rhs, ITypeBinding targetType, int indent) {
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		String text = emitter.adaptNumeric(emitter.expr(rhs), rhs.resolveTypeBinding(), targetType);
		StringBuilder b = new StringBuilder();
		for (String p : emitter.prelude) b.append(ind(indent)).append(p).append('\n');
		b.append(ind(indent)).append(lhs).append(" = ").append(text).append('\n');
		emitter.prelude = saved;
		return b.toString();
	}

	// ---------------------------------------------------------------- switch statement

	String emitSwitchStatement(SwitchStatement sw, int indent) {
		StringBuilder b = new StringBuilder();
		String subject = emitter.exprInto(sw.getExpression(), b, indent);
		ITypeBinding subjectType = sw.getExpression().resolveTypeBinding();
		b.append(ind(indent)).append("switch ").append(subject).append(" {\n");
		emitter.loopSwitchDepth++;
		List<?> stmts = sw.statements();
		int i = 0;
		while (i < stmts.size()) {
			List<SwitchCase> labelGroup = new ArrayList<>();
			while (i < stmts.size() && stmts.get(i) instanceof SwitchCase sc) {
				labelGroup.add(sc);
				i++;
			}
			List<Statement> body = new ArrayList<>();
			while (i < stmts.size() && !(stmts.get(i) instanceof SwitchCase)) {
				body.add((Statement) stmts.get(i));
				i++;
			}
			boolean isLastGroup = i >= stmts.size();
			boolean isDefault = labelGroup.stream().anyMatch(SwitchCase::isDefault);
			boolean isArrow = labelGroup.get(0).isSwitchLabeledRule();
			if (isDefault) {
				b.append(ind(indent)).append("default:\n");
			} else {
				List<String> labels = new ArrayList<>();
				for (SwitchCase sc : labelGroup) {
					for (Object ex : sc.expressions()) {
						Expression e = (Expression) ex;
						// Java allows a case constant of a narrower type than the switch subject
						// (a char case on an int switch, e.g. SWT.ESC) - Go's typed constants need
						// the same explicit conversion an ordinary binary operand would.
						labels.add(emitter.adaptNumeric(emitter.expr(e), e.resolveTypeBinding(), subjectType));
					}
				}
				b.append(ind(indent)).append("case ").append(String.join(", ", labels)).append(":\n");
			}
			for (Statement st : body) b.append(emitter.stmt(st, indent + 1));
			// Java's case falls through by default, Go's does not - add an explicit fallthrough
			// unless the body exits, or this is the last group (Go rejects a trailing one).
			if (!isArrow && !isLastGroup && !bodyExits(body)) {
				b.append(ind(indent + 1)).append("fallthrough\n");
			}
		}
		emitter.loopSwitchDepth--;
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private boolean bodyExits(List<Statement> body) {
		return !body.isEmpty() && stmtExits(body.get(body.size() - 1));
	}

	private boolean stmtExits(Statement s) {
		if (s instanceof BreakStatement || s instanceof ReturnStatement
				|| s instanceof ThrowStatement || s instanceof ContinueStatement) return true;
		if (s instanceof Block bl && !bl.statements().isEmpty()) {
			return stmtExits((Statement) bl.statements().get(bl.statements().size() - 1));
		}
		if (s instanceof IfStatement is) return is.getElseStatement() != null && stmtExits(is.getThenStatement()) && stmtExits(is.getElseStatement());
		return false;
	}

	// ---------------------------------------------------------------- throw / try / catch / finally

	String emitThrow(ThrowStatement ts, int indent) {
		StringBuilder b = new StringBuilder();
		String expr = emitter.exprInto(ts.getExpression(), b, indent);
		b.append(ind(indent)).append("panic(").append(expr).append(")\n");
		return b.toString();
	}

	String emitTry(TryStatement ts, int indent) {
		StringBuilder b = new StringBuilder();
		List<String> resourceCloses = new ArrayList<>();
		for (Object o : ts.resources()) {
			if (o instanceof VariableDeclarationExpression vde) {
				for (Object fo : vde.fragments()) {
					VariableDeclarationFragment f = (VariableDeclarationFragment) fo;
					String name = emitter.sanitizeIdent(f.getName().getIdentifier());
					String goType = dev.gowt.j2go.GoTypes.map(vde.getType().resolveBinding(), emitter);
					String init = emitter.exprInto(f.getInitializer(), b, indent);
					b.append(ind(indent)).append("var ").append(name).append(' ').append(goType)
							.append(" = ").append(init).append('\n');
					// An unmapped JDK resource (InputStream) is any: close it only if it can be.
					resourceCloses.add(goType.equals("any") ? "func() { if c, ok := " + name + ".(interface{ Close() }); ok { c.Close() } }()"
							: name + ".Close()");
				}
			} else {
				emitter.unsupported.add("TryStatement: non-declaration resource " + o);
			}
		}
		List<?> catches = ts.catchClauses();
		Block finallyBlock = ts.getFinally();
		// No catch: nothing needs panic/recover, so the body is inlined directly (no closure);
		// a defer for the resource/finally cleanup then runs on any exit path, return included.
		if (catches.isEmpty()) {
			if (finallyBlock != null) b.append(deferBlock(finallyBlock, indent));
			for (int i = resourceCloses.size() - 1; i >= 0; i--) {
				b.append(ind(indent)).append("defer ").append(resourceCloses.get(i)).append('\n');
			}
			b.append(emitter.block(ts.getBody(), indent));
			return b.toString();
		}
		if (finallyBlock != null) b.append(deferBlock(finallyBlock, indent));
		for (int i = resourceCloses.size() - 1; i >= 0; i--) {
			b.append(ind(indent)).append("defer ").append(resourceCloses.get(i)).append('\n');
		}
		b.append(emitTryCatch(ts, catches, indent));
		return b.toString();
	}

	private String deferBlock(Block block, int indent) {
		return ind(indent) + "defer func() {\n" + emitter.block(block, indent + 1) + ind(indent) + "}()\n";
	}

	/** synchronized (x) { body }: the one global jrt monitor (see internal/jrt/lang.go), released
	 * by a defer inside a block-scoped closure - the lock expression itself is not evaluated. */
	String emitSynchronized(SynchronizedStatement ss, int indent) {
		emitter.fileImports.add("github.com/haiodo/gowt/internal/jrt");
		return ind(indent) + "jrt.MonitorEnter()\n" + emitClosure(ss.getBody(), List.of(), "jrt.MonitorExit()", indent);
	}

	/** A catch needs panic/recover, which needs a Go closure boundary - see README for how a
	 * return/break/continue inside the try/catch body is made to escape it correctly. */
	@SuppressWarnings("unchecked")
	private String emitTryCatch(TryStatement ts, List<?> catchesRaw, int indent) {
		return emitClosure(ts.getBody(), (List<CatchClause>) catchesRaw, null, indent);
	}

	/** body inside `func() { defer ...; body }()`: catches become a recover() dispatch, deferText
	 * (if any) a plain defer; return/break/continue escape via flags re-played after the call. */
	private String emitClosure(Block body, List<CatchClause> catches, String deferText, int indent) {
		EscapeScanner scan = new EscapeScanner();
		body.accept(scan);
		for (CatchClause cc : catches) cc.getBody().accept(scan);

		StringBuilder b = new StringBuilder();
		boolean voidReturn = emitter.currentReturnType == null || dev.gowt.j2go.GoTypes.map(emitter.currentReturnType, emitter).isEmpty();
		String retVar = scan.hasReturn && !voidReturn ? "tret" + (++emitter.tempCounter) : null;
		String returnedFlag = scan.hasReturn ? "tretd" + (++emitter.tempCounter) : null;
		String brokeFlag = scan.hasBreak ? "tbrk" + (++emitter.tempCounter) : null;
		String continuedFlag = scan.hasContinue ? "tcnt" + (++emitter.tempCounter) : null;
		if (retVar != null) b.append(ind(indent)).append("var ").append(retVar).append(' ')
				.append(dev.gowt.j2go.GoTypes.map(emitter.currentReturnType, emitter)).append('\n');
		if (returnedFlag != null) b.append(ind(indent)).append(returnedFlag).append(" := false\n");
		if (brokeFlag != null) b.append(ind(indent)).append(brokeFlag).append(" := false\n");
		if (continuedFlag != null) b.append(ind(indent)).append(continuedFlag).append(" := false\n");

		String savedReturnedFlag = emitter.currentEscapeReturnedFlag;
		String savedBrokeFlag = emitter.currentEscapeBrokeFlag;
		String savedContinuedFlag = emitter.currentEscapeContinuedFlag;
		String savedRetVar = emitter.currentEscapeRetVar;
		int savedDepth = emitter.loopSwitchDepth;
		emitter.currentEscapeReturnedFlag = returnedFlag;
		emitter.currentEscapeBrokeFlag = brokeFlag;
		emitter.currentEscapeContinuedFlag = continuedFlag;
		emitter.currentEscapeRetVar = retVar;
		emitter.loopSwitchDepth = 0;

		b.append(ind(indent)).append("func() {\n");
		if (deferText != null) b.append(ind(indent + 1)).append("defer ").append(deferText).append('\n');
		if (!catches.isEmpty()) {
			b.append(ind(indent + 1)).append("defer func() {\n");
			b.append(ind(indent + 2)).append("r := recover()\n");
			b.append(ind(indent + 2)).append("if r == nil {\n").append(ind(indent + 3)).append("return\n")
					.append(ind(indent + 2)).append("}\n");
			b.append(emitCatchDispatch(catches, indent + 2));
			b.append(ind(indent + 1)).append("}()\n");
		}
		b.append(emitter.block(body, indent + 1));
		b.append(ind(indent)).append("}()\n");

		emitter.currentEscapeReturnedFlag = savedReturnedFlag;
		emitter.currentEscapeBrokeFlag = savedBrokeFlag;
		emitter.currentEscapeContinuedFlag = savedContinuedFlag;
		emitter.currentEscapeRetVar = savedRetVar;
		emitter.loopSwitchDepth = savedDepth;

		// Last statement of a non-void method: Java guarantees the body returned or threw, and Go
		// needs a terminating statement here, not a conditional one.
		if (retVar != null && isLastInMethodBody(body.getParent())) {
			b.append(ind(indent)).append("_ = ").append(returnedFlag).append('\n');
			b.append(emitter.returnOrEscape(indent, retVar));
		} else if (returnedFlag != null) {
			b.append(ind(indent)).append("if ").append(returnedFlag).append(" {\n");
			b.append(emitter.returnOrEscape(indent + 1, retVar));
			b.append(ind(indent)).append("}\n");
		}
		if (brokeFlag != null) {
			b.append(ind(indent)).append("if ").append(brokeFlag).append(" {\n");
			b.append(emitter.breakOrEscape(indent + 1));
			b.append(ind(indent)).append("}\n");
		}
		if (continuedFlag != null) {
			b.append(ind(indent)).append("if ").append(continuedFlag).append(" {\n");
			b.append(emitter.continueOrEscape(indent + 1));
			b.append(ind(indent)).append("}\n");
		}
		return b.toString();
	}

	private static boolean isLastInMethodBody(ASTNode stmt) {
		if (!(stmt.getParent() instanceof Block b) || !(b.getParent() instanceof MethodDeclaration)) return false;
		List<?> stmts = b.statements();
		return stmts.get(stmts.size() - 1) == stmt;
	}

	private String emitCatchDispatch(List<CatchClause> catches, int indent) {
		StringBuilder b = new StringBuilder();
		for (int i = 0; i < catches.size(); i++) {
			CatchClause cc = catches.get(i);
			SingleVariableDeclaration param = cc.getException();
			List<String> altTypes = catchAlternativeGoTypes(param);
			String varName = emitter.sanitizeIdent(param.getName().getIdentifier());
			boolean broad = altTypes.size() == 1 && altTypes.get(0).equals("error");
			b.append(ind(indent)).append(i == 0 ? "if " : "} else if ");
			if (altTypes.isEmpty()) {
				// An unmapped JDK exception (NumberFormatException, ...): nothing translated throws it.
				b.append("false {\n").append(ind(indent + 1)).append("var ").append(varName).append(" error\n");
				b.append(ind(indent + 1)).append("_ = ").append(varName).append('\n');
			} else if (broad) {
				b.append(varName).append(", ok := r.(error); ok {\n");
				b.append(ind(indent + 1)).append("_ = ").append(varName).append('\n');
			} else {
				b.append("func() bool { switch r.(type) { case ").append(String.join(", ", altTypes))
						.append(": return true }; return false }() {\n");
				if (altTypes.size() == 1) {
					// A single concrete alternative (catch (IOException e)) gets a real assertion,
					// so the catch body can use e as that type (e.g. pass it where error is wanted).
					b.append(ind(indent + 1)).append(varName).append(" := r.(").append(altTypes.get(0)).append(")\n");
				} else {
					// A multi-catch of unrelated concrete types has no single Go assertion - matched
					// via the type-switch probe above; the catch var keeps r's static (any) type.
					b.append(ind(indent + 1)).append(varName).append(" := r\n");
				}
				b.append(ind(indent + 1)).append("_ = ").append(varName).append('\n');
			}
			b.append(emitter.block(cc.getBody(), indent + 1));
		}
		b.append(ind(indent)).append("} else {\n");
		b.append(ind(indent + 1)).append("panic(r)\n");
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private List<String> catchAlternativeGoTypes(SingleVariableDeclaration param) {
		List<String> out = new ArrayList<>();
		Type t = param.getType();
		if (t instanceof UnionType ut) {
			for (Object o : ut.types()) out.add(catchGoType(((Type) o).resolveBinding()));
		} else {
			out.add(catchGoType(t.resolveBinding()));
		}
		out.removeIf(Objects::isNull);
		// Collapse to Go's error interface whenever every alternative resolves to it (a catch of
		// java.lang.RuntimeException/Error/Exception/Throwable - see README "Known gaps").
		if (!out.isEmpty() && out.stream().allMatch(s -> s.equals("error"))) return List.of("error");
		return out;
	}

	private String catchGoType(ITypeBinding t) {
		String qualified = t.getErasure().getQualifiedName();
		if (qualified.equals(Manual.JAVA_RUNTIME_EXCEPTION) || qualified.equals(Manual.JAVA_ERROR)
				|| qualified.equals(Manual.JAVA_EXCEPTION) || qualified.equals(Manual.JAVA_THROWABLE)) {
			return "error";
		}
		TypeModel.ClassInfo ci = emitter.model.lookup(t);
		if (ci != null) return (ci.isStruct || ci.isInterface) ? ci.goTypeName : "*" + ci.goTypeName;
		if (Manual.isManual(qualified)) {
			emitter.addManualImport(qualified);
			String gt = Manual.goTypeName(qualified);
			return Manual.isValueType(qualified) ? gt : "*" + gt;
		}
		emitter.unsupported.add("CatchClause: unresolved exception type " + qualified);
		return null;
	}
}
