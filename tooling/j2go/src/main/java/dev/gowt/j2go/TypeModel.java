package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.util.*;

/**
 * Hierarchy of the translated-set classes: Go names, superclass links, and the set of
 * methods overridden anywhere in a hierarchy (needed for the impl-interface dispatch pattern).
 */
class TypeModel {

	static class ClassInfo {
		ITypeBinding binding;
		String binaryName;
		String goTypeName;
		String goFuncPrefix;
		ClassInfo superclass;
		ClassInfo root;
		final List<ClassInfo> children = new ArrayList<>();
		final Set<String> declaredMethodNames = new LinkedHashSet<>();
		// only meaningful on the root: methodName -> root's own declaration of it.
		final Map<String, IMethodBinding> overriddenRootMethods = new LinkedHashMap<>();

		boolean isOverridden(String methodName) {
			return root.overriddenRootMethods.containsKey(methodName);
		}

		boolean hierarchyHasSubclasses() {
			return this == root && !root.children.isEmpty() || (root != this);
		}
	}

	private final Map<String, ClassInfo> byBinaryName = new LinkedHashMap<>();

	ClassInfo lookup(ITypeBinding t) {
		return byBinaryName.get(t.getErasure().getBinaryName());
	}

	Collection<ClassInfo> all() {
		return byBinaryName.values();
	}

	void build(List<CompilationUnit> units, Names names) {
		for (CompilationUnit cu : units) {
			for (Object t : cu.types()) {
				collect((AbstractTypeDeclaration) t, names);
			}
		}
		for (ClassInfo ci : byBinaryName.values()) {
			ITypeBinding superBinding = ci.binding.getSuperclass();
			ci.superclass = superBinding == null ? null : byBinaryName.get(superBinding.getErasure().getBinaryName());
			if (ci.superclass != null) ci.superclass.children.add(ci);
		}
		for (ClassInfo ci : byBinaryName.values()) {
			ClassInfo r = ci;
			while (r.superclass != null) r = r.superclass;
			ci.root = r;
		}
		for (ClassInfo ci : byBinaryName.values()) {
			if (ci.superclass == null) continue;
			for (String name : ci.declaredMethodNames) {
				ClassInfo anc = ci.superclass;
				while (anc != null) {
					if (anc.declaredMethodNames.contains(name)) {
						IMethodBinding rootDecl = findMethodBinding(ci.root.binding, name);
						ci.root.overriddenRootMethods.put(name, rootDecl);
						break;
					}
					anc = anc.superclass;
				}
			}
		}
	}

	private IMethodBinding findMethodBinding(ITypeBinding type, String name) {
		for (IMethodBinding m : type.getDeclaredMethods()) {
			if (m.getName().equals(name)) return m;
		}
		return null;
	}

	private void collect(AbstractTypeDeclaration decl, Names names) {
		if (!(decl instanceof TypeDeclaration td)) return;
		ITypeBinding binding = td.resolveBinding();
		ClassInfo ci = new ClassInfo();
		ci.binding = binding;
		ci.binaryName = binding.getErasure().getBinaryName();
		ci.goTypeName = Names.goTypeName(binding);
		ci.goFuncPrefix = Names.goFuncPrefix(binding);
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
				}
			} else if (o instanceof TypeDeclaration nested) {
				collect(nested, names);
			}
		}
	}
}
