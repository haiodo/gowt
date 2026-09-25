package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.Map;

import static dev.gowt.j2go.emit.EmitUtil.*;

/** Infix operators, numeric widening/adaptation, object upcasting, zero values - split out of
 * ExpressionEmitter (same file, moved verbatim) to keep it under the line budget. */
final class NumericEmitter {

	private final Emitter emitter;

	NumericEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	private static final Map<String, Integer> NUMERIC_RANK = Map.of(
			"int8", 1, "int16", 2, "uint16", 2, "int32", 3, "int64", 4, "float32", 5, "float64", 6);

	String emitInfix(InfixExpression ie) {
		InfixExpression.Operator op = ie.getOperator();
		ITypeBinding resultType = ie.resolveTypeBinding();
		if (op == InfixExpression.Operator.PLUS && resultType != null
				&& resultType.getQualifiedName().equals("java.lang.String")) {
			return emitStringConcat(ie);
		}
		// Go rejects `1 << 31` as an overflowing int32 constant - fold it to the decimal instead.
		if (op == InfixExpression.Operator.LEFT_SHIFT
				&& ie.getLeftOperand() instanceof NumberLiteral ln && ie.getRightOperand() instanceof NumberLiteral rn) {
			String folded = foldShift(ln, rn, resultType);
			if (folded != null) return folded;
		}
		String paramNullCheck = stringParamNullCheck(ie, op);
		if (paramNullCheck != null) return paramNullCheck;
		String goOp = goOperator(ie);
		String hoisted = hoistBooleanChainIfNeeded(ie, op, goOp);
		if (hoisted != null) return hoisted;
		String left = parenthesize(ie.getLeftOperand(), emitter.expr(ie.getLeftOperand()), goOp, false);
		String right = parenthesize(ie.getRightOperand(), emitter.expr(ie.getRightOperand()), goOp, true);
		if (op == InfixExpression.Operator.RIGHT_SHIFT_UNSIGNED && ie.extendedOperands().isEmpty()) {
			return unsignedShift(left, resultType, right);
		}
		// x == null / x != null where x is a String: null was translated to "" (see adaptNumeric),
		// so the comparison must be against "" too - Go's string has no nil to compare against.
		if (right.equals("nil") && isGoString(ie.getLeftOperand())) right = "\"\"";
		if (left.equals("nil") && isGoString(ie.getRightOperand())) left = "\"\"";
		// Same for a cocoa struct (NSPoint): null is stored as its zero value (adaptNumeric).
		// Parenthesized: a bare composite literal in an `if` condition is a Go parse error.
		if (right.equals("nil")) right = "(" + emitter.adaptNumeric(right, null, ie.getLeftOperand().resolveTypeBinding()) + ")";
		if (left.equals("nil")) left = "(" + emitter.adaptNumeric(left, null, ie.getRightOperand().resolveTypeBinding()) + ")";
		if (op == InfixExpression.Operator.EQUALS || op == InfixExpression.Operator.NOT_EQUALS) {
			ITypeBinding lt = ie.getLeftOperand().resolveTypeBinding();
			ITypeBinding rt = ie.getRightOperand().resolveTypeBinding();
			// Java array == is reference identity, but Go slices only compare to nil - the identity
			// check is a fast path anyway, so a full slices.Equal is a harmless replacement.
			if (lt != null && lt.isArray() && rt != null && rt.isArray()) {
				emitter.fileImports.add("slices");
				String call = "slices.Equal(" + left + ", " + right + ")";
				return op == InfixExpression.Operator.EQUALS ? call : "!" + call;
			}
			// x == y / x != y across two related object types (this == shell, *Control vs *Shell):
			// Go's == needs identical types, so whichever side is the narrower one gets upcast.
			if (lt != null && rt != null && !lt.isPrimitive() && !rt.isPrimitive()) {
				String upLeft = upcastObject(left, lt, rt);
				boolean upcastApplied = !upLeft.equals(left);
				if (upcastApplied) {
					left = upLeft;
				} else {
					String upRight = upcastObject(right, rt, lt);
					upcastApplied = !upRight.equals(right);
					if (upcastApplied) right = upRight;
				}
				// Sibling types (Button vs Label): upcastObject can't reconcile them - identityCompare
				// instead, unless one side is already plain `any` (already compiles as-is).
				boolean nullCompare = ie.getLeftOperand() instanceof NullLiteral || ie.getRightOperand() instanceof NullLiteral;
				String goL = dev.gowt.j2go.GoTypes.map(lt, emitter), goR = dev.gowt.j2go.GoTypes.map(rt, emitter);
				if (!upcastApplied && !nullCompare && !goL.equals(goR) && !goL.equals("any") && !goR.equals("any")) {
					String identity = identityCompare(left, right, lt, rt);
					if (identity != null) return op == InfixExpression.Operator.EQUALS ? identity : "!" + identity;
				}
			}
		}
		StringBuilder b = new StringBuilder();
		// A 3+ operand chain (a+b+c): every operand needs the shared result type, not just a pair.
		if (!ie.extendedOperands().isEmpty() && isArithmeticOrBitwise(op) && resultType != null) {
			b.append(emitter.adaptNumeric(left, ie.getLeftOperand().resolveTypeBinding(), resultType))
					.append(' ').append(goOp).append(' ')
					.append(emitter.adaptNumeric(right, ie.getRightOperand().resolveTypeBinding(), resultType));
			for (Object ext : ie.extendedOperands()) {
				Expression e = (Expression) ext;
				String r = emitter.adaptNumeric(parenthesize(e, emitter.expr(e), goOp, true), e.resolveTypeBinding(), resultType);
				b.append(' ').append(goOp).append(' ').append(r);
			}
			return b.toString();
		}
		String[] adapted = adaptBinaryOperands(left, right, ie.getLeftOperand(), ie.getRightOperand());
		b.append(adapted[0]).append(' ').append(goOp).append(' ').append(adapted[1]);
		for (Object ext : ie.extendedOperands()) {
			String r = parenthesize((Expression) ext, emitter.expr((Expression) ext), goOp, true);
			b.append(' ').append(goOp).append(' ').append(r);
		}
		return b.toString();
	}

