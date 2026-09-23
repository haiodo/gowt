package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.Map;

import static dev.gowt.j2go.emit.EmitUtil.*;

/** Expression dispatch, prefix/postfix, assignment, infix, numeric adaptation, literals, names
 * and field access, static field refs, conditional hoisting, array creation, zero values. */
final class ExpressionEmitter {

	private final Emitter emitter;

	ExpressionEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- expressions

	String emitExpr(Expression e) {
		if (e instanceof NumberLiteral nl) return stripNumericSuffix(nl.getToken());
		if (e instanceof CharacterLiteral cl) return cl.getEscapedValue(); // Go rune literal syntax matches Java's here
		if (e instanceof BooleanLiteral bl) return Boolean.toString(bl.booleanValue());
		if (e instanceof StringLiteral sl) return goStringLiteral(sl.getLiteralValue());
		if (e instanceof NullLiteral) return "nil";
		if (e instanceof ThisExpression) return "this";
		if (e instanceof ParenthesizedExpression pe) return "(" + emitExpr(pe.getExpression()) + ")";
		if (e instanceof PrefixExpression pf) return emitPrefix(pf);
		if (e instanceof PostfixExpression pf) return emitPostfix(pf);
		if (e instanceof CastExpression ce) return emitter.emitCast(ce);
		if (e instanceof InfixExpression ie) return emitInfix(ie);
		if (e instanceof Assignment a) return emitInlineAssign(a);
		if (e instanceof SimpleName sn) return emitSimpleName(sn);
		if (e instanceof FieldAccess fa) return emitFieldAccess(fa);
		if (e instanceof QualifiedName qn) return emitQualifiedName(qn);
		if (e instanceof MethodInvocation mi) return emitter.emitMethodInvocation(mi);
		if (e instanceof SuperMethodInvocation smi) return emitter.emitSuperMethodInvocation(smi);
		if (e instanceof ClassInstanceCreation cic) return emitter.emitNew(cic);
		if (e instanceof ConditionalExpression ce) return emitConditionalHoisted(ce);
		if (e instanceof InstanceofExpression ioe) return emitter.emitPlainInstanceof(ioe);
		if (e instanceof PatternInstanceofExpression pie) return emitter.emitPatternInstanceof(pie);
		if (e instanceof ArrayCreation ac) return emitArrayCreation(ac);
		if (e instanceof ArrayInitializer ai) return emitArrayInitializer(ai, ai.resolveTypeBinding());
		if (e instanceof ArrayAccess aa) return emitExpr(aa.getArray()) + "[" + emitExpr(aa.getIndex()) + "]";
		if (e instanceof SwitchExpression) {
			emitter.unsupported.add("SwitchExpression: used outside assignment lowering: " + e);
			return panicClosure(e, "unsupported SwitchExpression");
		}
		String oneLine = e.toString().replace("\n", " ").trim();
		emitter.unsupported.add(e.getClass().getSimpleName() + ": " + oneLine);
		return panicClosure(e, "unsupported " + e.getClass().getSimpleName()) + " /* TODO(gowt-port): " + oneLine + " */";
	}

	// Closure typed to e's own resolved type, so it type-checks wherever e's text lands.
	String panicClosure(Expression e, String message) {
		ITypeBinding t = e.resolveTypeBinding();
		String goType = t == null ? "" : dev.gowt.j2go.GoTypes.map(t, emitter);
		return panicClosureTyped(goType, message);
	}

	String panicClosureTyped(String goType, String message) {
		if (goType.isEmpty() || goType.startsWith("unsupported_")) goType = "any";
		return "func() " + goType + " { panic(\"j2go: " + message + "\") }()";
	}

	private String emitInlineAssign(Assignment a) {
		// nested assignment outside the statement-level chain handler: hoist to prelude.
		String tmp = "cond" + (++emitter.tempCounter);
		String rhs = adaptNumeric(emitExpr(a.getRightHandSide()), a.getRightHandSide().resolveTypeBinding(),
				a.getLeftHandSide().resolveTypeBinding());
		emitter.prelude.add(tmp + " := " + rhs);
		emitter.prelude.add(emitExpr(a.getLeftHandSide()) + " = " + tmp);
		return tmp;
	}

	private String emitPrefix(PrefixExpression pf) {
		String op = pf.getOperator().toString();
		if (op.equals("-") || op.equals("!")) return op + emitExpr(pf.getOperand());
		if (op.equals("~")) return "^" + emitExpr(pf.getOperand()); // Java's bitwise complement is Go's ^x
		if (op.equals("++") || op.equals("--")) {
			// Go has no pre-increment expression: bump first, then read the now-new value.
			String operand = emitExpr(pf.getOperand());
			emitter.prelude.add(operand + op);
			return operand;
		}
		emitter.unsupported.add("PrefixExpression: " + pf);
		return panicClosure(pf, "unsupported PrefixExpression " + pf);
	}

