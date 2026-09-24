package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import static dev.gowt.j2go.emit.EmitUtil.*;

/** Expression dispatch, prefix/postfix, assignment, literals, names and field access, static
 * field refs, conditional hoisting, array creation. Infix/numeric adaptation live in NumericEmitter. */
final class ExpressionEmitter {

	private final Emitter emitter;

	ExpressionEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- expressions

	String emitExpr(Expression e) {
		if (e instanceof NumberLiteral nl) return emitNumber(nl);
		if (e instanceof CharacterLiteral cl) return runeLiteral(cl.charValue());
		if (e instanceof BooleanLiteral bl) return Boolean.toString(bl.booleanValue());
		if (e instanceof StringLiteral sl) return goStringLiteral(sl.getLiteralValue());
		if (e instanceof NullLiteral) return "nil";
		if (e instanceof ThisExpression te) return te.getQualifier() == null && emitter.anonThis != null ? emitter.anonThis : "this";
		if (e instanceof ParenthesizedExpression pe) return "(" + emitExpr(pe.getExpression()) + ")";
		if (e instanceof PrefixExpression pf) return emitPrefix(pf);
		if (e instanceof PostfixExpression pf) return emitPostfix(pf);
		if (e instanceof CastExpression ce) return emitter.emitCast(ce);
		if (e instanceof InfixExpression ie) return emitter.emitInfix(ie);
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
		if (e instanceof ExpressionMethodReference emr) return emitter.emitMethodReference(emr);
		if (e instanceof LambdaExpression le) return emitter.emitLambda(le);
		if (e instanceof SwitchExpression se) return emitSwitchExpressionHoisted(se);
		// X.class: java.lang.Class is reflect.Type (Manual).
		if (e instanceof TypeLiteral tl) {
			emitter.fileImports.add("reflect");
			return "reflect.TypeFor[" + dev.gowt.j2go.GoTypes.map(tl.getType().resolveBinding(), emitter) + "]()";
		}
		String oneLine = e.toString().replace("\n", " ").trim();
		emitter.unsupported.add(e.getClass().getSimpleName() + ": " + oneLine);
		return panicClosure(e, "unsupported " + e.getClass().getSimpleName()) + " /* TODO(gowt-port): " + oneLine + " */";
	}

