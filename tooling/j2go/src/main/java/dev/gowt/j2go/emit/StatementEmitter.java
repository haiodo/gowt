package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

import static dev.gowt.j2go.emit.EmitUtil.ind;

/** Blocks, var decls, expression statements, if/for/enhanced-for/while/do, return/break/continue
 * and the escape helpers, plus the expr-with-hoisted-prelude bridge used by every statement form. */
final class StatementEmitter {

	private final Emitter emitter;

	StatementEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	String emitBlockBody(Block block, int indent) {
		StringBuilder b = new StringBuilder();
		for (Object s : block.statements()) {
			b.append(emitStatement((Statement) s, indent));
		}
		return b.toString();
	}

	String emitStatement(Statement s, int indent) {
		return withPrelude(() -> emitStatementInner(s, indent), indent);
	}

	/** A lambda's expression body (`e -> x = v`) emitted as the statement it stands for. */
	String emitExpressionAsStatement(Expression e, int indent) {
		return withPrelude(() -> emitExpressionStatement(e, indent), indent);
	}

	private String withPrelude(java.util.function.Supplier<String> inner, int indent) {
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		String main = inner.get();
		StringBuilder b = new StringBuilder();
		for (String p : emitter.prelude) b.append(ind(indent)).append(p).append('\n');
		b.append(main);
		emitter.prelude = saved;
		return b.toString();
	}

	private String emitStatementInner(Statement s, int indent) {
		if (s instanceof ExpressionStatement es) return emitExpressionStatement(es.getExpression(), indent);
		if (s instanceof ReturnStatement rs) return emitReturn(rs, indent);
		if (s instanceof IfStatement is) return emitIf(is, indent);
		if (s instanceof VariableDeclarationStatement vds) return emitVarDecl(vds, indent);
		if (s instanceof Block b) return ind(indent) + "{\n" + emitBlockBody(b, indent + 1) + ind(indent) + "}\n";
		if (s instanceof SwitchStatement sw) return emitter.emitSwitchStatement(sw, indent);
		if (s instanceof ThrowStatement ts) return emitter.emitThrow(ts, indent);
		if (s instanceof TryStatement ts) return emitter.emitTry(ts, indent);
		if (s instanceof SynchronizedStatement ss) return emitter.emitSynchronized(ss, indent);
		if (s instanceof EmptyStatement) return "";
		if (s instanceof ForStatement fs) return emitFor(fs, indent);
		if (s instanceof EnhancedForStatement efs) return emitEnhancedFor(efs, indent);
		if (s instanceof WhileStatement ws) return emitWhile(ws, indent);
		if (s instanceof DoStatement ds) return emitDo(ds, indent);
		if (s instanceof LabeledStatement ls) {
			return ind(indent) + emitter.sanitizeIdent(ls.getLabel().getIdentifier()) + ":\n" + emitStatement(ls.getBody(), indent);
		}
		if (s instanceof BreakStatement bs) {
			if (bs.getLabel() == null) return breakOrEscape(indent);
			return ind(indent) + "break " + emitter.sanitizeIdent(bs.getLabel().getIdentifier()) + "\n";
		}
		if (s instanceof ContinueStatement cs) {
			if (cs.getLabel() == null) return continueOrEscape(indent);
			return ind(indent) + "continue " + emitter.sanitizeIdent(cs.getLabel().getIdentifier()) + "\n";
		}
		return unsupportedStmt(s, indent);
	}

