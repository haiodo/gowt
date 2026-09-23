package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.util.*;

/**
 * Hierarchy of the translated-set classes: Go names, superclass links, and the set of
 * methods overridden anywhere in a hierarchy (needed for the impl-interface dispatch pattern).
 */
public class TypeModel {

	public static class ClassInfo {
		public ITypeBinding binding;
		public String binaryName;
		public String goTypeName;
		public String goFuncPrefix;
		public ClassInfo superclass;
		public ClassInfo root;
		public boolean isInterface;
		// Go package this class lands in (GoTypes.goPackageOf) - qualifies a reference from
		// a different Go package (e.g. swt calling into cocoa).
		public String javaPackage;
		public String goPackage;
		/** Plain data class (extends Object, no methods besides toString): Go value type, not *T. */
		public boolean isStruct;
		// Non-null when the Java superclass is external but manual-embeddable (SWTException
		// extends RuntimeException) - see Manual.isManualSuper / README "Manual superclass embedding".
		public String manualSuperQualifiedName;
		public final List<ClassInfo> children = new ArrayList<>();
		final Set<String> declaredMethodNames = new LinkedHashSet<>();
		// signature ("name(erasedParamType,...)") -> binding, this class's own declarations only.
		public final Map<String, IMethodBinding> declaredMethods = new LinkedHashMap<>();
		// only meaningful on the root: signature of a chain's topmost declaration -> that
		// declaration's binding (not necessarily the root's own - see ClassInfo.overridePoint).
		public final Map<String, IMethodBinding> overriddenRootMethods = new LinkedHashMap<>();
		// same keys -> the Go name to use everywhere (two unrelated chains sharing a bare Java
		// name, e.g. Cocoa's "setValue", can't both be plain "SetValue" in one Go interface).
		public final Map<String, String> overriddenRootMethodGoNames = new LinkedHashMap<>();

		// Topmost ancestor (inclusive) declaring `sig` with a descendant also declaring it -
		// null for an unrelated overload of a same-named method (needs matching erased params).
		public ClassInfo overridePoint(String sig) {
			ClassInfo point = null;
			for (ClassInfo cur = this; cur != null; cur = cur.superclass) {
				if (cur.declaredMethods.containsKey(sig) && hasOverrideBelow(cur, sig)) point = cur;
			}
			return point;
		}

		private static boolean hasOverrideBelow(ClassInfo ci, String sig) {
			for (ClassInfo child : ci.children) {
				if (child.declaredMethods.containsKey(sig) || hasOverrideBelow(child, sig)) return true;
			}
			return false;
		}

		public IMethodBinding declaredBinding(String sig) {
			return declaredMethods.get(sig);
		}
	}

	public static String signature(IMethodBinding mb) {
		StringBuilder sb = new StringBuilder(mb.getName()).append('(');
		ITypeBinding[] params = mb.getParameterTypes();
		for (int i = 0; i < params.length; i++) {
			if (i > 0) sb.append(',');
			sb.append(params[i].getErasure().getBinaryName());
		}
		return sb.append(')').toString();
	}

	private final Map<String, ClassInfo> byBinaryName = new LinkedHashMap<>();

	public ClassInfo lookup(ITypeBinding t) {
		return byBinaryName.get(t.getErasure().getBinaryName());
	}

	public Collection<ClassInfo> all() {
		return byBinaryName.values();
	}