	/** Java's &/| always evaluates every operand; Go's &&/|| stops early. Only a side effect past
	 * the first operand needs hoisting - each operand into its own temp, then combine with &&/||. */
	private String hoistBooleanChainIfNeeded(InfixExpression ie, InfixExpression.Operator op, String goOp) {
		if (op != InfixExpression.Operator.AND && op != InfixExpression.Operator.OR) return null;
		if (!goOp.equals("&&") && !goOp.equals("||")) return null; // int &/|, not boolean: unaffected
		java.util.List<Expression> operands = new java.util.ArrayList<>();
		operands.add(ie.getLeftOperand());
		operands.add(ie.getRightOperand());
		for (Object o : ie.extendedOperands()) operands.add((Expression) o);
		boolean needsHoist = false;
		for (int i = 1; i < operands.size() && !needsHoist; i++) needsHoist = hasSideEffect(operands.get(i));
		if (!needsHoist) return null;
		java.util.List<String> temps = new java.util.ArrayList<>();
		for (Expression e : operands) {
			String tmp = "b" + (++emitter.tempCounter);
			emitter.prelude.add(tmp + " := " + emitter.expr(e));
			temps.add(tmp);
		}
		return String.join(" " + goOp + " ", temps);
	}

	/** True if e can have a side effect Go's &&/|| would wrongly skip as a boolean &/|'s non-first
	 * operand: a call, `new`, an assignment, or ++/--. */
	boolean hasSideEffect(Expression e) {
		boolean[] found = {false};
		e.accept(new ASTVisitor() {
			@Override public boolean visit(MethodInvocation node) { return stop(); }
			@Override public boolean visit(SuperMethodInvocation node) { return stop(); }
			@Override public boolean visit(ClassInstanceCreation node) { return stop(); }
			@Override public boolean visit(Assignment node) { return stop(); }
			@Override public boolean visit(PrefixExpression node) {
				String op = node.getOperator().toString();
				return op.equals("++") || op.equals("--") ? stop() : true;
			}
			@Override public boolean visit(PostfixExpression node) { return stop(); }
			@Override public boolean visit(QualifiedName node) {
				return emitter.selectorStringOf(node) != null ? stop() : true;
			}
			private boolean stop() { found[0] = true; return false; }
		});
		return found[0];
	}

	// Java's boolean &, | and ^ have no Go bool operator: &&, || (short-circuiting - a ceiling
	// only when the right operand has side effects, see hoistBooleanChainIfNeeded) and !=.
	private static String goOperator(InfixExpression ie) {
		String op = ie.getOperator().toString();
		ITypeBinding lt = ie.getLeftOperand().resolveTypeBinding();
		if (lt == null || !lt.getName().equals("boolean")) return op;
		return switch (op) {
			case "&" -> "&&";
			case "|" -> "||";
			case "^" -> "!=";
			default -> op;
		};
	}