	private String emitExpressionStatement(Expression e, int indent) {
		if (e instanceof Assignment a) {
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof Assignment) {
				return emitChainedAssignment(a, indent);
			}
			String lhs = emitter.expr(a.getLeftHandSide());
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof ConditionalExpression ce) {
				return emitCondIntoLvalue(lhs, ce, a.getLeftHandSide().resolveTypeBinding(), indent);
			}
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof SwitchExpression se) {
				return emitter.emitSwitchExpressionAssign(lhs, se, indent);
			}
			String rhs = emitter.adaptNumeric(emitter.expr(a.getRightHandSide()), a.getRightHandSide().resolveTypeBinding(), a.getLeftHandSide().resolveTypeBinding());
			String boolOp = booleanCompoundOp(a, lhs, rhs);
			if (boolOp != null) return ind(indent) + boolOp + "\n";
			return ind(indent) + compoundAssign(a, lhs, rhs) + "\n";
		}
		// x++;/--x; as their own statement: Go's native x++/x-- directly, no throwaway temp
		// (the generic emitExpr path below would leave a bare, unused-value statement).
		if (e instanceof PostfixExpression pf) return ind(indent) + emitter.expr(pf.getOperand()) + pf.getOperator().toString() + "\n";
		if (e instanceof PrefixExpression pf && (pf.getOperator().toString().equals("++") || pf.getOperator().toString().equals("--"))) {
			return ind(indent) + emitter.expr(pf.getOperand()) + pf.getOperator().toString() + "\n";
		}
		String text = emitter.expr(e);
		// An intrinsic can lower a call to a bare value (Objects.requireNonNull(x) -> x) - Go
		// rejects an unused value as a statement.
		if (!text.endsWith(")") && !text.endsWith("*/")) return ind(indent) + "_ = " + text + "\n";
		return ind(indent) + text + "\n";
	}

	private String emitChainedAssignment(Assignment a, int indent) {
		List<Expression> targets = new ArrayList<>();
		Expression cur = a;
		while (cur instanceof Assignment ca && ca.getOperator() == Assignment.Operator.ASSIGN) {
			targets.add(ca.getLeftHandSide());
			cur = ca.getRightHandSide();
		}
		String prev = emitter.expr(cur);
		ITypeBinding prevType = cur.resolveTypeBinding();
		StringBuilder b = new StringBuilder();
		// A chain's targets need not share one type (child = update[i] = composite: Composite
		// upcasts to Control at the outer target) - adapt at each step, not just reuse raw text.
		for (int i = targets.size() - 1; i >= 0; i--) {
			Expression target = targets.get(i);
			String lhsText = emitter.expr(target);
			String adapted = emitter.adaptNumeric(prev, prevType, target.resolveTypeBinding());
			b.append(ind(indent)).append(lhsText).append(" = ").append(adapted).append('\n');
			prev = lhsText;
			prevType = target.resolveTypeBinding();
		}
		return b.toString();
	}

	/** ExpressionStatement/assignment whose RHS is a plain (no-instanceof) ternary, or var-decl init. */
	private String emitCondIntoLvalue(String lhsText, ConditionalExpression ce, ITypeBinding targetType, int indent) {
		StringBuilder b = new StringBuilder();
		String cond = emitExprInto(ce.getExpression(), b, indent);
		String thenText = emitter.adaptNumeric(emitter.expr(ce.getThenExpression()), ce.getThenExpression().resolveTypeBinding(), targetType);
		String elseText = emitter.adaptNumeric(emitter.expr(ce.getElseExpression()), ce.getElseExpression().resolveTypeBinding(), targetType);
		b.append(ind(indent)).append("if ").append(cond).append(" {\n");
		// A branch that assigns the target to itself (columnWidth = cond ? columnWidth : ...) is
		// a real no-op Go's own `go vet` flags as suspicious - just skip the line.
		if (!thenText.equals(lhsText)) b.append(ind(indent + 1)).append(lhsText).append(" = ").append(thenText).append('\n');
		b.append(ind(indent)).append("} else {\n");
		if (!elseText.equals(lhsText)) b.append(ind(indent + 1)).append(lhsText).append(" = ").append(elseText).append('\n');
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	// `for cond; post {`; a condition with hoisted statements (`(bit >>= b) != 0`) is re-evaluated
	// at the top of every iteration instead.
	private String loopHead(Expression cond, String post, int indent) {
		List<String> hoisted = new ArrayList<>();
		String c = cond == null ? "" : withPrelude(() -> emitter.expr(cond), hoisted);
		String inHead = hoisted.isEmpty() ? c : "";
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

	/** Emits any prelude for e (instanceof hoisting) directly at `indent` into b, returns e's inline text. */
	String emitExprInto(Expression e, StringBuilder b, int indent) {
		List<String> savedPrelude = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		String text = emitter.expr(e);
		for (String p : emitter.prelude) b.append(ind(indent)).append(p).append('\n');
		emitter.prelude = savedPrelude;
		return text;
	}

	private String emitVarDecl(VariableDeclarationStatement vds, int indent) {
		StringBuilder b = new StringBuilder();
		ITypeBinding declType = vds.getType().resolveBinding();
		for (Object o : vds.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			// C-style declaration (`Touch touches[]`): the []  binds to the FRAGMENT, not the
			// shared type node - vds.getType() alone would miss it and declare a bare *Touch.
			ITypeBinding fragType = f.resolveBinding() != null ? f.resolveBinding().getType() : declType;
			String goType = dev.gowt.j2go.GoTypes.map(fragType, emitter);
			String name = emitter.sanitizeIdent(f.getName().getIdentifier());
			if (f.getInitializer() == null) {
				b.append(ind(indent)).append("var ").append(name).append(' ').append(goType).append('\n');
				continue;
			}
			if (f.getInitializer() instanceof ConditionalExpression ce) {
				b.append(ind(indent)).append("var ").append(name).append(' ').append(goType).append('\n');
				b.append(emitCondIntoLvalue(name, ce, fragType, indent));
				continue;
			}
			String init = emitter.adaptNumeric(emitExprInto(f.getInitializer(), b, indent), f.getInitializer().resolveTypeBinding(), fragType);
			// Explicit `var name Type = init`: `:=` would give a literal RHS Go's default int,
			// not int32, silently breaking later int32 comparisons/arithmetic against it.
			b.append(ind(indent)).append("var ").append(name).append(' ').append(goType).append(" = ").append(init).append('\n');
		}
		return b.toString();
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
			String boolOp = booleanCompoundOp(a, lhs, rhs);
			if (boolOp != null) return boolOp;
			return compoundAssign(a, lhs, rhs);
		}
		return emitter.expr(e);
	}

	private String compoundAssign(Assignment a, String lhs, String rhs) {
		if (a.getOperator() != Assignment.Operator.RIGHT_SHIFT_UNSIGNED_ASSIGN) return lhs + " " + a.getOperator() + " " + rhs;
		ITypeBinding lt = a.getLeftHandSide().resolveTypeBinding();
		String goType = dev.gowt.j2go.GoTypes.map(lt, emitter);
		String shifted = emitter.unsignedShift(lhs, lt, rhs);
		return lhs + " = " + (goType.equals("int32") || goType.equals("int64") ? shifted : goType + "(" + shifted + ")");
	}

	// Java allows |=/&=/^= on boolean operands (non-short-circuit logical assignment); Go has no
	// bool|bool at all, so these lower to the equivalent ||/&&/!= form instead.
	String booleanCompoundOp(Assignment a, String lhs, String rhs) {
		ITypeBinding lt = a.getLeftHandSide().resolveTypeBinding();
		if (lt == null || !lt.getName().equals("boolean")) return null;
		String goOp = switch (a.getOperator().toString()) {
			case "|=" -> "||";
			case "&=" -> "&&";
			case "^=" -> "!=";
			default -> null;
		};
		if (goOp == null) return null;
		// |=/&= don't short-circuit in Java either, but Go's ||/&& would skip a side-effecting rhs
		// once lhs already decides the result - hoist it into a temp first so it always runs.
		if (!goOp.equals("!=") && emitter.hasSideEffect(a.getRightHandSide())) {
			String tmp = "b" + (++emitter.tempCounter);
			emitter.prelude.add(tmp + " := " + rhs);
			rhs = tmp;
		}
		return lhs + " = " + lhs + " " + goOp + " " + rhs;
	}

	private String emitFor(ForStatement fs, int indent) {
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
			b.append(emitAsBlock(fs.getBody(), indent + 1));
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
					String init = emitter.adaptNumeric(emitExprInto(f.getInitializer(), b, indent), f.getInitializer().resolveTypeBinding(), vde.getType().resolveBinding());
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
		b.append(emitAsBlock(fs.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		b.append(ind(indent)).append("}\n");
		b.append(ind(indent - 1)).append("}\n");
		return b.toString();
	}

	private String emitEnhancedFor(EnhancedForStatement efs, int indent) {
		ITypeBinding collType = efs.getExpression().resolveTypeBinding();
		String varName = emitter.sanitizeIdent(efs.getParameter().getName().getIdentifier());
		StringBuilder b = new StringBuilder();
		String collText = emitExprInto(efs.getExpression(), b, indent);
		if (collType != null && collType.isArray()) {
			b.append(ind(indent)).append("for _, ").append(varName).append(" := range ").append(collText).append(" {\n");
			emitter.loopSwitchDepth++;
			b.append(emitAsBlock(efs.getBody(), indent + 1));
			emitter.loopSwitchDepth--;
			b.append(ind(indent)).append("}\n");
			return b.toString();
		}
		emitter.unsupported.add("EnhancedForStatement: non-array Iterable " + efs);
		b.append(ind(indent)).append("panic(\"j2go: unsupported EnhancedForStatement over non-array\")\n");
		return b.toString();
	}

	private String emitWhile(WhileStatement ws, int indent) {
		StringBuilder b = new StringBuilder(loopHead(ws.getExpression(), "", indent));
		emitter.loopSwitchDepth++;
		b.append(emitAsBlock(ws.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private String emitDo(DoStatement ds, int indent) {
		StringBuilder b = new StringBuilder();
		b.append(ind(indent)).append("for {\n");
		emitter.loopSwitchDepth++;
		b.append(emitAsBlock(ds.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		String cond = emitExprInto(ds.getExpression(), b, indent + 1);
		b.append(ind(indent + 1)).append("if !(").append(cond).append(") {\n");
		b.append(ind(indent + 2)).append("break\n");
		b.append(ind(indent + 1)).append("}\n");
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private String emitIf(IfStatement is, int indent) {
		StringBuilder b = new StringBuilder();
		String cond = emitExprInto(is.getExpression(), b, indent);
		b.append(ind(indent)).append("if ").append(cond).append(" {\n");
		b.append(emitAsBlock(is.getThenStatement(), indent + 1));
		if (is.getElseStatement() == null) {
			b.append(ind(indent)).append("}\n");
		} else {
			b.append(ind(indent)).append("} else {\n");
			b.append(emitAsBlock(is.getElseStatement(), indent + 1));
			b.append(ind(indent)).append("}\n");
		}
		return b.toString();
	}

	private String emitAsBlock(Statement s, int indent) {
		if (s instanceof Block b) return emitBlockBody(b, indent);
		return emitStatement(s, indent);
	}

	private String emitReturn(ReturnStatement rs, int indent) {
		StringBuilder b = new StringBuilder();
		if (rs.getExpression() == null) {
			b.append(returnOrEscape(indent, null));
			return b.toString();
		}
		Expression e = rs.getExpression();
		String text = emitExprInto(e, b, indent);
		// Java implicitly widens/narrows a return value (char -> int) and upcasts an object one
		// (Composite -> Control) to the method's declared type; Go needs both spelled out.
		if (emitter.currentReturnType != null) {
			text = emitter.adaptNumeric(text, e.resolveTypeBinding(), emitter.currentReturnType);
		}
		b.append(returnOrEscape(indent, text));
		return b.toString();
	}

	/** Plain `return`/`break`/`continue`, or - inside a try/catch's recover() closure whose body
	 * needs to escape it (see README) - the flag-setting form that a post-closure check re-plays. */
	String returnOrEscape(int indent, String exprTextOrNull) {
		if (emitter.currentEscapeReturnedFlag == null) {
			return ind(indent) + "return" + (exprTextOrNull == null ? "" : " " + exprTextOrNull) + "\n";
		}
		StringBuilder b = new StringBuilder();
		if (exprTextOrNull != null && emitter.currentEscapeRetVar != null) {
			b.append(ind(indent)).append(emitter.currentEscapeRetVar).append(" = ").append(exprTextOrNull).append('\n');
		}
		b.append(ind(indent)).append(emitter.currentEscapeReturnedFlag).append(" = true\n");
		b.append(ind(indent)).append("return\n");
		return b.toString();
	}

	String breakOrEscape(int indent) {
		if (emitter.currentEscapeBrokeFlag != null && emitter.loopSwitchDepth == 0) {
			return ind(indent) + emitter.currentEscapeBrokeFlag + " = true\n" + ind(indent) + "return\n";
		}
		return ind(indent) + "break\n";
	}

	String continueOrEscape(int indent) {
		if (emitter.currentEscapeContinuedFlag != null && emitter.loopSwitchDepth == 0) {
			return ind(indent) + emitter.currentEscapeContinuedFlag + " = true\n" + ind(indent) + "return\n";
		}
		return ind(indent) + "continue\n";
	}

	private String unsupportedStmt(Statement s, int indent) {
		String oneLine = s.toString().replace("\n", " ").trim();
		emitter.unsupported.add(s.getClass().getSimpleName() + ": " + oneLine);
		return ind(indent) + "panic(\"j2go: unsupported " + s.getClass().getSimpleName() + "\") // TODO(gowt-port): " + oneLine + "\n";
	}
}
