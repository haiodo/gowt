package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;

import java.util.Map;

/** Cross-package qualification: a reference from another Go package needs "<pkg>." + an import;
 * packages are layered cocoa < swt < examples and may only reference lower layers (README
 * "Multiple Go packages", "Round 10 controlexample"). */
final class PackageQualifier {

	private final Emitter emitter;

	PackageQualifier(Emitter emitter) {
		this.emitter = emitter;
	}

	// The 2 lowercase Java class names in this codebase (id, objc_super) need more than a bare
	// first-letter capitalize to reach their hand-written exported alias (id_manual.go's
	// Id/ObjcSuper) - "objc_super" capitalized is "Objc_super", not "ObjcSuper".
	private static final Map<String, String> LOWERCASE_ALIASES = Map.of("id", "Id", "objc_super", "ObjcSuper");

	/** Qualifies an already-built bare identifier, e.g. "New" + ci.goFuncPrefix. Capitalized when
	 * actually crossing packages: Go visibility is the identifier's own case. */
	String qualify(String bareIdent, TypeModel.ClassInfo ci) {
		String prefix = packagePrefix(ci.goPackage);
		if (prefix.isEmpty()) return bareIdent;
		String alias = LOWERCASE_ALIASES.get(bareIdent);
		return prefix + (alias != null ? alias : Names.capitalize(bareIdent));
	}

	private String packagePrefix(String targetGoPackage) {
		if (targetGoPackage.equals(emitter.currentGoPackage)) return "";
		if (GoTypes.layer(targetGoPackage) > GoTypes.layer(emitter.currentGoPackage)) {
			System.err.println("j2go: guard violated: " + emitter.currentGoPackage + " file " + emitter.currentJavaPackage
					+ " must not reference a " + targetGoPackage + " type");
			System.exit(1);
		}
		emitter.fileImports.add(GoTypes.importPath(targetGoPackage));
		return targetGoPackage + ".";
	}

	/** Same guard, for a type GoTypes.map could not resolve at all (not in model, not manual). */
	void checkNoForeignPackageLeak(String qualifiedJavaTypeName) {
		if (!emitter.currentGoPackage.equals("cocoa")) return;
		int dot = qualifiedJavaTypeName.lastIndexOf('.');
		String javaPackage = dot < 0 ? "" : qualifiedJavaTypeName.substring(0, dot);
		if (javaPackage.startsWith("org.eclipse.swt") && !GoTypes.isCocoaPackage(javaPackage)) {
			System.err.println("j2go: guard violated: cocoa file " + emitter.currentJavaPackage
					+ " must not reference swt type " + qualifiedJavaTypeName);
			System.exit(1);
		}
	}
}
