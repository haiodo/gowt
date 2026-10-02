package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.Modifier;

import java.util.ArrayList;
import java.util.List;

/** Erased generics lose the self type: `ButtonFactory.tooltip()` is declared `F tooltip()` in a generic base, so in Go it
 * returns the base's bound and a chain breaks at the first inherited call. A non-generic class whose direct superclass is
 * parameterized therefore gets a typed forwarding method for every public inherited method whose Go signature differs once
 * the type arguments are substituted (README "Round 22 jface"). Same Go name as the base's, so it shadows it. */
final class CovariantBridges {

	private CovariantBridges() {}

	/** The inherited method as `cls` sees it (substituted types) next to its generic declaration (erased types). */
	private record Bridge(IMethodBinding seen, IMethodBinding declared) {}

	private static List<Bridge> bridges(Emitter emitter, ITypeBinding cls) {
		List<Bridge> out = new ArrayList<>();
		ITypeBinding sup = cls.getSuperclass();
		if (cls.isGenericType() || cls.isParameterizedType() || cls.isInterface() || sup == null || !sup.isParameterizedType()) return out;
		TypeModel.ClassInfo supCi = emitter.model.lookup(sup), ci = emitter.model.lookup(cls);
		if (supCi == null || ci == null || !supCi.goPackage.equals(ci.goPackage) || ci.isStruct) return out;
		List<ITypeBinding> below = new ArrayList<>(List.of(cls));
		for (ITypeBinding a = sup; a != null && emitter.model.lookup(a) != null; a = a.getSuperclass()) {
			for (IMethodBinding am : a.getDeclaredMethods()) {
				int mods = am.getModifiers();
				if (am.isConstructor() || Modifier.isStatic(mods) || !Modifier.isPublic(mods) || am.getMethodDeclaration().getTypeParameters().length > 0
						|| overriddenBelow(below, am) || !differs(emitter, am)) continue;
				// A nearer ancestor's declaration of the same signature wins.
				String sig = TypeModel.signature(am.getMethodDeclaration());
				if (out.stream().noneMatch(b -> TypeModel.signature(b.declared).equals(sig))) out.add(new Bridge(am, am.getMethodDeclaration()));
			}
			below.add(a);
		}
		return out;
	}

	private static boolean overriddenBelow(List<ITypeBinding> below, IMethodBinding am) {
		for (ITypeBinding b : below) for (IMethodBinding bm : b.getDeclaredMethods()) if (bm.overrides(am)) return true;
		return false;
	}

	private static boolean differs(Emitter emitter, IMethodBinding seen) {
		IMethodBinding decl = seen.getMethodDeclaration();
		if (!GoTypes.map(decl.getReturnType(), emitter).equals(GoTypes.map(seen.getReturnType(), emitter))) return true;
		for (int i = 0; i < decl.getParameterTypes().length; i++) {
			if (!GoTypes.map(decl.getParameterTypes()[i], emitter).equals(GoTypes.map(seen.getParameterTypes()[i], emitter))) return true;
		}
		return false;
	}

	/** Does Go's method lookup on a `recv`-typed value find a typed forwarder for the call (so it needs no cast)? */
	static boolean covers(Emitter emitter, ITypeBinding recv, IMethodBinding call) {
		String key = call.getMethodDeclaration().getKey();
		for (ITypeBinding c = recv; c != null && !c.isTypeVariable() && !c.isGenericType() && !c.isParameterizedType(); c = c.getSuperclass()) {
			ITypeBinding sup = c.getSuperclass();
			if (sup != null && sup.isParameterizedType()) return bridges(emitter, c).stream().anyMatch(b -> b.declared.getKey().equals(key));
		}
		return false;
	}

	static String emit(Emitter emitter, TypeModel.ClassInfo ci) {
		StringBuilder out = new StringBuilder();
		String field = emitter.model.lookup(ci.binding.getSuperclass()) == null ? null : emitter.model.lookup(ci.binding.getSuperclass()).goTypeName;
		for (Bridge b : bridges(emitter, ci.binding)) {
			List<String> pre = new ArrayList<>(), args = new ArrayList<>();
			String params = EmitUtil.publicParamList(emitter, b.seen, null, pre);
			for (int i = 0; i < b.seen.getParameterTypes().length; i++) args.add("a" + i);
			String name = emitter.names.goMemberName(b.declared, Names.javaMethodBaseGoName(b.declared.getName()));
			String call = "this." + field + "." + name + "(" + String.join(", ", args) + ")";
			String ret = emitter.retType(b.seen);
			out.append("func (this *").append(ci.goTypeName).append(") ").append(name).append('(').append(params).append(") ").append(ret).append(" {\n");
			for (String p : pre) out.append('\t').append(p).append('\n');
			if (ret.isEmpty()) {
				out.append('\t').append(call).append('\n');
			} else {
				String narrowed = ret.equals(GoTypes.map(b.declared.getReturnType(), emitter)) ? call
						: emitter.typeTestEmitter.narrow(call, b.declared.getReturnType(), b.seen.getReturnType());
				if (narrowed == null) {
					emitter.fileImports.add(Manual.JRT_IMPORT);
					narrowed = "jrt.Cast[" + ret + "](" + call + ")";
				}
				out.append("\treturn ").append(narrowed).append('\n');
			}
			out.append("}\n\n");
		}
		return out.toString();
	}
}