	public void build(List<CompilationUnit> units, Names names) {
		for (CompilationUnit cu : units) {
			for (Object t : cu.types()) {
				collect((AbstractTypeDeclaration) t, names);
			}
		}
		for (ClassInfo ci : byBinaryName.values()) {
			ITypeBinding superBinding = ci.binding.getSuperclass();
			ci.superclass = superBinding == null ? null : byBinaryName.get(superBinding.getErasure().getBinaryName());
			if (ci.superclass != null) ci.superclass.children.add(ci);
			if (ci.superclass == null && superBinding != null) {
				String q = superBinding.getErasure().getQualifiedName();
				if (Manual.isManualSuper(q)) ci.manualSuperQualifiedName = q;
			}
		}
		for (ClassInfo ci : byBinaryName.values()) {
			ClassInfo r = ci;
			while (r.superclass != null) r = r.superclass;
			ci.root = r;
		}
		// Root-level maps: interface/default-stub generation only (Emitter). Dispatch itself
		// goes through ClassInfo.overridePoint below.
		for (ClassInfo ci : byBinaryName.values()) {
			for (String sig : ci.declaredMethods.keySet()) {
				ClassInfo point = ci.overridePoint(sig);
				if (point != null) ci.root.overriddenRootMethods.put(sig, point.declaredBinding(sig));
			}
		}
		for (ClassInfo ci : byBinaryName.values()) {
			if (ci != ci.root) continue;
			// A wide tree can reuse a bare Java name for an unrelated method anywhere below the
			// root; every class shares one `impl` field, so the Go name must be unique tree-wide.
			List<ClassInfo> tree = new ArrayList<>();
			collectAllDescendants(ci, tree);
			Map<String, Set<String>> sigsByBareName = new LinkedHashMap<>();
			for (ClassInfo c : tree) {
				for (IMethodBinding m : c.declaredMethods.values()) {
					sigsByBareName.computeIfAbsent(m.getName(), k -> new LinkedHashSet<>()).add(signature(m));
				}
			}
			for (var e : ci.overriddenRootMethods.entrySet()) {
				IMethodBinding decl = e.getValue();
				String base = names.goMemberName(decl, Names.javaMethodBaseGoName(decl.getName()));
				boolean collides = sigsByBareName.get(decl.getName()).size() > 1;
				String finalName = collides ? base + "On" + lookup(decl.getDeclaringClass()).goTypeName : base;
				ci.overriddenRootMethodGoNames.put(e.getKey(), finalName);
			}
		}
		resolveCrossFamilyNameCollisions(names);
	}

	// Two unrelated method-name families declared on the same class can independently compute
	// the same Go name (Control's setBackground()/setBackground(Color) overload suffix
	// "SetBackground"+"Color" collides with the separately-named setBackgroundColor's own base
	// name) - goMemberName has no visibility into sibling families to catch this on its own, so
	// it is resolved here, once per class, with every declared method in hand. Cascade
	// (overridden) methods are named separately (see above) and skipped here.
	private void resolveCrossFamilyNameCollisions(Names names) {
		for (ClassInfo ci : byBinaryName.values()) {
			Map<String, IMethodBinding> claimedBy = new LinkedHashMap<>();
			int idx = 0;
			for (IMethodBinding mb : ci.declaredMethods.values()) {
				idx++;
				if (ci.overridePoint(signature(mb)) != null) continue;
				String candidate = names.goMemberName(mb, Names.javaMethodBaseGoName(mb.getName()));
				IMethodBinding owner = claimedBy.putIfAbsent(candidate, mb);
				if (owner != null && !owner.getName().equals(mb.getName())) {
					names.addOverride(Names.erasureKey(mb), candidate + "_" + idx);
				}
			}
		}
	}

	private static void collectAllDescendants(ClassInfo ci, List<ClassInfo> out) {
		out.add(ci);
		for (ClassInfo c : ci.children) collectAllDescendants(c, out);
	}

	private void collect(AbstractTypeDeclaration decl, Names names) {
		if (!(decl instanceof TypeDeclaration td)) return;
		ITypeBinding binding = td.resolveBinding();
		ClassInfo ci = new ClassInfo();
		ci.binding = binding;
		ci.binaryName = binding.getErasure().getBinaryName();
		ci.goTypeName = Names.goTypeName(binding);
		ci.goFuncPrefix = Names.goFuncPrefix(binding);
		ci.isInterface = td.isInterface();
		ci.javaPackage = binding.getPackage() == null ? "" : binding.getPackage().getName();
		ci.goPackage = GoTypes.goPackageOf(ci.javaPackage);
		byBinaryName.put(ci.binaryName, ci);

		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md) {
				IMethodBinding mb = md.resolveBinding();
				if (mb == null) continue;
				List<String> paramNames = new ArrayList<>();
				for (Object p : md.parameters()) {
					paramNames.add(((SingleVariableDeclaration) p).getName().getIdentifier());
				}
				names.registerDeclaration(mb, paramNames);
				if (!md.isConstructor() && !Modifier.isStatic(md.getModifiers())) {
					ci.declaredMethodNames.add(md.getName().getIdentifier());
					ci.declaredMethods.put(signature(mb), mb);
				}
			} else if (o instanceof TypeDeclaration nested) {
				collect(nested, names);
			}
		}
		boolean superIsObject = binding.getSuperclass() == null
				|| binding.getSuperclass().getQualifiedName().equals("java.lang.Object");
		boolean onlyToString = ci.declaredMethodNames.isEmpty()
				|| ci.declaredMethodNames.equals(Set.of("toString"));
		ci.isStruct = !ci.isInterface && superIsObject && onlyToString;
	}
}