	// Go ranks & with * and | ^ with +, both above comparisons; Java ranks them below. A nested
	// infix operand keeps Java's grouping only if parenthesized wherever Go would regroup it.
	private String parenthesize(Expression operand, String text, String parentGoOp, boolean rightSide) {
		if (!(operand instanceof InfixExpression child)) return text;
		int c = goPrecedence(goOperator(child));
		int p = goPrecedence(parentGoOp);
		return c < p || c == p && rightSide ? "(" + text + ")" : text;
	}

	private static int goPrecedence(String goOp) {
		return switch (goOp) {
			case "*", "/", "%", "<<", ">>", "&", "&^" -> 5;
			case "+", "-", "|", "^" -> 4;
			case "==", "!=", "<", "<=", ">", ">=" -> 3;
			case "&&" -> 2;
			case "||" -> 1;
			default -> 6;
		};
	}

	// Result type equals the promoted operand type only for these; relational/logical ops always
	// resolve to boolean regardless of operand types.
	private boolean isArithmeticOrBitwise(InfixExpression.Operator op) {
		return op == InfixExpression.Operator.PLUS || op == InfixExpression.Operator.MINUS
				|| op == InfixExpression.Operator.TIMES || op == InfixExpression.Operator.DIVIDE
				|| op == InfixExpression.Operator.REMAINDER || op == InfixExpression.Operator.AND
				|| op == InfixExpression.Operator.OR || op == InfixExpression.Operator.XOR;
	}

	/** `if (param == null) error(SWT.ERROR_NULL_ARGUMENT)` (or a throw) on a String parameter: the
	 * check becomes constant false (`!=`: true). A Go caller cannot pass null, so it must not fire
	 * for "" (README "Round 11 null-string"). Every other String null check keeps `== ""`. */
	private String stringParamNullCheck(InfixExpression ie, InfixExpression.Operator op) {
		if (op != InfixExpression.Operator.EQUALS && op != InfixExpression.Operator.NOT_EQUALS) return null;
		Expression l = ie.getLeftOperand(), r = ie.getRightOperand();
		Expression other = r instanceof NullLiteral ? l : l instanceof NullLiteral ? r : null;
		if (!(other instanceof SimpleName n) || !(n.resolveBinding() instanceof IVariableBinding vb)) return null;
		if (!vb.isParameter() || !isGoString(other) || !isNullArgumentGuard(ie)) return null;
		return op == InfixExpression.Operator.EQUALS ? "false" : "true";
	}

	// e is the condition of an if (alone or inside a || chain) whose then-branch is
	// error(...ERROR_NULL_ARGUMENT) or a throw.
	private static boolean isNullArgumentGuard(Expression e) {
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
				&& mi.arguments().get(0) instanceof Name arg && arg.getFullyQualifiedName().endsWith("ERROR_NULL_ARGUMENT");
	}

	private boolean isGoString(Expression e) {
		ITypeBinding t = e.resolveTypeBinding();
		return t != null && dev.gowt.j2go.GoTypes.map(t, emitter).equals("string");
	}

	/** Folds a literal << literal to its Java-shift decimal result, or null if not foldable. */
	private String foldShift(NumberLiteral base, NumberLiteral shiftLit, ITypeBinding resultType) {
		String goType = resultType == null ? null : dev.gowt.j2go.GoTypes.map(resultType, emitter);
		if (!"int32".equals(goType) && !"int64".equals(goType)) return null;
		long b, s;
		try {
			b = Long.decode(stripNumericSuffix(base.getToken()));
			s = Long.decode(stripNumericSuffix(shiftLit.getToken()));
		} catch (NumberFormatException e) {
			return null;
		}
		if (goType.equals("int32")) return Integer.toString((int) b << (s & 31));
		return Long.toString(b << (s & 63));
	}

