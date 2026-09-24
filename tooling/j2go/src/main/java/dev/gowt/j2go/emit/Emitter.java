package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.Natives;
import dev.gowt.j2go.Selectors;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

import static dev.gowt.j2go.emit.EmitUtil.ind;

/**
 * Recursive JDT-AST -> Go text emitter for the translated-set classes: facade + shared
 * per-compilation-unit state for the emit.* components, plus the thin cross-component
 * delegators (expr/stmt/block/...) that let those components avoid referencing each other
 * directly. See tooling/j2go/README.md "Source layout" for what each component owns.
 */
public class Emitter {

	public final TypeModel model;
	final Names names;
	final Natives natives;
	final Selectors selectors;
	public final List<String> unsupported = new ArrayList<>();
	final Set<String> generatedHelpers = new LinkedHashSet<>();

	Set<String> fileImports;
	List<String> fileHelperSource;
	List<String> prelude;
	// func init() body: static-field assignments whose initializer needs prelude or calls
	// something (a native's own lazy binding happens on its first call - see README).
	List<String> deferredStaticInits;
	// Parallel to deferredStaticInits: a human-readable label (the field's Go name, or the
	// enclosing class for a static {} block) for the per-entry recover() diagnostic.
	List<String> deferredStaticInitLabels;
	int tempCounter;
	ITypeBinding currentReturnType; // declared Go return type of the method body being emitted, or null
	String currentJavaPackage; // this compilation unit's Java package, e.g. "org.eclipse.swt.widgets"
	String currentGoPackage; // GoTypes.goPackageOf(currentJavaPackage): "swt" or "cocoa"
	String currentClassGoTypeName; // enclosing class's own Go type name, e.g. "id" (see sanitizeIdent)
	TypeModel.ClassInfo currentClassInfo; // enclosing class, for super.method()'s field-path lookup
	// Non-null while emitting an anonymous class's method body (FunctionalEmitter): the Go
	// variable holding the anonymous instance, and its Java type, for `this`/inherited members.
	String anonThis;
	ITypeBinding anonType;
	int anonCounter; // per compilation unit, names <Class>Anon<N> struct types

	// try/catch escape scheme (README "try/catch/finally"): non-null while emitting a try/catch
	// body whose return/unlabeled break/continue must escape the recover() closure boundary.
	String currentEscapeReturnedFlag;
	String currentEscapeBrokeFlag;
	String currentEscapeContinuedFlag;
	String currentEscapeRetVar;
	// Loop/switch nesting since the current closure boundary - an unlabeled break/continue needs
	// the escape scheme above only when this is 0 (not already targeting a loop inside it).
	int loopSwitchDepth;

	private final ClassEmitter classEmitter;
	private final ConstructorEmitter constructorEmitter;
	private final NativeEmitter nativeEmitter;
	private final StatementEmitter statementEmitter;
	private final ControlFlowEmitter controlFlowEmitter;
	private final ExpressionEmitter expressionEmitter;
	private final InvocationEmitter invocationEmitter;
	private final JdkIntrinsics jdkIntrinsics;
	private final TypeTestEmitter typeTestEmitter;
	private final NumericEmitter numericEmitter;
	private final FunctionalEmitter functionalEmitter;

	public Emitter(TypeModel model, Names names, Natives natives, Selectors selectors) {
		this.model = model;
		this.names = names;
		this.natives = natives;
		this.selectors = selectors;
		this.classEmitter = new ClassEmitter(this);
		this.constructorEmitter = new ConstructorEmitter(this);
		this.nativeEmitter = new NativeEmitter(this);
		this.statementEmitter = new StatementEmitter(this);
		this.controlFlowEmitter = new ControlFlowEmitter(this);
		this.expressionEmitter = new ExpressionEmitter(this);
		this.invocationEmitter = new InvocationEmitter(this);
		this.jdkIntrinsics = new JdkIntrinsics(this);
		this.typeTestEmitter = new TypeTestEmitter(this);
		this.numericEmitter = new NumericEmitter(this);
		this.functionalEmitter = new FunctionalEmitter(this);
	}

	public record EmitResult(String body, Set<String> imports) {}