	// Java escapes like NUL or quote are not Go rune syntax - printable ASCII verbatim, the rest as hex.
	private static String runeLiteral(char c) {
		if (c >= 0x20 && c < 0x7f && c != '\'' && c != '\\') return "'" + c + "'";
		return String.format("'\\u%04x'", (int) c);
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
		String rhs = emitter.adaptNumeric(emitExpr(a.getRightHandSide()), a.getRightHandSide().resolveTypeBinding(),
				a.getLeftHandSide().resolveTypeBinding());
		if (a.getOperator() != Assignment.Operator.ASSIGN) {
			// Compound (`sp = spr += d`): apply in place, the expression's value is the new lhs.
			String lhs = emitExpr(a.getLeftHandSide());
			if (a.getOperator() == Assignment.Operator.RIGHT_SHIFT_UNSIGNED_ASSIGN) {
				emitter.prelude.add(lhs + " = " + emitter.unsignedShift(lhs, a.getLeftHandSide().resolveTypeBinding(), rhs));
			} else {
				emitter.prelude.add(lhs + " " + a.getOperator() + " " + rhs);
			}
			return lhs;
		}
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


	// A hex/octal int literal past 0x7fffffff (0xFF000000) is a negative Java int: Go needs the value.
	private String emitNumber(NumberLiteral nl) {
		String t = stripNumericSuffix(nl.getToken());
		ITypeBinding type = nl.resolveTypeBinding();
		if (type == null || !(t.startsWith("0x") || t.startsWith("0X"))) return t;
		long v = Long.parseUnsignedLong(t.substring(2).replace("_", ""), 16);
		if (type.getName().equals("int") && v > Integer.MAX_VALUE) return Integer.toString((int) v);
		if (type.getName().equals("long") && v < 0) return Long.toString(v);
		return t;
	}

	private String emitSimpleName(SimpleName sn) {
		IBinding b = sn.resolveBinding();
		if (b instanceof IVariableBinding vb && vb.isField()) {
			if (Modifier.isStatic(vb.getModifiers())) return staticFieldRef(vb);
			return emitter.implicitThis(vb.getDeclaringClass()) + "." + fieldGoName(vb);
		}
		return emitter.sanitizeIdent(sn.getIdentifier());
	}

	String fieldGoName(IVariableBinding vb) {
		String n = vb.getName();
		// A field declared on a manual-super type (jrt.EventObject's Source, ...) keeps that
		// type's own Go name, which is always exported - jrt is a separate Go package, so an
		// unexported field there wouldn't even be visible from swt's promoted-field access.
		String declaringQualified = vb.getDeclaringClass().getErasure().getQualifiedName();
		if (Manual.isManualSuper(declaringQualified)) return Names.capitalize(n);
		if (Modifier.isPublic(vb.getModifiers())) return Names.capitalize(n);
		return EmitUtil.fieldIdent(n);
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
		// Selector enum is elided (see README): its constant sel_x is OS's own sel_x field.
		TypeModel.ClassInfo os = qualified.equals("org.eclipse.swt.internal.cocoa.Selector")
				? emitter.model.lookupBinaryName("org.eclipse.swt.internal.cocoa.OS") : null;
		if (os != null) return emitter.qualifiedFuncPrefix(os) + Names.capitalize(vb.getName());
		String jdk = JDK_CONSTANTS.get(qualified + "." + vb.getName());
		if (jdk != null) {
			if (jdk.contains(".")) emitter.fileImports.add(jdk.substring(0, jdk.indexOf('.')));
			return jdk;
		}
		if (Manual.isManual(qualified)) return Manual.staticMember(qualified, vb.getName());
		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (ci != null) {
			String goName = staticFieldGoName(emitter, ci, vb.getName());
			return staticFieldClashesWithMethod(declaring, vb.getName()) ? goName + "_" : goName;
		}
		emitter.unsupported.add("StaticField: unresolved declaring type for " + vb.getName());
		return vb.getName();
	}

	private static final java.util.Map<String, String> JDK_CONSTANTS = java.util.Map.of(
			"java.lang.Integer.MAX_VALUE", "math.MaxInt32", "java.lang.Integer.MIN_VALUE", "math.MinInt32",
			"java.lang.Long.MAX_VALUE", "math.MaxInt64", "java.lang.Long.MIN_VALUE", "math.MinInt64",
			"java.lang.Double.MAX_VALUE", "math.MaxFloat64", "java.lang.Double.MIN_VALUE", "math.SmallestNonzeroFloat64",
			"java.lang.Float.MAX_VALUE", "math.MaxFloat32", "java.lang.Float.MIN_VALUE", "math.SmallestNonzeroFloat32",
			"java.lang.Thread.MAX_PRIORITY", "10", "java.lang.System.out", "os.Stdout");

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
		String thenText = emitter.adaptNumeric(emitExpr(ce.getThenExpression()), ce.getThenExpression().resolveTypeBinding(), target);
		String elseText = emitter.adaptNumeric(emitExpr(ce.getElseExpression()), ce.getElseExpression().resolveTypeBinding(), target);
		emitter.prelude.add("if " + cond + " {");
		emitter.prelude.add("\t" + tmp + " = " + thenText);
		emitter.prelude.add("} else {");
		emitter.prelude.add("\t" + tmp + " = " + elseText);
		emitter.prelude.add("}");
		return tmp;
	}

	/** A switch expression outside a plain assignment (`return switch ...`, an argument): lowered
	 * into the prelude as an assignment to a temp, like a ternary. */
	private String emitSwitchExpressionHoisted(SwitchExpression se) {
		String tmp = "sw" + (++emitter.tempCounter);
		emitter.prelude.add("var " + tmp + " " + dev.gowt.j2go.GoTypes.map(se.resolveTypeBinding(), emitter));
		for (String line : emitter.emitSwitchExpressionAssign(tmp, se, 0).split("\n")) emitter.prelude.add(line);
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
			if (componentType != null) text = emitter.adaptNumeric(text, e.resolveTypeBinding(), componentType);
			elems.add(text);
		}
		return goType + "{" + String.join(", ", elems) + "}";
	}
}
