package dev.gowt.j2go.emit;

import dev.gowt.j2go.TypeModel;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.Set;

/** The instanceof helpers committed shared files call but the reference platform defined in its own files. */
public final class HelperFiller {

	private HelperFiller() {}

	/** Source of the {root}ImplAs{target} helpers among `referenced` that nothing has defined yet. */
	public static String fill(Emitter emitter, Collection<String> referenced) {
		Set<String> wanted = Set.copyOf(referenced);
		emitter.fileHelperSource = new ArrayList<>();
		emitter.currentGoPackage = "swt";
		for (TypeModel.ClassInfo root : emitter.model.all()) {
			if (root != root.root) continue;
			List<TypeModel.ClassInfo> tree = new ArrayList<>();
			collect(root, tree);
			for (TypeModel.ClassInfo target : tree) {
				if (wanted.contains(TypeTestEmitter.cascadeHelperName(root, target))) emitter.ensureCascadeHelper(root, target);
				if (target.isStruct || target.isInterface) continue;
				for (TypeModel.ClassInfo from : tree) {
					if (from.isStruct || from.isInterface || from == target) continue;
					String pair = from.goTypeName + "To" + target.goTypeName;
					if (wanted.contains("upcast" + pair)) emitter.upcastObject("x", from.binding, target.binding);
					if (wanted.contains("is" + pair)) new TypeTestEmitter(emitter).nilSafeInstanceof(from.binding, from, target);
				}
			}
		}
		return String.join("", emitter.fileHelperSource);
	}

	private static void collect(TypeModel.ClassInfo ci, List<TypeModel.ClassInfo> out) {
		out.add(ci);
		for (TypeModel.ClassInfo c : ci.children) collect(c, out);
	}
}