	public EmitResult emitCompilationUnit(CompilationUnit cu) {
		currentJavaPackage = cu.getPackage().getName().getFullyQualifiedName();
		currentGoPackage = dev.gowt.j2go.GoTypes.goPackageOf(currentJavaPackage);
		fileImports = new LinkedHashSet<>();
		fileHelperSource = new ArrayList<>();
		deferredStaticInits = new ArrayList<>();
		deferredStaticInitLabels = new ArrayList<>();
		anonCounter = 0;
		StringBuilder out = new StringBuilder();
		for (Object t : cu.types()) {
			classEmitter.emitTopLevelClass((TypeDeclaration) t, out);
		}
		for (String h : fileHelperSource) out.append(h);
		if (!deferredStaticInits.isEmpty()) {
			fileImports.add("os");
			fileImports.add("fmt");
			out.append("func init() {\n");
			// Some deferred fields (e.g. kUTType*) resolve only inside SWT's own native lib,
			// which this port lacks - recover per-entry so one bad symbol doesn't sink the rest.
			for (int i = 0; i < deferredStaticInits.size(); i++) {
				String label = deferredStaticInitLabels.get(i);
				out.append("\tfunc() {\n\t\tdefer func() {\n\t\t\tif r := recover(); r != nil {\n")
						.append("\t\t\t\tfmt.Fprintln(os.Stderr, \"gowt/internal/cocoa: deferred init ")
						.append(label).append(":\", r)\n")
						.append("\t\t\t}\n\t\t}()\n").append(deferredStaticInits.get(i)).append("\t}()\n");
			}
			out.append("}\n\n");
		}
		// Type mapping records imports as a side effect even where the mapped text is only
		// compared, never emitted - keep just the imports the body actually references.
		String body = out.toString();
		fileImports.removeIf(imp -> !body.contains(imp.substring(imp.lastIndexOf('/') + 1) + "."));
		return new EmitResult(body, fileImports);
	}

	// ---------------------------------------------------------------- cross-component delegators
	//
	// Components call each other only through these, never by holding a direct reference to one
	// another - keeps each component file free to be read (and changed) on its own.

	String expr(Expression e) {
		return expressionEmitter.emitExpr(e);
	}

	String exprInto(Expression e, StringBuilder b, int indent) {
		return statementEmitter.emitExprInto(e, b, indent);
	}

	String stmt(Statement s, int indent) {
		return statementEmitter.emitStatement(s, indent);
	}

	String block(Block b, int indent) {
		return statementEmitter.emitBlockBody(b, indent);
	}

	String returnOrEscape(int indent, String exprTextOrNull) {
		return statementEmitter.returnOrEscape(indent, exprTextOrNull);
	}

	String breakOrEscape(int indent) {
		return statementEmitter.breakOrEscape(indent);
	}

	String continueOrEscape(int indent) {
		return statementEmitter.continueOrEscape(indent);
	}

	String emitSwitchStatement(SwitchStatement sw, int indent) {
		return controlFlowEmitter.emitSwitchStatement(sw, indent);
	}

	String emitSwitchExpressionAssign(String targetLhs, SwitchExpression se, int indent) {
		return controlFlowEmitter.emitSwitchExpressionAssign(targetLhs, se, indent);
	}

	String emitThrow(ThrowStatement ts, int indent) {
		return controlFlowEmitter.emitThrow(ts, indent);
	}

	String emitTry(TryStatement ts, int indent) {
		return controlFlowEmitter.emitTry(ts, indent);
	}

	String emitSynchronized(SynchronizedStatement ss, int indent) {
		return controlFlowEmitter.emitSynchronized(ss, indent);
	}

	String emitLambda(LambdaExpression le) {
		return functionalEmitter.emitLambda(le);
	}

	String emitMethodReference(ExpressionMethodReference emr) {
		return functionalEmitter.emitMethodReference(emr);
	}

	String emitAnonymous(ClassInstanceCreation cic) {
		return functionalEmitter.emitAnonymous(cic);
	}

	String emitCast(CastExpression ce) {
		return typeTestEmitter.emitCast(ce);
	}

	String emitPlainInstanceof(InstanceofExpression ioe) {
		return typeTestEmitter.emitPlainInstanceof(ioe);
	}

	String emitPatternInstanceof(PatternInstanceofExpression pie) {
		return typeTestEmitter.emitPatternInstanceof(pie);
	}

	String ensureCascadeHelper(TypeModel.ClassInfo root, TypeModel.ClassInfo target) {
		return typeTestEmitter.ensureCascadeHelper(root, target);
	}

	String emitMethodInvocation(MethodInvocation mi) {
		return invocationEmitter.emitMethodInvocation(mi);
	}

	String emitNew(ClassInstanceCreation cic) {
		return invocationEmitter.emitNew(cic);
	}

	List<String> buildArgs(List<?> javaArgs, IMethodBinding mb) {
		return invocationEmitter.buildArgs(javaArgs, mb);
	}

	String adaptArg(Expression a, IMethodBinding mb, int i) {
		return invocationEmitter.adaptArg(a, mb, i);
	}

	String tryIntrinsic(MethodInvocation mi, IMethodBinding mb) {
		return jdkIntrinsics.tryIntrinsic(mi, mb);
	}