	private String emitPostfix(PostfixExpression pf) {
		// Go has no post-increment expression: stash the old value, bump, hand back the stash.
		String operand = emitExpr(pf.getOperand());
		String tmp = "t" + (++emitter.tempCounter);
		emitter.prelude.add(tmp + " := " + operand);
		emitter.prelude.add(operand + pf.getOperator().toString());
		return tmp;
	}

	private static final Map<String, Integer> NUMERIC_RANK = Map.of(
			"int8", 1, "int16", 2, "uint16", 2, "int32", 3, "int64", 4, "float32", 5, "float64", 6);

	private String emitInfix(InfixExpression ie) {
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
		String left = emitExpr(ie.getLeftOperand());
		String right = emitExpr(ie.getRightOperand());
		// x == null / x != null where x is a String: null was translated to "" (see adaptNumeric),
		// so the comparison must be against "" too - Go's string has no nil to compare against.
		if (right.equals("nil") && isGoString(ie.getLeftOperand())) right = "\"\"";
		if (left.equals("nil") && isGoString(ie.getRightOperand())) left = "\"\"";
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
				if (!upLeft.equals(left)) left = upLeft;
				else right = upcastObject(right, rt, lt);
			}
		}
		String[] adapted = adaptBinaryOperands(left, right, ie.getLeftOperand(), ie.getRightOperand());
		String goOp = op == InfixExpression.Operator.XOR ? "^" : op.toString();
		StringBuilder b = new StringBuilder();
		b.append(adapted[0]).append(' ').append(goOp).append(' ').append(adapted[1]);
		for (Object ext : ie.extendedOperands()) {
			String r = emitExpr((Expression) ext);
			b.append(' ').append(goOp).append(' ').append(r);
		}
		return b.toString();
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
		if (lt == null || rt == null || lt.equals(rt)) return new String[]{leftText, rightText};
		Integer lr = NUMERIC_RANK.get(lt);
		Integer rr = NUMERIC_RANK.get(rt);
		if (lr == null || rr == null) return new String[]{leftText, rightText};
		if (lr < rr && !isLiteral(left)) return new String[]{rt + "(" + leftText + ")", rightText};
		if (rr < lr && !isLiteral(right)) return new String[]{leftText, lt + "(" + rightText + ")"};
		return new String[]{leftText, rightText};
	}

	private boolean isLiteral(Expression e) {
		if (e instanceof NumberLiteral) return true;
		if (e instanceof PrefixExpression pf) return isLiteral(pf.getOperand());
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
				args.add(emitExpr(op));
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

	private String emitSimpleName(SimpleName sn) {
		IBinding b = sn.resolveBinding();
		if (b instanceof IVariableBinding vb && vb.isField()) {
			if (Modifier.isStatic(vb.getModifiers())) return staticFieldRef(vb);
			return "this." + fieldGoName(vb);
		}
		return emitter.sanitizeIdent(sn.getIdentifier());
	}

	String fieldGoName(IVariableBinding vb) {
		String n = vb.getName();
		if (Modifier.isPublic(vb.getModifiers())) return Names.capitalize(n);
		return n;
	}

	/** Java's array.length is a pseudo-field with no Go equivalent syntax; len(x) replaces it. */
	private boolean isArrayLength(Expression qualifier, String memberName) {
		if (!memberName.equals("length")) return false;
		ITypeBinding t = qualifier.resolveTypeBinding();
		return t != null && t.isArray();
	}

	private String emitFieldAccess(FieldAccess fa) {
		if (isArrayLength(fa.getExpression(), fa.getName().getIdentifier())) {
			return "int32(len(" + emitExpr(fa.getExpression()) + "))"; // int32: Java's length is int
		}
		IVariableBinding vb = fa.resolveFieldBinding();
		String recv = emitExpr(fa.getExpression());
		return recv + "." + fieldGoName(vb);
	}

	/** The selector string for a `Selector.sel_x.value`-shaped QualifiedName, or null. */
	String selectorStringOf(QualifiedName qn) {
		if (!qn.getName().getIdentifier().equals("value")) return null;
		if (!(qn.getQualifier() instanceof QualifiedName inner)) return null;
		IBinding innerBinding = inner.resolveBinding();
		if (!(innerBinding instanceof IVariableBinding enumConst) || !enumConst.isEnumConstant()) return null;
		return emitter.selectors.stringFor(inner.getName().getIdentifier());
	}

	/** OS.java's ~1360 `sel_x = Selector.sel_x.value;`: Selector itself is elided (see README). */
	private String selectorValueRewrite(QualifiedName qn) {
		String selectorString = selectorStringOf(qn);
		if (selectorString == null) return null;
		emitter.fileImports.add("sync");
		emitter.fileImports.add("github.com/ebitengine/purego");
		return "OSSel_registerName(" + goStringLiteral(selectorString) + ")";
	}

	private String emitQualifiedName(QualifiedName qn) {
		if (isArrayLength(qn.getQualifier(), qn.getName().getIdentifier())) {
			return "int32(len(" + emitExpr(qn.getQualifier()) + "))";
		}
		String selRewrite = selectorValueRewrite(qn);
		if (selRewrite != null) return selRewrite;
		IBinding b = qn.resolveBinding();
		if (b instanceof IVariableBinding vb && vb.isField()) {
			if (Modifier.isStatic(vb.getModifiers())) {
				return staticFieldRef(vb);
			}
			String recv = emitExpr(qn.getQualifier());
			return recv + "." + fieldGoName(vb);
		}
		// A type or package qualifier (Foo.Bar as a type name, not a value): just the bare name.
		return qn.getName().getIdentifier();
	}

	private String staticFieldRef(IVariableBinding vb) {
		ITypeBinding declaring = vb.getDeclaringClass();
		String qualified = declaring.getErasure().getQualifiedName();
		if (Manual.isManual(qualified)) return Manual.staticMember(qualified, vb.getName());
		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (ci != null) {
			String goName = emitter.qualifiedFuncPrefix(ci) + Names.capitalize(vb.getName());
			return staticFieldClashesWithMethod(declaring, vb.getName()) ? goName + "_" : goName;
		}
		emitter.unsupported.add("StaticField: unresolved declaring type for " + vb.getName());
		return vb.getName();
	}

	/** True if any call appears anywhere in e (own AST subtree only, not into other methods). */
	boolean containsCall(Expression e) {
		boolean[] found = {false};
		e.accept(new ASTVisitor() {
			@Override
			public boolean visit(MethodInvocation node) {
				found[0] = true;
				return false;
			}

			@Override
			public boolean visit(QualifiedName node) {
				if (selectorStringOf(node) != null) found[0] = true;
				return false;
			}
		});
		return found[0];
	}

	// ---------------------------------------------------------------- conditional

	private String emitConditionalHoisted(ConditionalExpression ce) {
		String tmp = "cond" + (++emitter.tempCounter);
		ITypeBinding target = ce.resolveTypeBinding();
		String goType = dev.gowt.j2go.GoTypes.map(target, emitter);
		emitter.prelude.add("var " + tmp + " " + goType);
		String cond = emitExpr(ce.getExpression());
		String thenText = adaptNumeric(emitExpr(ce.getThenExpression()), ce.getThenExpression().resolveTypeBinding(), target);
		String elseText = adaptNumeric(emitExpr(ce.getElseExpression()), ce.getElseExpression().resolveTypeBinding(), target);
		emitter.prelude.add("if " + cond + " {");
		emitter.prelude.add("\t" + tmp + " = " + thenText);
		emitter.prelude.add("} else {");
		emitter.prelude.add("\t" + tmp + " = " + elseText);
		emitter.prelude.add("}");
		return tmp;
	}

	// ---------------------------------------------------------------- arrays

	private String emitArrayCreation(ArrayCreation ac) {
		String goType = dev.gowt.j2go.GoTypes.map(ac.resolveTypeBinding(), emitter);
		if (ac.getInitializer() != null) {
			return emitArrayInitializer(ac.getInitializer(), ac.resolveTypeBinding());
		}
		// new T[n]: a sized, zero-valued array - {} is a Go LENGTH-0 slice, not one of length n.
		java.util.List<?> dims = ac.dimensions();
		if (!dims.isEmpty() && dims.get(0) != null) {
			return "make(" + goType + ", " + emitExpr((Expression) dims.get(0)) + ")";
		}
		return goType + "{}";
	}

	private String emitArrayInitializer(ArrayInitializer ai, ITypeBinding arrayType) {
		String goType = dev.gowt.j2go.GoTypes.map(arrayType, emitter);
		ITypeBinding componentType = arrayType != null && arrayType.isArray() ? arrayType.getComponentType() : null;
		java.util.List<String> elems = new java.util.ArrayList<>();
		for (Object o : ai.expressions()) {
			Expression e = (Expression) o;
			String text = emitExpr(e);
			if (componentType != null) text = adaptNumeric(text, e.resolveTypeBinding(), componentType);
			elems.add(text);
		}
		return goType + "{" + String.join(", ", elems) + "}";
	}

	// ---------------------------------------------------------------- numeric adaptation

	String adaptNumeric(String text, ITypeBinding from, ITypeBinding to) {
		// A Java `null` passed where a String is expected: Go's string has no nil value, so the
		// String mapping uses "" as null's stand-in (see also emitInfix's ==/!= null case).
		if (text.equals("nil") && to != null && dev.gowt.j2go.GoTypes.map(to, emitter).equals("string")) return "\"\"";
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
		if (toQualified.equals("org.eclipse.swt.internal.cocoa.id")) return text + ".AsId()";
		String toBare = bareGoName(toQualified);
		if (toBare == null) return text;
		return "&" + text + "." + toBare;
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
		return dev.gowt.j2go.Manual.isManual(qualified) ? dev.gowt.j2go.Manual.goTypeName(qualified) : null;
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
		return dev.gowt.j2go.Manual.manualSuperclassOf(qualified);
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
		return "nil";
	}
}
