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
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		String main = emitStatementInner(s, indent);
		StringBuilder b = new StringBuilder();
		for (String p : emitter.prelude) b.append(ind(indent)).append(p).append('\n');
		b.append(main);
		emitter.prelude = saved;
		return b.toString();
	}

	private String emitStatementInner(Statement s, int indent) {
		if (s instanceof ExpressionStatement es) return emitExpressionStatement(es, indent);
		if (s instanceof ReturnStatement rs) return emitReturn(rs, indent);
		if (s instanceof IfStatement is) return emitIf(is, indent);
		if (s instanceof VariableDeclarationStatement vds) return emitVarDecl(vds, indent);
		if (s instanceof Block b) return ind(indent) + "{\n" + emitBlockBody(b, indent + 1) + ind(indent) + "}\n";
		if (s instanceof SwitchStatement sw) return emitter.emitSwitchStatement(sw, indent);
		if (s instanceof ThrowStatement ts) return emitter.emitThrow(ts, indent);
		if (s instanceof TryStatement ts) return emitter.emitTry(ts, indent);
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

	private String emitExpressionStatement(ExpressionStatement es, int indent) {
		Expression e = es.getExpression();
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
			return ind(indent) + lhs + " " + a.getOperator().toString() + " " + rhs + "\n";
		}
		// x++;/--x; as their own statement: Go's native x++/x-- directly, no throwaway temp
		// (the generic emitExpr path below would leave a bare, unused-value statement).
		if (e instanceof PostfixExpression pf) return ind(indent) + emitter.expr(pf.getOperand()) + pf.getOperator().toString() + "\n";
		if (e instanceof PrefixExpression pf && (pf.getOperator().toString().equals("++") || pf.getOperator().toString().equals("--"))) {
			return ind(indent) + emitter.expr(pf.getOperand()) + pf.getOperator().toString() + "\n";
		}
		return ind(indent) + emitter.expr(e) + "\n";
	}

	private String emitChainedAssignment(Assignment a, int indent) {
		List<Expression> targets = new ArrayList<>();
		Expression cur = a;
		while (cur instanceof Assignment ca && ca.getOperator() == Assignment.Operator.ASSIGN) {
			targets.add(ca.getLeftHandSide());
			cur = ca.getRightHandSide();
		}
		String rhs = emitter.expr(cur);
		StringBuilder b = new StringBuilder();
		String prev = rhs;
		for (int i = targets.size() - 1; i >= 0; i--) {
			String lhsText = emitter.expr(targets.get(i));
			b.append(ind(indent)).append(lhsText).append(" = ").append(prev).append('\n');
			prev = lhsText;
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
		b.append(ind(indent + 1)).append(lhsText).append(" = ").append(thenText).append('\n');
		b.append(ind(indent)).append("} else {\n");
		b.append(ind(indent + 1)).append(lhsText).append(" = ").append(elseText).append('\n');
		b.append(ind(indent)).append("}\n");
		return b.toString();
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
			return lhs + " " + a.getOperator().toString() + " " + rhs;
		}
		return emitter.expr(e);
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
			String condText = fs.getExpression() == null ? "" : emitter.expr(fs.getExpression());
			String updText = updaters.isEmpty() ? "" : exprAsSimpleStmt((Expression) updaters.get(0));
			b.append(ind(indent)).append("for ").append(initText).append("; ").append(condText).append("; ").append(updText).append(" {\n");
			emitter.loopSwitchDepth++;
			b.append(emitAsBlock(fs.getBody(), indent + 1));
			emitter.loopSwitchDepth--;
			b.append(ind(indent)).append("}\n");
			return b.toString();
		}
		// General form (0/multiple initializers or updaters): hoist inits before the loop,
		// append updaters at the end of the body - always correct, just less idiomatic.
		for (Object o : inits) {
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
		String condText = fs.getExpression() == null ? "" : emitExprInto(fs.getExpression(), b, indent);
		b.append(ind(indent)).append("for ").append(condText).append(" {\n");
		emitter.loopSwitchDepth++;
		b.append(emitAsBlock(fs.getBody(), indent + 1));
		emitter.loopSwitchDepth--;
		for (Object o : updaters) b.append(ind(indent + 1)).append(exprAsSimpleStmt((Expression) o)).append('\n');
		b.append(ind(indent)).append("}\n");
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
		StringBuilder b = new StringBuilder();
		String cond = emitExprInto(ws.getExpression(), b, indent);
		b.append(ind(indent)).append("for ").append(cond).append(" {\n");
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
