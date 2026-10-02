package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.IMethodBinding;

import java.util.ArrayList;
import java.util.List;

/** The <root>Hooked wrapper of a split cascade: SetImpl_ installs it around an anonymous subclass from
 * another Go package, which overrides a public/protected method by an exported name that the exported
 * API of the root does not have. While a hook runs, a call of the same method reaches the default (super). */
final class HookEmitter {
	private HookEmitter() {}

	static void emit(Emitter emitter, TypeModel.ClassInfo root, StringBuilder out) {
		String impl = root.goTypeName + "Impl";
		String hooked = Names.decapitalize(root.goTypeName) + "Hooked";
		out.append("// j2go: wraps a subclass from another package; its exported hook names override the defaults.\n");
		out.append("type ").append(hooked).append(" struct {\n\t").append(impl).append("\n\thook   ").append(impl)
				.append("\n\tactive string\n}\n\n");
		out.append("func (this *").append(hooked).append(") enter(name string) func() {\n")
				.append("\tprev := this.active\n\tthis.active = name\n\treturn func() { this.active = prev }\n}\n\n");
		for (var e : root.overriddenRootHookNames.entrySet()) {
			IMethodBinding decl = root.overriddenRootMethods.get(e.getKey());
			String dispatch = root.overriddenRootMethodGoNames.get(e.getKey());
			String params = emitter.paramList(decl, null);
			String ret = emitter.retType(decl);
			List<String> args = new ArrayList<>();
			for (int i = 0; i < decl.getParameterTypes().length; i++) args.add("a" + i);
			String argList = String.join(", ", args);
			String r = ret.isEmpty() ? "" : "return ";
			out.append("func (this *").append(hooked).append(") ").append(dispatch).append('(').append(params).append(") ")
					.append(ret).append(ret.isEmpty() ? "" : " ").append("{\n");
			// An abstract declaration has no super call to route to the default, so a re-entrant call (composite.layout() from
			// inside Layout.layout) must reach the override again.
			boolean guard = !java.lang.reflect.Modifier.isAbstract(decl.getModifiers());
			out.append("\tif h, ok := this.hook.(interface{ ").append(e.getValue()).append('(').append(params).append(") ")
					.append(ret).append(" }); ok").append(guard ? " && this.active != \"" + dispatch + "\"" : "").append(" {\n");
			if (guard) out.append("\t\tdefer this.enter(\"").append(dispatch).append("\")()\n");
			out.append("\t\t").append(r).append("h.").append(e.getValue()).append('(').append(argList).append(")\n");
			if (ret.isEmpty()) out.append("\t\treturn\n"); // a void hook replaces the default, it does not precede it
			out.append("\t}\n");
			out.append('\t').append(r).append("this.").append(impl).append('.').append(dispatch).append('(').append(argList).append(")\n}\n\n");
		}
	}

	/** A class with a cascade of its own that also overrides a base from another package: the dispatch method keeps its
	 * unexported name, the base's exported hook name forwards into it. */
	static void wrapper(Emitter emitter, TypeModel.ClassInfo ci, String hookName, String dispatch, IMethodBinding sig,
			org.eclipse.jdt.core.dom.MethodDeclaration md, StringBuilder out) {
		String ret = emitter.retType(sig);
		out.append("func (this *").append(ci.goTypeName).append(") ").append(hookName).append('(').append(emitter.paramList(sig, md)).append(") ")
				.append(ret).append(" {\n\t").append(ret.isEmpty() ? "" : "return ").append("this.impl.").append(dispatch)
				.append('(').append(emitter.argNames(md)).append(")\n}\n\n");
	}
}
