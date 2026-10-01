package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.util.*;

/**
 * Struct-typed parameters a method body writes through. Java passes the object by reference, so the
 * caller sees the writes; a Go struct is copied, so such a parameter is a pointer and its call sites
 * pass the address. Writing means a field assignment or ++/--, or handing the parameter to a native
 * (which fills it) or to another method that writes through its own parameter.
 */
public final class MutatedStructParams {
	private final TypeModel model;
	private final Map<String, Set<Integer>> memo = new HashMap<>();
	private final Set<String> inProgress = new HashSet<>();

	MutatedStructParams(TypeModel model) {
		this.model = model;
	}

	// Every declaration by name and parameter types: an interface method and its implementations (or
	// overrides) must agree on the signature, so one writer makes the parameter a pointer for all.
	private Map<String, List<MethodDeclaration>> family;

	private static String familyKey(IMethodBinding mb) {
		StringBuilder k = new StringBuilder(mb.getName()).append('(');
		for (ITypeBinding t : mb.getMethodDeclaration().getParameterTypes()) k.append(t.getErasure().getQualifiedName()).append(',');
		return k.toString();
	}

	/** Whether parameter index of mb, or of any declaration with the same name and parameter types, is written through. */
	public boolean isMutated(IMethodBinding mb, int index) {
		// cocoa has three such parameters too (FixRect, SendMeasureItem): left by value until a GUI run on the Mac can check them.
		if (!GoTypes.piPackage.equals("gtk") || mb == null || Modifier.isNative(mb.getModifiers()) || mb.isConstructor() || index >= mb.getParameterTypes().length) return false;
		TypeModel.ClassInfo ci = model.lookup(mb.getParameterTypes()[index]);
		if (ci == null || !ci.isStruct) return false;
		if (family == null) {
			family = new HashMap<>();
			for (MethodDeclaration md : model.allDeclarations()) {
				IMethodBinding b = md.resolveBinding();
				if (b != null && !md.isConstructor()) family.computeIfAbsent(familyKey(b), k -> new ArrayList<>()).add(md);
			}
		}
		for (MethodDeclaration md : family.getOrDefault(familyKey(mb), List.of())) if (mutated(md).contains(index)) return true;
		return false;
	}

	/** Whether v is a parameter that isMutated says is passed as a pointer. */
	public boolean isMutatedParam(IVariableBinding v) {
		if (v == null || !v.isParameter() || v.getDeclaringMethod() == null) return false;
		MethodDeclaration md = model.declarationOf(v.getDeclaringMethod());
		if (md == null) return false;
		for (int i = 0; i < md.parameters().size(); i++) {
			IVariableBinding p = ((SingleVariableDeclaration) md.parameters().get(i)).resolveBinding();
			if (p != null && p.isEqualTo(v)) return isMutated(v.getDeclaringMethod(), i);
		}
		return false;
	}

	private Set<Integer> mutated(MethodDeclaration md) {
		IMethodBinding mb = md.resolveBinding();
		if (mb == null || md.getBody() == null) return Set.of();
		String key = Names.erasureKey(mb);
		Set<Integer> done = memo.get(key);
		if (done != null) return done;
		if (!inProgress.add(key)) return Set.of();
		Map<IVariableBinding, Integer> params = new HashMap<>();
		for (int i = 0; i < md.parameters().size(); i++) {
			IVariableBinding v = ((SingleVariableDeclaration) md.parameters().get(i)).resolveBinding();
			TypeModel.ClassInfo ci = v == null ? null : model.lookup(v.getType());
			if (ci != null && ci.isStruct) params.put(v, i);
		}
		Set<Integer> out = new HashSet<>();
		Set<Integer> bare = new HashSet<>(); // parameters also used as a whole value (null test, assignment, ...): they stay by value
		if (!params.isEmpty()) {
			md.getBody().accept(new ASTVisitor() {
				private void write(Expression target) {
					Expression owner = target instanceof QualifiedName q ? q.getQualifier()
							: target instanceof FieldAccess f ? f.getExpression() : null;
					if (owner instanceof Name n && n.resolveBinding() instanceof IVariableBinding v && params.containsKey(v)) out.add(params.get(v));
				}

				@Override public boolean visit(SimpleName n) {
					if (!(n.resolveBinding() instanceof IVariableBinding v) || !params.containsKey(v) || n.getLocationInParent() == SingleVariableDeclaration.NAME_PROPERTY) return true;
					ASTNode parent = n.getParent();
					boolean fieldOwner = parent instanceof QualifiedName q && q.getQualifier() == n || parent instanceof FieldAccess f && f.getExpression() == n;
					boolean writtenArg = false;
					if (parent instanceof MethodInvocation mi && mi.arguments().contains(n) && mi.resolveMethodBinding() != null) {
						IMethodBinding callee = mi.resolveMethodBinding();
						int i = mi.arguments().indexOf(n);
						writtenArg = Modifier.isNative(callee.getModifiers()) ? model.isNativeStructPointerParam(callee, i) : isMutated(callee, i);
					}
					if (!fieldOwner && !writtenArg) bare.add(params.get(v));
					return true;
				}

				@Override public boolean visit(Assignment a) { write(a.getLeftHandSide()); return true; }
				@Override public boolean visit(PostfixExpression e) { write(e.getOperand()); return true; }
				@Override public boolean visit(PrefixExpression e) { write(e.getOperand()); return true; }

				@Override public boolean visit(MethodInvocation mi) {
					IMethodBinding callee = mi.resolveMethodBinding();
					for (int i = 0; callee != null && i < mi.arguments().size(); i++) {
						if (!(mi.arguments().get(i) instanceof Name n) || !(n.resolveBinding() instanceof IVariableBinding v) || !params.containsKey(v)) continue;
						boolean writes = Modifier.isNative(callee.getModifiers()) ? model.isNativeStructPointerParam(callee, i) : isMutated(callee, i);
						if (writes) out.add(params.get(v));
					}
					return true;
				}
			});
		}
		out.removeAll(bare);
		inProgress.remove(key);
		memo.put(key, out);
		return out;
	}
}