	void emitStaticNativeMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		nativeEmitter.emitStaticNativeMethod(md, ci, out);
	}

	void emitUnsupportedNative(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		nativeEmitter.emitUnsupportedNative(md, ci, out);
	}

	TypeModel.ClassInfo structParamTarget(ITypeBinding t) {
		return nativeEmitter.structParamTarget(t);
	}

	String emitSuperMethodInvocation(SuperMethodInvocation smi) {
		return constructorEmitter.emitSuperMethodInvocation(smi);
	}

	void emitConstructor(MethodDeclaration md, TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		constructorEmitter.emitConstructor(md, ci, td, out);
	}

	void emitImplicitConstructor(TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		constructorEmitter.emitImplicitConstructor(ci, td, out);
	}

	String staticMethodGoName(IMethodBinding mb, TypeModel.ClassInfo ci) {
		return classEmitter.staticMethodGoName(mb, ci);
	}

	String fieldGoName(IVariableBinding vb) {
		return expressionEmitter.fieldGoName(vb);
	}

	String selectorStringOf(QualifiedName qn) {
		return expressionEmitter.selectorStringOf(qn);
	}

	boolean containsCall(Expression e) {
		return expressionEmitter.containsCall(e);
	}

	String panicClosure(Expression e, String message) {
		return expressionEmitter.panicClosure(e, message);
	}

	String panicClosureTyped(String goType, String message) {
		return expressionEmitter.panicClosureTyped(goType, message);
	}

	String adaptNumeric(String text, ITypeBinding from, ITypeBinding to) {
		return numericEmitter.adaptNumeric(text, from, to);
	}

	String upcastObject(String text, ITypeBinding from, ITypeBinding to) {
		return numericEmitter.upcastObject(text, from, to);
	}

	String zeroValue(ITypeBinding t) {
		return numericEmitter.zeroValue(t);
	}

	String emitInfix(InfixExpression ie) {
		return numericEmitter.emitInfix(ie);
	}

	String ctorGoName(IMethodBinding ctor, String prefix) {
		return constructorEmitter.ctorGoName(ctor, prefix);
	}

	// ---------------------------------------------------------------- function-body state

	/** Per-function emission state a nested Go func literal (lambda/anonymous method) must not
	 * inherit from the enclosing method: return type, try/catch escape flags, loop depth. */
	record BodyState(ITypeBinding returnType, String returned, String broke, String continued, String retVar, int depth) {}

	BodyState enterFunctionBody(ITypeBinding returnType) {
		BodyState s = new BodyState(currentReturnType, currentEscapeReturnedFlag, currentEscapeBrokeFlag,
				currentEscapeContinuedFlag, currentEscapeRetVar, loopSwitchDepth);
		currentReturnType = returnType;
		currentEscapeReturnedFlag = currentEscapeBrokeFlag = currentEscapeContinuedFlag = currentEscapeRetVar = null;
		loopSwitchDepth = 0;
		return s;
	}

	void exitFunctionBody(BodyState s) {
		currentReturnType = s.returnType();
		currentEscapeReturnedFlag = s.returned();
		currentEscapeBrokeFlag = s.broke();
		currentEscapeContinuedFlag = s.continued();
		currentEscapeRetVar = s.retVar();
		loopSwitchDepth = s.depth();
	}

	/** Receiver for an unqualified member access: the anonymous instance when the member is
	 * inherited by the anonymous class but not by the enclosing class, else "this". */
	String implicitThis(ITypeBinding declaringClass) {
		if (anonThis == null || declaringClass == null) return "this";
		boolean anonHasIt = anonType.getErasure().isSubTypeCompatible(declaringClass.getErasure());
		boolean outerHasIt = currentClassInfo != null && currentClassInfo.binding.isSubTypeCompatible(declaringClass.getErasure());
		return anonHasIt && !outerHasIt ? anonThis : "this";
	}

	// ---------------------------------------------------------------- shared utilities
	//
	// Cross-cutting helpers bound to this facade's own per-compilation-unit state
	// (currentClassGoTypeName, fileImports) rather than owned by a single component.

	private static final Set<String> GO_KEYWORDS = Set.of(
			"break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct",
			"chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range", "type",
			"continue", "for", "import", "return", "var");

	// A Java local named "string" (common in toString() methods) legally shadows Go's builtin
	// string type for the rest of the function, breaking a later `func() string {...}` closure.
	private static final Set<String> GO_BUILTIN_TYPE_NAMES = Set.of(
			"string", "error", "any", "byte", "rune", "bool");

	// A Go keyword/builtin type name, or a name that shadows the enclosing class's own Go type
	// (id.java's `id(id id)` ctor: param "id" would hide the type "id" for &id{} in its body).
	String sanitizeIdent(String javaName) {
		if (GO_KEYWORDS.contains(javaName) || GO_BUILTIN_TYPE_NAMES.contains(javaName)) return javaName + "_";
		if (javaName.equals(currentClassGoTypeName)) return javaName + "_";
		return javaName;
	}

	String paramList(IMethodBinding mb, MethodDeclaration mdOrNull) {
		ITypeBinding[] types = mb.getParameterTypes();
		List<String> names_ = new ArrayList<>();
		if (mdOrNull != null) {
			for (Object p : mdOrNull.parameters()) {
				names_.add(((SingleVariableDeclaration) p).getName().getIdentifier());
			}
		}
		List<String> parts = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			String n = i < names_.size() ? sanitizeIdent(names_.get(i)) : "a" + i;
			parts.add(n + " " + dev.gowt.j2go.GoTypes.map(types[i], this));
		}
		return String.join(", ", parts);
	}

	String argNames(MethodDeclaration md) {
		List<String> n = new ArrayList<>();
		for (Object p : md.parameters()) n.add(sanitizeIdent(((SingleVariableDeclaration) p).getName().getIdentifier()));
		return String.join(", ", n);
	}

	String retType(IMethodBinding mb) {
		String t = dev.gowt.j2go.GoTypes.map(mb.getReturnType(), this);
		return t.isEmpty() ? "" : t;
	}

	/** Adds the import a manual (jrt.*) type's Go spelling needs, if any. */
	public void addManualImport(String manualQualifiedName) {
		String imp = dev.gowt.j2go.Manual.importPath(manualQualifiedName);
		if (imp != null) fileImports.add(imp);
	}

	/** ".impl" in-package, ".Impl()" cross-package (id_manual.go's accessor - the field itself
	 * is unexported). */
	String implAccess(TypeModel.ClassInfo rootCi) {
		return rootCi.goPackage.equals(currentGoPackage) ? ".impl" : ".Impl()";
	}

	// ---------------------------------------------------------------- cross-package qualification
	// swt referencing a cocoa class needs "cocoa." + an import; cocoa referencing swt is a
	// contract violation (see README "Multiple Go packages").

	/** ci.goTypeName, qualified for use from the file currently being emitted. Capitalized when
	 * actually crossing packages: Go visibility is the identifier's own case, and 2 translated
	 * types (id, objc_super) have a lowercase Java class name - see internal/cocoa/id_manual.go's
	 * Id/ObjcSuper aliases, the exported names a cross-package reference must use instead. */
	public String qualifiedTypeName(TypeModel.ClassInfo ci) {
		return qualify(ci.goTypeName, ci);
	}

	/** ci.goFuncPrefix, qualified the same way - valid only where goFuncPrefix is the FIRST
	 * token of the name being built (a package qualifier must lead the whole identifier). */
	String qualifiedFuncPrefix(TypeModel.ClassInfo ci) {
		return qualify(ci.goFuncPrefix, ci);
	}

	// The 2 lowercase Java class names in this codebase (id, objc_super) need more than a bare
	// first-letter capitalize to reach their hand-written exported alias (id_manual.go's
	// Id/ObjcSuper) - "objc_super" capitalized is "Objc_super", not "ObjcSuper".
	private static final java.util.Map<String, String> LOWERCASE_ALIASES = java.util.Map.of(
			"id", "Id", "objc_super", "ObjcSuper");

	/** Qualifies an already-built bare identifier, e.g. "New" + ci.goFuncPrefix. */
	String qualify(String bareIdent, TypeModel.ClassInfo ci) {
		String prefix = packagePrefix(ci.goPackage);
		if (prefix.isEmpty()) return bareIdent;
		String alias = LOWERCASE_ALIASES.get(bareIdent);
		return prefix + (alias != null ? alias : Names.capitalize(bareIdent));
	}

	private String packagePrefix(String targetGoPackage) {
		if (targetGoPackage.equals(currentGoPackage)) return "";
		if (currentGoPackage.equals("cocoa")) {
			System.err.println("j2go: guard violated: cocoa file " + currentJavaPackage
					+ " must not reference a " + targetGoPackage + " type");
			System.exit(1);
		}
		fileImports.add(dev.gowt.j2go.GoTypes.COCOA_IMPORT);
		return targetGoPackage + ".";
	}

	/** Same guard, for a type GoTypes.map could not resolve at all (not in model, not manual). */
	public void checkNoForeignPackageLeak(String qualifiedJavaTypeName) {
		if (!currentGoPackage.equals("cocoa")) return;
		int dot = qualifiedJavaTypeName.lastIndexOf('.');
		String javaPackage = dot < 0 ? "" : qualifiedJavaTypeName.substring(0, dot);
		if (javaPackage.startsWith("org.eclipse.swt") && !dev.gowt.j2go.GoTypes.isCocoaPackage(javaPackage)) {
			System.err.println("j2go: guard violated: cocoa file " + currentJavaPackage
					+ " must not reference swt type " + qualifiedJavaTypeName);
			System.exit(1);
		}
	}
}
