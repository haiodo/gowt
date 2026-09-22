package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.util.*;

/** Recursive JDT-AST -> Go text emitter for the translated-set classes. */
class Emitter {

	final TypeModel model;
	final Names names;
	final List<String> unsupported = new ArrayList<>();
	final Set<String> generatedHelpers = new LinkedHashSet<>();

	Set<String> fileImports;
	List<String> fileHelperSource;
	List<String> prelude;
	int tempCounter;
	ITypeBinding currentReturnType; // declared Go return type of the method body being emitted, or null

	Emitter(TypeModel model, Names names) {
		this.model = model;
		this.names = names;
	}

	record EmitResult(String body, Set<String> imports) {}

	EmitResult emitCompilationUnit(CompilationUnit cu) {
		fileImports = new LinkedHashSet<>();
		fileHelperSource = new ArrayList<>();
		StringBuilder out = new StringBuilder();
		for (Object t : cu.types()) {
			emitTopLevelClass((TypeDeclaration) t, out);
		}
		for (String h : fileHelperSource) out.append(h);
		return new EmitResult(out.toString(), fileImports);
	}

	// ---------------------------------------------------------------- classes

	private void emitTopLevelClass(TypeDeclaration td, StringBuilder out) {
		emitClass(td, out);
	}

	private void emitClass(TypeDeclaration td, StringBuilder out) {
		TypeModel.ClassInfo ci = model.lookup(td.resolveBinding());
		boolean needsImpl = !ci.root.children.isEmpty();

		if (ci == ci.root && needsImpl) {
			out.append("type ").append(ci.goTypeName).append("Impl interface {\n");
			for (var e : ci.root.overriddenRootMethods.entrySet()) {
				IMethodBinding rootDecl = e.getValue();
				out.append('\t').append(names.goMemberName(rootDecl, Names.capitalize(e.getKey())))
						.append("(").append(paramList(rootDecl, null)).append(") ")
						.append(GoTypes.map(rootDecl.getReturnType(), model)).append('\n');
			}
			out.append("}\n\n");
		}

		out.append("type ").append(ci.goTypeName).append(" struct {\n");
		if (ci.superclass != null) {
			out.append('\t').append(ci.superclass.goTypeName).append('\n');
		}
		Set<String> methodGoNames = collectMethodGoNames(td);
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && !Modifier.isStatic(fd.getModifiers())) {
				emitStructFields(fd, methodGoNames, out);
			}
		}
		if (ci == ci.root && needsImpl) {
			out.append("\timpl ").append(ci.goTypeName).append("Impl\n");
		}
		out.append("}\n\n");

		for (Object o : td.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && Modifier.isStatic(fd.getModifiers())) {
				emitStaticFields(fd, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && md.isConstructor()) {
				emitConstructor(md, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && !Modifier.isStatic(md.getModifiers())) {
				emitInstanceMethod(md, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && Modifier.isStatic(md.getModifiers())) {
				emitStaticMethod(md, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof TypeDeclaration nested) {
				emitClass(nested, out);
			}
		}
	}

	private Set<String> collectMethodGoNames(TypeDeclaration td) {
		Set<String> s = new HashSet<>();
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && !Modifier.isStatic(md.getModifiers())) {
				s.add(names.goMemberName(md.resolveBinding(), Names.capitalize(md.getName().getIdentifier())));
			}
		}
		return s;
	}

	private void emitStructFields(FieldDeclaration fd, Set<String> methodGoNames, StringBuilder out) {
		boolean pub = Modifier.isPublic(fd.getModifiers());
		List<String> goNames = new ArrayList<>();
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String javaName = f.getName().getIdentifier();
			if (javaName.equals("serialVersionUID")) continue;
			String goName = pub ? Names.capitalize(javaName) : javaName;
			if (pub && methodGoNames.contains(goName)) {
				goName = goName + "_";
				unsupported.add("FieldMethodNameClash: " + javaName + " clashes with a method Go name, suffixed _");
			}
			goNames.add(goName);
		}
		if (goNames.isEmpty()) return;
		out.append('\t').append(String.join(", ", goNames)).append(' ')
				.append(GoTypes.map(fd.getType().resolveBinding(), model)).append('\n');
	}

	private void emitStaticFields(FieldDeclaration fd, TypeModel.ClassInfo ci, StringBuilder out) {
		boolean isFinal = Modifier.isFinal(fd.getModifiers());
		ITypeBinding type = fd.getType().resolveBinding();
		boolean isConst = isFinal && (type.isPrimitive() || type.getQualifiedName().equals("java.lang.String"));
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String javaName = f.getName().getIdentifier();
			if (javaName.equals("serialVersionUID")) continue;
			String goName = ci.goFuncPrefix + Names.capitalize(javaName);
			String init = f.getInitializer() == null ? zeroValue(type) : emitExprHoisted(f.getInitializer());
			out.append(isConst ? "const " : "var ").append(goName).append(" ")
					.append(GoTypes.map(type, model)).append(" = ").append(init).append('\n');
		}
		out.append('\n');
	}

	// ---------------------------------------------------------------- constructors

	private void emitConstructor(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		boolean pub = Modifier.isPublic(md.getModifiers());
		String prefix = (pub ? "New" : "new") + ci.goFuncPrefix;
		String goName = names.goMemberName(mb, prefix);
		String initPrefix = (pub ? "init" : "init") + ci.goFuncPrefix; // init methods are always unexported by shape (lowercase receiver call)
		String initName = names.goMemberName(mb, initPrefix);

		boolean needsImpl = !ci.root.children.isEmpty();
		out.append("func ").append(goName).append('(').append(paramList(mb, md)).append(") *")
				.append(ci.goTypeName).append(" {\n");
		out.append("\tthis := &").append(ci.goTypeName).append("{}\n");
		if (needsImpl) out.append("\tthis.impl = this\n");
		out.append("\tthis.").append(initName).append('(').append(argNames(md)).append(")\n");
		out.append("\treturn this\n}\n\n");

		out.append("func (this *").append(ci.goTypeName).append(") ").append(initName)
				.append('(').append(paramList(mb, md)).append(") {\n");
		currentReturnType = null;
		out.append(emitConstructorBody(md, ci));
		out.append("}\n\n");
	}

	private String emitConstructorBody(MethodDeclaration md, TypeModel.ClassInfo ci) {
		List<Statement> stmts = md.getBody().statements();
		StringBuilder b = new StringBuilder();
		int start = 0;
		if (!stmts.isEmpty() && stmts.get(0) instanceof SuperConstructorInvocation sci) {
			b.append(emitSuperInvocation(sci, ci));
			start = 1;
		} else if (!stmts.isEmpty() && stmts.get(0) instanceof ConstructorInvocation cti) {
			b.append(emitThisInvocation(cti, ci));
			start = 1;
		} else if (ci.superclass != null) {
			IMethodBinding zeroArg = findZeroArgCtor(ci.superclass.binding);
			if (zeroArg != null) {
				String initName = names.goMemberName(zeroArg, "init" + ci.superclass.goFuncPrefix);
				b.append("\tthis.").append(ci.superclass.goTypeName).append('.').append(initName).append("()\n");
			}
		}
		for (int i = start; i < stmts.size(); i++) {
			b.append(emitStatement(stmts.get(i), 1));
		}
		return b.toString();
	}

	private IMethodBinding findZeroArgCtor(ITypeBinding t) {
		for (IMethodBinding m : t.getDeclaredMethods()) {
			if (m.isConstructor() && m.getParameterTypes().length == 0) return m;
		}
		return null;
	}

	private String emitSuperInvocation(SuperConstructorInvocation sci, TypeModel.ClassInfo ci) {
		IMethodBinding mb = sci.resolveConstructorBinding();
		String initName = names.goMemberName(mb, "init" + ci.superclass.goFuncPrefix);
		List<String> args = new ArrayList<>();
		for (Object a : sci.arguments()) {
			args.add(adaptArg((Expression) a, mb, sci.arguments().indexOf(a)));
		}
		return "\tthis." + ci.superclass.goTypeName + "." + initName + "(" + String.join(", ", args) + ")\n";
	}

	private String emitThisInvocation(ConstructorInvocation cti, TypeModel.ClassInfo ci) {
		IMethodBinding mb = cti.resolveConstructorBinding();
		String initName = names.goMemberName(mb, "init" + ci.goFuncPrefix);
		List<String> args = new ArrayList<>();
		for (Object a : cti.arguments()) {
			args.add(adaptArg((Expression) a, mb, cti.arguments().indexOf(a)));
		}
		return "\tthis." + initName + "(" + String.join(", ", args) + ")\n";
	}

	// ---------------------------------------------------------------- methods

	private void emitInstanceMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String base = switch (javaName) {
			case "equals" -> "Equals";
			case "hashCode" -> "HashCode";
			case "toString" -> "String";
			default -> Names.capitalize(javaName);
		};
		String goName = names.goMemberName(mb, base);
		boolean overridden = ci.isOverridden(javaName);
		IMethodBinding sigSource = overridden ? ci.root.overriddenRootMethods.get(javaName) : mb;

		out.append("func (this *").append(ci.goTypeName).append(") ").append(goName)
				.append('(').append(paramList(sigSource, md)).append(") ")
				.append(retType(sigSource)).append(" {\n");
		currentReturnType = sigSource.getReturnType();
		out.append(emitBlockBody(md.getBody(), 1));
		currentReturnType = null;
		out.append("}\n\n");
	}

	private void emitStaticMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String goName = ci.goFuncPrefix + names.goMemberName(mb, Names.capitalize(md.getName().getIdentifier()));
		out.append("func ").append(goName).append('(').append(paramList(mb, md)).append(") ")
				.append(retType(mb)).append(" {\n");
		currentReturnType = mb.getReturnType();
		out.append(emitBlockBody(md.getBody(), 1));
		currentReturnType = null;
		out.append("}\n\n");
	}

	private String retType(IMethodBinding mb) {
		String t = GoTypes.map(mb.getReturnType(), model);
		return t.isEmpty() ? "" : t;
	}

	private String paramList(IMethodBinding mb, MethodDeclaration mdOrNull) {
		ITypeBinding[] types = mb.getParameterTypes();
		List<String> names_ = new ArrayList<>();
		if (mdOrNull != null) {
			for (Object p : mdOrNull.parameters()) {
				names_.add(((SingleVariableDeclaration) p).getName().getIdentifier());
			}
		}
		List<String> parts = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			String n = i < names_.size() ? names_.get(i) : "a" + i;
			parts.add(n + " " + GoTypes.map(types[i], model));
		}
		return String.join(", ", parts);
	}

	private String argNames(MethodDeclaration md) {
		List<String> n = new ArrayList<>();
		for (Object p : md.parameters()) n.add(((SingleVariableDeclaration) p).getName().getIdentifier());
		return String.join(", ", n);
	}

	// ---------------------------------------------------------------- statements

	private String emitBlockBody(Block block, int indent) {
		StringBuilder b = new StringBuilder();
		for (Object s : block.statements()) {
			b.append(emitStatement((Statement) s, indent));
		}
		return b.toString();
	}

	private String ind(int n) {
		return "\t".repeat(n);
	}

	private String emitStatement(Statement s, int indent) {
		List<String> saved = prelude;
		prelude = new ArrayList<>();
		String main = emitStatementInner(s, indent);
		StringBuilder b = new StringBuilder();
		for (String p : prelude) b.append(ind(indent)).append(p).append('\n');
		b.append(main);
		prelude = saved;
		return b.toString();
	}

	private String emitStatementInner(Statement s, int indent) {
		if (s instanceof ExpressionStatement es) return emitExpressionStatement(es, indent);
		if (s instanceof ReturnStatement rs) return emitReturn(rs, indent);
		if (s instanceof IfStatement is) return emitIf(is, indent);
		if (s instanceof VariableDeclarationStatement vds) return emitVarDecl(vds, indent);
		if (s instanceof Block b) return ind(indent) + "{\n" + emitBlockBody(b, indent + 1) + ind(indent) + "}\n";
		if (s instanceof SwitchStatement sw) return emitSwitchStatement(sw, indent, null);
		return unsupportedStmt(s, indent);
	}

	private String emitExpressionStatement(ExpressionStatement es, int indent) {
		Expression e = es.getExpression();
		if (e instanceof Assignment a) {
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof Assignment) {
				return emitChainedAssignment(a, indent);
			}
			String lhs = emitExpr(a.getLeftHandSide());
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof ConditionalExpression ce) {
				return emitCondIntoLvalue(lhs, ce, a.getLeftHandSide().resolveTypeBinding(), indent);
			}
			if (a.getOperator() == Assignment.Operator.ASSIGN && a.getRightHandSide() instanceof SwitchExpression se) {
				return emitSwitchExpressionAssign(lhs, se, indent);
			}
			String rhs = adaptNumeric(emitExpr(a.getRightHandSide()), a.getRightHandSide().resolveTypeBinding(), a.getLeftHandSide().resolveTypeBinding());
			return ind(indent) + lhs + " " + a.getOperator().toString() + " " + rhs + "\n";
		}
		return ind(indent) + emitExpr(e) + "\n";
	}

	private String emitChainedAssignment(Assignment a, int indent) {
		List<Expression> targets = new ArrayList<>();
		Expression cur = a;
		while (cur instanceof Assignment ca && ca.getOperator() == Assignment.Operator.ASSIGN) {
			targets.add(ca.getLeftHandSide());
			cur = ca.getRightHandSide();
		}
		String rhs = emitExpr(cur);
		StringBuilder b = new StringBuilder();
		String prev = rhs;
		for (int i = targets.size() - 1; i >= 0; i--) {
			String lhsText = emitExpr(targets.get(i));
			b.append(ind(indent)).append(lhsText).append(" = ").append(prev).append('\n');
			prev = lhsText;
		}
		return b.toString();
	}

	/** ExpressionStatement/assignment whose RHS is a plain (no-instanceof) ternary, or var-decl init. */
	private String emitCondIntoLvalue(String lhsText, ConditionalExpression ce, ITypeBinding targetType, int indent) {
		StringBuilder b = new StringBuilder();
		String cond = emitExprInto(ce.getExpression(), b, indent);
		String thenText = adaptNumeric(emitExpr(ce.getThenExpression()), ce.getThenExpression().resolveTypeBinding(), targetType);
		String elseText = adaptNumeric(emitExpr(ce.getElseExpression()), ce.getElseExpression().resolveTypeBinding(), targetType);
		b.append(ind(indent)).append("if ").append(cond).append(" {\n");
		b.append(ind(indent + 1)).append(lhsText).append(" = ").append(thenText).append('\n');
		b.append(ind(indent)).append("} else {\n");
		b.append(ind(indent + 1)).append(lhsText).append(" = ").append(elseText).append('\n');
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	/** Emits any prelude for e (instanceof hoisting) directly at `indent` into b, returns e's inline text. */
	private String emitExprInto(Expression e, StringBuilder b, int indent) {
		List<String> savedPrelude = prelude;
		prelude = new ArrayList<>();
		String text = emitExpr(e);
		for (String p : prelude) b.append(ind(indent)).append(p).append('\n');
		prelude = savedPrelude;
		return text;
	}

	private String emitVarDecl(VariableDeclarationStatement vds, int indent) {
		StringBuilder b = new StringBuilder();
		ITypeBinding declType = vds.getType().resolveBinding();
		String goType = GoTypes.map(declType, model);
		for (Object o : vds.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String name = f.getName().getIdentifier();
			if (f.getInitializer() == null) {
				b.append(ind(indent)).append("var ").append(name).append(' ').append(goType).append('\n');
				continue;
			}
			if (f.getInitializer() instanceof ConditionalExpression ce) {
				b.append(ind(indent)).append("var ").append(name).append(' ').append(goType).append('\n');
				b.append(emitCondIntoLvalue(name, ce, declType, indent));
				continue;
			}
			String init = adaptNumeric(emitExprInto(f.getInitializer(), b, indent), f.getInitializer().resolveTypeBinding(), declType);
			b.append(ind(indent)).append(name).append(" := ").append(init).append('\n');
		}
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
			b.append(ind(indent)).append("return\n");
			return b.toString();
		}
		Expression e = rs.getExpression();
		String text = emitExprInto(e, b, indent);
		text = upcastForReturn(text, e.resolveTypeBinding(), b, indent);
		b.append(ind(indent)).append("return ").append(text).append('\n');
		return b.toString();
	}

	/**
	 * Go has no covariant returns: a return expression whose type is a strict subtype of the
	 * enclosing method's declared return type needs an explicit upcast to the embedded field.
	 */
	private String upcastForReturn(String text, ITypeBinding exprType, StringBuilder b, int indent) {
		if (currentReturnType == null || exprType == null) return text;
		TypeModel.ClassInfo exprCi = model.lookup(exprType);
		TypeModel.ClassInfo targetCi = model.lookup(currentReturnType);
		if (exprCi == null || targetCi == null || exprCi == targetCi) return text;
		for (TypeModel.ClassInfo a = exprCi.superclass; a != null; a = a.superclass) {
			if (a == targetCi) {
				String tmp = "n" + (++tempCounter);
				b.append(ind(indent)).append(tmp).append(" := ").append(text).append('\n');
				return "&" + tmp + "." + targetCi.goTypeName;
			}
		}
		return text;
	}

	private String unsupportedStmt(Statement s, int indent) {
		String oneLine = s.toString().replace("\n", " ").trim();
		unsupported.add(s.getClass().getSimpleName() + ": " + oneLine);
		return ind(indent) + "panic(\"j2go: unsupported " + s.getClass().getSimpleName() + "\") // TODO(gowt-port): " + oneLine + "\n";
	}

	// ---------------------------------------------------------------- switch expression (RGB HSB only)

	/** Handles `target = switch (e) { case C -> { stmts; yield v; } ... };` lowering. */
	private String emitSwitchExpressionAssign(String targetLhs, SwitchExpression se, int indent) {
		StringBuilder b = new StringBuilder();
		String subject = emitExprInto(se.getExpression(), b, indent);
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
				for (Object ex : sc.expressions()) labels.add(emitExpr((Expression) ex));
				b.append(ind(indent)).append("case ").append(String.join(", ", labels)).append(":\n");
			}
			b.append(emitSwitchCaseBody(body, targetLhs, indent + 1));
		}
		b.append(ind(indent)).append("}\n");
		return b.toString();
	}

	private String emitSwitchCaseBody(Statement body, String targetLhs, int indent) {
		List<Statement> stmts = body instanceof Block bl ? bl.statements() : List.of(body);
		StringBuilder b = new StringBuilder();
		for (Statement s : stmts) {
			if (s instanceof YieldStatement ys) {
				b.append(emitStatement0AsAssign(targetLhs, ys.getExpression(), indent));
			} else {
				b.append(emitStatement(s, indent));
			}
		}
		return b.toString();
	}

	private String emitStatement0AsAssign(String lhs, Expression rhs, int indent) {
		List<String> saved = prelude;
		prelude = new ArrayList<>();
		String text = emitExpr(rhs);
		StringBuilder b = new StringBuilder();
		for (String p : prelude) b.append(ind(indent)).append(p).append('\n');
		b.append(ind(indent)).append(lhs).append(" = ").append(text).append('\n');
		prelude = saved;
		return b.toString();
	}

	private String emitSwitchStatement(SwitchStatement sw, int indent, String unused) {
		return unsupportedStmt(sw, indent);
	}

	// ---------------------------------------------------------------- expressions

	private String emitExpr(Expression e) {
		if (e instanceof NumberLiteral nl) return stripNumericSuffix(nl.getToken());
		if (e instanceof BooleanLiteral bl) return Boolean.toString(bl.booleanValue());
		if (e instanceof StringLiteral sl) return goStringLiteral(sl.getLiteralValue());
		if (e instanceof NullLiteral) return "nil";
		if (e instanceof ThisExpression) return "this";
		if (e instanceof ParenthesizedExpression pe) return "(" + emitExpr(pe.getExpression()) + ")";
		if (e instanceof PrefixExpression pf) return emitPrefix(pf);
		if (e instanceof CastExpression ce) return emitCast(ce);
		if (e instanceof InfixExpression ie) return emitInfix(ie);
		if (e instanceof Assignment a) return emitInlineAssign(a);
		if (e instanceof SimpleName sn) return emitSimpleName(sn);
		if (e instanceof FieldAccess fa) return emitFieldAccess(fa);
		if (e instanceof QualifiedName qn) return emitQualifiedName(qn);
		if (e instanceof MethodInvocation mi) return emitMethodInvocation(mi);
		if (e instanceof ClassInstanceCreation cic) return emitNew(cic);
		if (e instanceof ConditionalExpression ce) return emitConditionalHoisted(ce);
		if (e instanceof InstanceofExpression ioe) return emitPlainInstanceof(ioe);
		if (e instanceof PatternInstanceofExpression pie) return emitPatternInstanceof(pie);
		if (e instanceof ArrayCreation ac) return emitArrayCreation(ac);
		if (e instanceof ArrayInitializer ai) return emitArrayInitializer(ai, ai.resolveTypeBinding());
		if (e instanceof ArrayAccess aa) return emitExpr(aa.getArray()) + "[" + emitExpr(aa.getIndex()) + "]";
		if (e instanceof SwitchExpression) {
			unsupported.add("SwitchExpression: used outside assignment lowering: " + e);
			return "func() any { panic(\"j2go: unsupported SwitchExpression\") }()";
		}
		String oneLine = e.toString().replace("\n", " ").trim();
		unsupported.add(e.getClass().getSimpleName() + ": " + oneLine);
		return "func() any { panic(\"j2go: unsupported " + e.getClass().getSimpleName() + "\") }() /* TODO(gowt-port): " + oneLine + " */";
	}

	private String emitExprHoisted(Expression e) {
		return emitExpr(e);
	}

	private String emitInlineAssign(Assignment a) {
		// nested assignment outside the statement-level chain handler: hoist to prelude.
		String tmp = "cond" + (++tempCounter);
		prelude.add(tmp + " := " + emitExpr(a.getRightHandSide()));
		prelude.add(emitExpr(a.getLeftHandSide()) + " = " + tmp);
		return tmp;
	}

	private String emitPrefix(PrefixExpression pf) {
		String op = pf.getOperator().toString();
		if (op.equals("-") || op.equals("!")) return op + emitExpr(pf.getOperand());
		unsupported.add("PrefixExpression: " + pf);
		return "panicExpr(\"" + pf + "\")";
	}

	private String emitCast(CastExpression ce) {
		ITypeBinding t = ce.getType().resolveBinding();
		String goType = GoTypes.map(t, model);
		return goType + "(" + emitExpr(ce.getExpression()) + ")";
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
		String left = emitExpr(ie.getLeftOperand());
		String right = emitExpr(ie.getRightOperand());
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
		String g = GoTypes.map(t, model);
		return NUMERIC_RANK.containsKey(g) ? g : null;
	}

	private String emitStringConcat(InfixExpression ie) {
		fileImports.add("fmt");
		List<Expression> operands = new ArrayList<>();
		operands.add(ie.getLeftOperand());
		operands.add(ie.getRightOperand());
		for (Object o : ie.extendedOperands()) operands.add((Expression) o);
		StringBuilder fmt = new StringBuilder();
		List<String> args = new ArrayList<>();
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
		String g = GoTypes.map(t, model);
		return switch (g) {
			case "int8", "int16", "uint16", "int32", "int64" -> "%d";
			case "float32", "float64" -> "%v";
			case "string" -> "%s";
			case "bool" -> "%t";
			default -> "%v";
		};
	}

	private String escapeForFormat(String s) {
		return s.replace("\\", "\\\\").replace("\"", "\\\"").replace("%", "%%");
	}

	private String goStringLiteral(String s) {
		return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"") + "\"";
	}

	private String stripNumericSuffix(String token) {
		if (token.isEmpty()) return token;
		char last = token.charAt(token.length() - 1);
		if (last == 'f' || last == 'F' || last == 'd' || last == 'D' || last == 'l' || last == 'L') {
			return token.substring(0, token.length() - 1);
		}
		return token;
	}

	private String emitSimpleName(SimpleName sn) {
		IBinding b = sn.resolveBinding();
		if (b instanceof IVariableBinding vb && vb.isField()) {
			return "this." + fieldGoName(vb);
		}
		return sn.getIdentifier();
	}

	private String fieldGoName(IVariableBinding vb) {
		String n = vb.getName();
		if (Modifier.isPublic(vb.getModifiers())) return Names.capitalize(n);
		return n;
	}

	private String emitFieldAccess(FieldAccess fa) {
		IVariableBinding vb = fa.resolveFieldBinding();
		String recv = emitExpr(fa.getExpression());
		return recv + "." + fieldGoName(vb);
	}

	private String emitQualifiedName(QualifiedName qn) {
		IBinding b = qn.resolveBinding();
		if (b instanceof IVariableBinding vb && vb.isField()) {
			if (Modifier.isStatic(vb.getModifiers())) {
				return staticFieldRef(vb);
			}
			String recv = emitExpr(qn.getQualifier());
			return recv + "." + fieldGoName(vb);
		}
		if (b instanceof ITypeBinding) {
			return qn.getName().getIdentifier();
		}
		return qn.getName().getIdentifier();
	}

	private String staticFieldRef(IVariableBinding vb) {
		ITypeBinding declaring = vb.getDeclaringClass();
		String qualified = declaring.getErasure().getQualifiedName();
		if (Manual.isManual(qualified)) return Manual.staticMember(qualified, vb.getName());
		TypeModel.ClassInfo ci = model.lookup(declaring);
		if (ci != null) return ci.goFuncPrefix + Names.capitalize(vb.getName());
		unsupported.add("StaticField: unresolved declaring type for " + vb.getName());
		return vb.getName();
	}

	// ---------------------------------------------------------------- calls / new

	private String emitMethodInvocation(MethodInvocation mi) {
		IMethodBinding mb = mi.resolveMethodBinding();
		ITypeBinding declaring = mb.getDeclaringClass();
		String qualified = declaring.getErasure().getQualifiedName();

		if (qualified.equals("java.lang.Math") && (mb.getName().equals("max") || mb.getName().equals("min"))) {
			return emitMathMinMax(mi, mb);
		}

		List<String> args = buildArgs(mi.arguments(), mb);

		if (Manual.isManual(qualified)) {
			if (Modifier.isStatic(mb.getModifiers())) {
				return Manual.staticMember(qualified, mb.getName()) + "(" + String.join(", ", args) + ")";
			}
			String recv = mi.getExpression() != null ? emitExpr(mi.getExpression()) : "this";
			return recv + "." + Manual.instanceMember(mb.getName()) + "(" + String.join(", ", args) + ")";
		}

		TypeModel.ClassInfo ci = model.lookup(declaring);
		if (ci == null) {
			unsupported.add("MethodInvocation: unresolved declaring type " + qualified + "." + mb.getName());
			return "panicExpr(\"unresolved call " + mb.getName() + "\")";
		}

		if (Modifier.isStatic(mb.getModifiers())) {
			String goName = ci.goFuncPrefix + names.goMemberName(mb, Names.capitalize(mb.getName()));
			return goName + "(" + String.join(", ", args) + ")";
		}

		String recv = mi.getExpression() != null ? emitExpr(mi.getExpression()) : "this";
		String base = switch (mb.getName()) {
			case "equals" -> "Equals";
			case "hashCode" -> "HashCode";
			case "toString" -> "String";
			default -> Names.capitalize(mb.getName());
		};
		String goName = names.goMemberName(mb, base);
		boolean overridden = ci.isOverridden(mb.getName());
		String callText = overridden
				? recv + ".impl." + goName + "(" + String.join(", ", args) + ")"
				: recv + "." + goName + "(" + String.join(", ", args) + ")";

		if (overridden) {
			ITypeBinding staticReturnType = mb.getReturnType(); // as resolved at THIS call site (covariant-aware)
			TypeModel.ClassInfo rootCi = ci.root;
			if (staticReturnType != null && !staticReturnType.getErasure().getBinaryName().equals(rootCi.binaryName)) {
				TypeModel.ClassInfo target = model.lookup(staticReturnType);
				if (target != null) {
					String tmp = "t" + (++tempCounter);
					prelude.add(tmp + " := " + callText);
					String helper = ensureCascadeHelper(rootCi, target);
					String tmp2 = "t" + (++tempCounter);
					prelude.add(tmp2 + ", _ := " + helper + "(" + tmp + ".impl)");
					return tmp2;
				}
			}
		}
		return callText;
	}

	/**
	 * java.lang.Math has no translated-set/manual rule of its own; Math.max/min(float,float) is
	 * the only overload the 4 target files use, so it is mapped inline to Go's math package.
	 */
	private String emitMathMinMax(MethodInvocation mi, IMethodBinding mb) {
		fileImports.add("math");
		String goFn = mb.getName().equals("max") ? "math.Max" : "math.Min";
		String a = emitExpr((Expression) mi.arguments().get(0));
		String b = emitExpr((Expression) mi.arguments().get(1));
		String goReturn = GoTypes.map(mb.getReturnType(), model);
		return goReturn + "(" + goFn + "(float64(" + a + "), float64(" + b + ")))";
	}

	private List<String> buildArgs(List<?> javaArgs, IMethodBinding mb) {
		List<String> args = new ArrayList<>();
		ITypeBinding[] paramTypes = mb.getParameterTypes();
		for (int i = 0; i < javaArgs.size(); i++) {
			Expression a = (Expression) javaArgs.get(i);
			String text = emitExpr(a);
			if (i < paramTypes.length) {
				text = adaptNumeric(text, a.resolveTypeBinding(), paramTypes[i]);
			}
			args.add(text);
		}
		return args;
	}

	private String adaptArg(Expression a, IMethodBinding mb, int i) {
		ITypeBinding[] paramTypes = mb.getParameterTypes();
		String text = emitExpr(a);
		if (i < paramTypes.length) text = adaptNumeric(text, a.resolveTypeBinding(), paramTypes[i]);
		return text;
	}

	private String emitNew(ClassInstanceCreation cic) {
		IMethodBinding ctor = cic.resolveConstructorBinding();
		ITypeBinding declaring = ctor.getDeclaringClass();
		TypeModel.ClassInfo ci = model.lookup(declaring);
		if (ci == null) {
			unsupported.add("ClassInstanceCreation: unresolved type " + declaring.getQualifiedName());
			return "panicExpr(\"unresolved new " + declaring.getName() + "\")";
		}
		boolean pub = Modifier.isPublic(ctor.getModifiers());
		String prefix = (pub ? "New" : "new") + ci.goFuncPrefix;
		String goName = names.goMemberName(ctor, prefix);
		List<String> args = buildArgs(cic.arguments(), ctor);
		return goName + "(" + String.join(", ", args) + ")";
	}

	// ---------------------------------------------------------------- conditional / instanceof

	private String emitConditionalHoisted(ConditionalExpression ce) {
		String tmp = "cond" + (++tempCounter);
		ITypeBinding target = ce.resolveTypeBinding();
		String goType = GoTypes.map(target, model);
		prelude.add("var " + tmp + " " + goType);
		String cond = emitExpr(ce.getExpression());
		String thenText = adaptNumeric(emitExpr(ce.getThenExpression()), ce.getThenExpression().resolveTypeBinding(), target);
		String elseText = adaptNumeric(emitExpr(ce.getElseExpression()), ce.getElseExpression().resolveTypeBinding(), target);
		prelude.add("if " + cond + " {");
		prelude.add("\t" + tmp + " = " + thenText);
		prelude.add("} else {");
		prelude.add("\t" + tmp + " = " + elseText);
		prelude.add("}");
		return tmp;
	}

	private String emitPlainInstanceof(InstanceofExpression ioe) {
		ITypeBinding target = ioe.getRightOperand().resolveBinding();
		String okVar = "ok" + (++tempCounter);
		String helperOrAssert = instanceofCheck(ioe.getLeftOperand(), target, "_", okVar);
		prelude.add(helperOrAssert);
		return okVar;
	}

	private String emitPatternInstanceof(PatternInstanceofExpression pie) {
		TypePattern tp = (TypePattern) pie.getPattern();
		String varName = tp.getPatternVariable().getName().getIdentifier();
		ITypeBinding target = tp.getPatternVariable().getType().resolveBinding();
		String okVar = "ok" + (++tempCounter);
		prelude.add(instanceofCheck(pie.getLeftOperand(), target, varName, okVar));
		return okVar;
	}

	private String instanceofCheck(Expression subject, ITypeBinding target, String varName, String okVar) {
		TypeModel.ClassInfo targetCi = model.lookup(target);
		String subjectText = emitExpr(subject);
		if (targetCi == null) {
			unsupported.add("instanceof: unresolved target type " + target.getQualifiedName());
			return varName + ", " + okVar + " := any(nil), false";
		}
		// A translated-class-typed subject carries its dynamic subtype in .impl;
		// an Object-typed subject (e.g. equals(Object)) already holds the concrete pointer.
		ITypeBinding subjectType = subject.resolveTypeBinding();
		TypeModel.ClassInfo subjectCi = subjectType == null ? null : model.lookup(subjectType);
		if (subjectCi != null) subjectText = subjectText + ".impl";
		String helper = ensureCascadeHelper(targetCi.root, targetCi);
		return varName + ", " + okVar + " := " + helper + "(" + subjectText + ")";
	}

	private String ensureCascadeHelper(TypeModel.ClassInfo root, TypeModel.ClassInfo target) {
		String targetLabel = target == root ? target.goFuncPrefix : target.goFuncPrefix.substring(root.goFuncPrefix.length());
		String name = Names.decapitalize(root.goFuncPrefix) + "ImplAs" + targetLabel;
		if (generatedHelpers.add(name)) {
			fileHelperSource.add(buildCascadeHelper(name, target));
		}
		return name;
	}

	private String buildCascadeHelper(String name, TypeModel.ClassInfo target) {
		List<TypeModel.ClassInfo> concrete = new ArrayList<>();
		collectDescendants(target, concrete);
		StringBuilder b = new StringBuilder();
		b.append("// j2go: instanceof helper for ").append(target.goTypeName)
				.append(" and its subclasses within the translated set.\n");
		b.append("func ").append(name).append("(x any) (*").append(target.goTypeName).append(", bool) {\n");
		b.append("\tswitch v := x.(type) {\n");
		for (TypeModel.ClassInfo c : concrete) {
			b.append("\tcase *").append(c.goTypeName).append(":\n");
			if (c == target) {
				b.append("\t\treturn v, true\n");
			} else {
				b.append("\t\treturn &v.").append(target.goTypeName).append(", true\n");
			}
		}
		b.append("\t}\n\treturn nil, false\n}\n\n");
		return b.toString();
	}

	private void collectDescendants(TypeModel.ClassInfo c, List<TypeModel.ClassInfo> out) {
		out.add(c);
		for (TypeModel.ClassInfo child : c.children) collectDescendants(child, out);
	}

	// ---------------------------------------------------------------- arrays

	private String emitArrayCreation(ArrayCreation ac) {
		String goType = GoTypes.map(ac.resolveTypeBinding(), model);
		if (ac.getInitializer() != null) {
			return emitArrayInitializer(ac.getInitializer(), ac.resolveTypeBinding());
		}
		return goType + "{}";
	}

	private String emitArrayInitializer(ArrayInitializer ai, ITypeBinding arrayType) {
		String goType = GoTypes.map(arrayType, model);
		ITypeBinding componentType = arrayType != null && arrayType.isArray() ? arrayType.getComponentType() : null;
		List<String> elems = new ArrayList<>();
		for (Object o : ai.expressions()) {
			Expression e = (Expression) o;
			String text = emitExpr(e);
			if (componentType != null) text = adaptNumeric(text, e.resolveTypeBinding(), componentType);
			elems.add(text);
		}
		return goType + "{" + String.join(", ", elems) + "}";
	}

	// ---------------------------------------------------------------- numeric adaptation

	private String adaptNumeric(String text, ITypeBinding from, ITypeBinding to) {
		if (from == null || to == null || !from.isPrimitive() || !to.isPrimitive()) return text;
		String fromGo = GoTypes.map(from, model);
		String toGo = GoTypes.map(to, model);
		if (fromGo.equals(toGo) || fromGo.equals("bool") || toGo.equals("bool")) return text;
		return toGo + "(" + text + ")";
	}

	private String zeroValue(ITypeBinding t) {
		if (t.isPrimitive()) {
			return switch (t.getName()) {
				case "boolean" -> "false";
				case "float", "double" -> "0";
				default -> "0";
			};
		}
		return "nil";
	}
}