	private String[] adaptBinaryOperands(String leftText, String rightText, Expression left, Expression right) {
		String lt = goPrimitiveOf(left.resolveTypeBinding());
		String rt = goPrimitiveOf(right.resolveTypeBinding());
		if (lt == null || rt == null) return new String[]{leftText, rightText};
		// Java promotes byte/short/char operands to int; Go keeps int8 (byte & 0xFF won't compile).
		if (NUMERIC_RANK.get(lt) < 3 && !isLiteral(left)) leftText = (lt = "int32") + "(" + leftText + ")";
		if (NUMERIC_RANK.get(rt) < 3 && !isLiteral(right)) rightText = (rt = "int32") + "(" + rightText + ")";
		if (lt.equals(rt)) return new String[]{leftText, rightText};
		Integer lr = NUMERIC_RANK.get(lt);
		Integer rr = NUMERIC_RANK.get(rt);
		if (lr == null || rr == null) return new String[]{leftText, rightText};
		if (lr < rr && !isLiteral(left)) return new String[]{rt + "(" + leftText + ")", rightText};
		if (rr < lr && !isLiteral(right)) return new String[]{leftText, lt + "(" + rightText + ")"};
		return new String[]{leftText, rightText};
	}

	private boolean isLiteral(Expression e) {
		if (e instanceof NumberLiteral) return true;
		if (e instanceof org.eclipse.jdt.core.dom.PrefixExpression pf) return isLiteral(pf.getOperand());
		return false;
	}

	private String goPrimitiveOf(ITypeBinding t) {
		if (t == null || !t.isPrimitive()) return null;
		String g = dev.gowt.j2go.GoTypes.map(t, emitter);
		return NUMERIC_RANK.containsKey(g) ? g : null;
	}

	private String emitStringConcat(InfixExpression ie) {
		emitter.fileImports.add("fmt");
		java.util.List<Expression> operands = new java.util.ArrayList<>();
		operands.add(ie.getLeftOperand());
		operands.add(ie.getRightOperand());
		for (Object o : ie.extendedOperands()) operands.add((Expression) o);
		StringBuilder fmt = new StringBuilder();
		java.util.List<String> args = new java.util.ArrayList<>();
		for (Expression op : operands) {
			if (op instanceof StringLiteral sl) {
				fmt.append(escapeForFormat(sl.getLiteralValue()));
			} else {
				fmt.append(verbFor(op.resolveTypeBinding()));
				args.add(emitter.expr(op));
			}
		}
		StringBuilder b = new StringBuilder();
		b.append("fmt.Sprintf(\"").append(fmt).append('"');
		for (String a : args) b.append(", ").append(a);
		b.append(')');
		return b.toString();
	}

	private String verbFor(ITypeBinding t) {
		if (t == null) return "%v";
		String g = dev.gowt.j2go.GoTypes.map(t, emitter);
		return switch (g) {
			case "int8", "int16", "uint16", "int32", "int64" -> "%d";
			case "float32", "float64" -> "%v";
			case "string" -> "%s";
			case "bool" -> "%t";
			default -> "%v";
		};
	}

	/** Java's >>> on an (int-promoted) operand: Go shifts unsigned types logically. */
	String unsignedShift(String operand, ITypeBinding type, String count) {
		String g = dev.gowt.j2go.GoTypes.map(type, emitter);
		String t = g.equals("int64") ? "int64" : "int32";
		return t + "(u" + t + "(" + operand + ") >> (" + count + "))";
	}

	// ---------------------------------------------------------------- numeric adaptation

	String adaptNumeric(String text, ITypeBinding from, ITypeBinding to) {
		// A Java `null` passed where a String is expected: Go's string has no nil value, so the
		// String mapping uses "" as null's stand-in (see also emitInfix's ==/!= null case).
		if (text.equals("nil") && to != null && dev.gowt.j2go.GoTypes.map(to, emitter).equals("string")) return "\"\"";
		// Same idea for a struct-valued target (cocoa.NSSize, ...): Go structs have no nil either.
		if (text.equals("nil") && to != null) {
			TypeModel.ClassInfo toCi = emitter.model.lookup(to);
			if (toCi != null && toCi.isStruct) return emitter.qualifiedTypeName(toCi) + "{}";
		}
		// A manual int-backed enum (Display.APPEARANCE, RoundingMode, ...) has no Go nil - falls
		// back to its zero value, collapsing "never set" into the first enum constant.
		if (text.equals("nil") && to != null) {
			String q = to.getErasure().getQualifiedName();
			String goType = Manual.goTypeName(q);
			boolean bareEnum = Manual.isValueType(q) && !goType.contains(".")
					&& !goType.equals("any") && !goType.equals("error") && !goType.equals("bool");
			if (bareEnum) return goType + "(0)";
		}
		if (from == null || to == null) return text;
		if (!from.isPrimitive() || !to.isPrimitive()) return upcastObject(text, from, to);
		String fromGo = dev.gowt.j2go.GoTypes.map(from, emitter);
		String toGo = dev.gowt.j2go.GoTypes.map(to, emitter);
		if (fromGo.equals(toGo) || fromGo.equals("bool") || toGo.equals("bool")) return text;
		return toGo + "(" + text + ")";
	}

