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
		// "As<GoTypeName>"/"<GoTypeName>Like" - swt-package, non-struct, non-interface classes
		// only (see README "Round 7 api"); null otherwise. Collision-resolved once in build().
		public String asMethodName;
		public String likeInterfaceName;
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

		// swt's public API: cascade methods dispatch through unexported names, exported names are
		// wrappers (README "Round 9 api"). internal/cocoa keeps exported cascade names.
		public boolean splitsDispatch() {
			return root.goPackage.equals("swt");
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

	// Native method (Names.erasureKey) -> indexes of its struct params JNI passes by pointer: every
	// struct param whose Javadoc lacks `flags=struct` (memmove's dest, objc_msgSendSuper's super).
	private final Map<String, Set<Integer>> nativeStructPointerParams = new HashMap<>();

	public boolean isNativeStructPointerParam(IMethodBinding mb, int index) {
		Set<Integer> s = nativeStructPointerParams.get(Names.erasureKey(mb));
		if (s == null || !s.contains(index)) return false;
		ClassInfo ci = lookup(mb.getParameterTypes()[index]);
		return ci != null && ci.isStruct;
	}

	// Methods bound via Type::method (Names.erasureKey) anywhere in the file set - the target's
	// signature is that functional interface's SAM, so "Round 7 api" must not widen its params.
	private final Set<String> methodReferenceTargets = new HashSet<>();

	public boolean isMethodReferenceTarget(IMethodBinding mb) {
		return methodReferenceTargets.contains(Names.erasureKey(mb));
	}

	/** Text of `tag` in md's Javadoc: for @param, only the entry naming paramName. */
	public static String javadocTag(MethodDeclaration md, String tag, String paramName) {
		if (md.getJavadoc() == null) return "";
		for (Object o : md.getJavadoc().tags()) {
			TagElement te = (TagElement) o;
			if (!tag.equals(te.getTagName())) continue;
			List<?> frags = te.fragments();
			StringBuilder sb = new StringBuilder();
			int start = 0;
			if (paramName != null) {
				if (frags.isEmpty() || !(frags.get(0) instanceof SimpleName sn) || !sn.getIdentifier().equals(paramName)) continue;
				start = 1;
			}
			for (int i = start; i < frags.size(); i++) sb.append(frags.get(i).toString());
			return sb.toString();
		}
		return "";
	}

	private void recordNativeStructParams(MethodDeclaration md, IMethodBinding mb) {
		Set<Integer> idx = new HashSet<>();
		for (int i = 0; i < md.parameters().size(); i++) {
			SingleVariableDeclaration p = (SingleVariableDeclaration) md.parameters().get(i);
			ITypeBinding t = mb.getParameterTypes()[i];
			if (t.isPrimitive() || t.isArray()) continue;
			if (!javadocTag(md, "@param", p.getName().getIdentifier()).contains("struct")) idx.add(i);
		}
		if (!idx.isEmpty()) nativeStructPointerParams.put(Names.erasureKey(mb), idx);
	}

	public ClassInfo lookup(ITypeBinding t) {
		return byBinaryName.get(t.getErasure().getBinaryName());
	}

	public ClassInfo lookupBinaryName(String binaryName) {
		return byBinaryName.get(binaryName);
	}

	public Collection<ClassInfo> all() {
		return byBinaryName.values();
	}

	public void build(List<CompilationUnit> units, Names names) {
		for (CompilationUnit cu : units) {
			for (Object t : cu.types()) {
				collect((AbstractTypeDeclaration) t, names);
			}
			cu.accept(new ASTVisitor() {
				@Override
				public boolean visit(ExpressionMethodReference node) {
					IMethodBinding mb = node.resolveMethodBinding();
					if (mb != null) methodReferenceTargets.add(Names.erasureKey(mb));
					return true;
				}
			});
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
			// Keyed by computed Go name, not bare Java name: TreeItem.getBounds(int) is GetBoundsIndex
			// and must not push Control.getBounds() off its plain GetBounds.
			Map<String, Set<String>> sigsByGoName = new LinkedHashMap<>();
			Set<String> treeTypeNames = new LinkedHashSet<>();
			for (ClassInfo c : tree) {
				treeTypeNames.add(c.goTypeName);
				for (IMethodBinding m : c.declaredMethods.values()) {
					String goName = names.goMemberName(m, Names.javaMethodBaseGoName(m.getName()));
					sigsByGoName.computeIfAbsent(goName, k -> new LinkedHashSet<>()).add(signature(m));
				}
			}
			// Group this root's cascade methods by their pre-suffix base Go name (two chains
			// can land on the same one) before naming them - see assignCollidingCascadeNames.
			Map<String, List<Map.Entry<String, IMethodBinding>>> byBase = new LinkedHashMap<>();
			for (var e : ci.overriddenRootMethods.entrySet()) {
				String base = names.goMemberName(e.getValue(), Names.javaMethodBaseGoName(e.getValue().getName()));
				byBase.computeIfAbsent(base, k -> new ArrayList<>()).add(e);
			}
			for (var g : byBase.entrySet()) {
				String base = g.getKey();
				// Split dispatch names are unexported, so only other cascade members can collide.
				int users = ci.splitsDispatch() ? g.getValue().size() : sigsByGoName.get(base).size();
				if (users <= 1) {
					putCascadeName(ci, treeTypeNames, g.getValue().get(0), base);
					continue;
				}
				assignCollidingCascadeNames(ci, treeTypeNames, base, g.getValue());
			}
		}
		resolveCrossFamilyNameCollisions(names);
		assignLikeNames(names);
	}

	private void putCascadeName(ClassInfo ci, Set<String> treeTypeNames, Map.Entry<String, IMethodBinding> e, String name) {
		if (ci.splitsDispatch()) {
			ci.overriddenRootMethodGoNames.put(e.getKey(), Names.decapitalize(name) + "_");
			return;
		}
		// A cascade name equal to a subclass's embedded field name - Go rejects a field and
		// method sharing a name.
		if (treeTypeNames.contains(name)) name = name + "Fn";
		ci.overriddenRootMethodGoNames.put(e.getKey(), name);
	}

	// "<base>On<class>" alone isn't unique when two members share both base and declaring class
	// (Control's setBackground(Color) and setBackgroundColor(NSColor) both -> SetBackgroundColor).
	// The member whose bare Java name literally is the base claims the plain spelling; the rest
	// append their own parameter types, then a stable ordinal if that still repeats.
	private void assignCollidingCascadeNames(ClassInfo ci, Set<String> treeTypeNames, String base,
			List<Map.Entry<String, IMethodBinding>> members) {
		members.sort(Comparator.comparingInt(e ->
				base.equals(Names.javaMethodBaseGoName(e.getValue().getName())) ? 0 : 1));
		Set<String> used = new HashSet<>();
		for (var e : members) {
			IMethodBinding decl = e.getValue();
			String declClass = lookup(decl.getDeclaringClass()).goTypeName;
			String candidate = base + "On" + declClass;
			if (!used.add(candidate)) {
				candidate = base + "On" + declClass + paramTypeTag(decl);
				int ordinal = 2;
				while (!used.add(candidate)) candidate = base + "On" + declClass + paramTypeTag(decl) + ordinal++;
			}
			putCascadeName(ci, treeTypeNames, e, candidate);
		}
	}

	private static String paramTypeTag(IMethodBinding decl) {
		StringBuilder sb = new StringBuilder();
		for (ITypeBinding t : decl.getParameterTypes()) sb.append(Names.capitalize(t.getErasure().getName()));
		return sb.toString();
	}

	// Every non-struct, non-interface swt-package class gets an upcast accessor + a 1-method
	// interface every subclass satisfies via embedding (see README "Round 7 api").
	private void assignLikeNames(Names names) {
		Set<String> swtTypeNames = new HashSet<>();
		for (ClassInfo c : byBinaryName.values()) {
			if (c.goPackage.equals("swt")) swtTypeNames.add(c.goTypeName);
		}
		for (ClassInfo ci : byBinaryName.values()) {
			if (!ci.goPackage.equals("swt") || ci.isInterface || ci.isStruct) continue;
			String as = "As" + ci.goTypeName;
			for (IMethodBinding mb : ci.declaredMethods.values()) {
				if (names.goMemberName(mb, Names.javaMethodBaseGoName(mb.getName())).equals(as)) {
					as = as + "_";
					break;
				}
			}
			ci.asMethodName = as;
			String like = ci.goTypeName + "Like";
			ci.likeInterfaceName = swtTypeNames.contains(like) ? like + "_" : like;
		}
	}

	// Safety net for two method-name families on one class computing the same Go name that
	// Names.goMemberName's "With" rule doesn't catch (suffix vs suffix). In internal/cocoa a
	// cascade name claims first: a non-cascade sibling must not collide with it either.
	private void resolveCrossFamilyNameCollisions(Names names) {
		for (ClassInfo ci : byBinaryName.values()) {
			// With split dispatch every method keeps its natural exported name, cascade or not.
			boolean split = ci.splitsDispatch();
			Map<String, IMethodBinding> claimedBy = new LinkedHashMap<>();
			for (IMethodBinding mb : ci.declaredMethods.values()) {
				if (split || ci.overridePoint(signature(mb)) == null) continue;
				String cascadeName = ci.root.overriddenRootMethodGoNames.get(signature(mb));
				if (cascadeName != null) claimedBy.put(cascadeName, mb);
			}
			int idx = 0;
			for (IMethodBinding mb : ci.declaredMethods.values()) {
				idx++;
				if (!split && ci.overridePoint(signature(mb)) != null) continue;
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
		if (Manual.isManual(binding.getErasure().getQualifiedName())) return; // hand-written nested class
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
				if (Modifier.isNative(md.getModifiers())) recordNativeStructParams(md, mb);
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
		// Only the cocoa C-struct mirrors (NSRect, ...) are Go value types; an swt data class
		// (DeviceData) is nullable in Java and stays a pointer.
		ci.isStruct = !ci.isInterface && superIsObject && onlyToString && GoTypes.isCocoaPackage(ci.javaPackage);
	}
}
