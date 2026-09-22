package dev.gowt.j2go;

import java.util.Set;

/**
 * Names for out-of-translated-set members that are hand-written in swt/*_manual.go
 * (see tooling/j2go/manual.txt). Must match the Go identifiers used there exactly.
 */
class Manual {

	private static final String SWT = "org.eclipse.swt.SWT";
	private static final String MONITOR = "org.eclipse.swt.widgets.Monitor";
	private static final String ROUNDING_MODE = "org.eclipse.swt.graphics.RoundingMode";

	private static final Set<String> MANUAL_TYPES = Set.of(SWT, MONITOR, ROUNDING_MODE);

	static boolean isManual(String qualifiedTypeName) {
		return MANUAL_TYPES.contains(qualifiedTypeName);
	}

	/** RoundingMode is translated as a Go value type (enum-like); Monitor keeps class/pointer semantics. */
	static boolean isValueType(String qualifiedTypeName) {
		return qualifiedTypeName.equals(ROUNDING_MODE);
	}

	static String goTypeName(String qualifiedTypeName) {
		return switch (qualifiedTypeName) {
			case MONITOR -> "Monitor";
			case ROUNDING_MODE -> "RoundingMode";
			default -> "unsupported_manual_type_" + qualifiedTypeName;
		};
	}

	/** Static field/method reference on a manual type: SWT has no prefix, others use Class+Name. */
	static String staticMember(String qualifiedTypeName, String javaMemberName) {
		if (qualifiedTypeName.equals(SWT)) return Names.capitalize(javaMemberName);
		return goTypeName(qualifiedTypeName) + Names.capitalize(javaMemberName);
	}

	/** Instance method on a manual type value, e.g. RoundingMode.round(float). */
	static String instanceMember(String javaMemberName) {
		return Names.capitalize(javaMemberName);
	}
}