	/** Go has no covariant object assignment: a value whose static type is a proper descendant of
	 * the target (real or manual chain, e.g. `control = control.parent`) needs an explicit upcast. */
	String upcastObject(String text, ITypeBinding from, ITypeBinding to) {
		if (from.isPrimitive() || to.isPrimitive()) return text;
		String fromGo = dev.gowt.j2go.GoTypes.map(from, emitter);
		String toGo = dev.gowt.j2go.GoTypes.map(to, emitter);
		if (fromGo.equals(toGo)) return text;
		String toQualified = to.getErasure().getQualifiedName();
		if (!isProperDescendant(from.getErasure().getQualifiedName(), toQualified)) return text;
		// cocoa's id: its embedded-field selector name (x.id) is unexported and cross-package-
		// unreachable no matter how it's spelled - id_manual.go's AsId() method works around it.
		String path = toQualified.equals("org.eclipse.swt.internal.cocoa.id") ? "x.AsId()" : null;
		String toBare = bareGoName(toQualified);
		if (path == null && toBare == null) return text;
		if (path == null) path = "&x." + toBare;
		return ensureUpcastHelper(fromGo, toGo, path) + "(" + text + ")";
	}

	/** .Impl() always normalizes to the concrete leaf pointer regardless of hierarchy shape -
	 * any-boxed so two different roots' differently-typed Impl() interfaces still compile. */
	private String identityCompare(String left, String right, ITypeBinding lt, ITypeBinding rt) {
		boolean lHas = hasImpl(lt);
		boolean rHas = hasImpl(rt);
		if (!lHas && !rHas) return null;
		String l = lHas ? left + ".Impl()" : left;
		String r = rHas ? right + ".Impl()" : right;
		return "(any(" + l + ") == any(" + r + "))";
	}

	private boolean hasImpl(ITypeBinding t) {
		TypeModel.ClassInfo ci = t == null ? null : emitter.model.lookup(t);
		return ci != null && ci.root.splitsDispatch() && !ci.root.children.isEmpty();
	}

	/** Java upcasts null to null; `&x.Base` on a nil x panics - so each upcast is a nil-checking
	 * helper, generated once per (from, to) pair. */
	private String ensureUpcastHelper(String fromGo, String toGo, String path) {
		String name = "upcast" + fromGo.replaceAll("[*.]", "") + "To" + toGo.replaceAll("[*.]", "");
		if (emitter.generatedHelpers.add(name)) {
			emitter.fileHelperSource.add("func " + name + "(x " + fromGo + ") " + toGo + " {\n\tif x == nil {\n\t\treturn nil\n\t}\n\treturn "
					+ path + "\n}\n\n");
		}
		return name;
	}

	private TypeModel.ClassInfo findByQualifiedName(String qualified) {
		for (TypeModel.ClassInfo c : emitter.model.all()) {
			if (c.binding.getErasure().getQualifiedName().equals(qualified)) return c;
		}
		return null;
	}

	private String bareGoName(String qualified) {
		TypeModel.ClassInfo c = findByQualifiedName(qualified);
		if (c != null) return c.goTypeName;
		return Manual.isManual(qualified) ? Manual.goTypeName(qualified) : null;
	}

	private boolean isProperDescendant(String fromQualified, String toQualified) {
		String cur = superQualifiedOf(fromQualified);
		for (int i = 0; i < 12 && cur != null; i++) {
			if (cur.equals(toQualified)) return true;
			cur = superQualifiedOf(cur);
		}
		return false;
	}

	private String superQualifiedOf(String qualified) {
		TypeModel.ClassInfo c = findByQualifiedName(qualified);
		if (c != null) return c.superclass == null ? null : c.superclass.binding.getErasure().getQualifiedName();
		return Manual.manualSuperclassOf(qualified);
	}

	String zeroValue(ITypeBinding t) {
		if (t.isPrimitive()) {
			return switch (t.getName()) {
				case "boolean" -> "false";
				case "float", "double" -> "0";
				default -> "0";
			};
		}
		TypeModel.ClassInfo ci = emitter.model.lookup(t);
		if (ci != null && ci.isStruct) return emitter.qualifiedTypeName(ci) + "{}";
		if (t.getQualifiedName().equals("java.lang.String")) return "\"\"";
		return "nil";
	}
}
