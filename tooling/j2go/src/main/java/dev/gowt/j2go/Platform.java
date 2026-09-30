package dev.gowt.j2go;

import java.util.ArrayList;
import java.util.List;

/** An SWT platform slot: source roots, the GOOS file suffix of its output, its PI package dir. */
public enum Platform {
	COCOA("cocoa", "darwin", "internal/cocoa"),
	WIN32("win32", "windows", "internal/win32"),
	GTK("gtk", "linux", "internal/gtk");

	private static final String B = "bundles/org.eclipse.swt/";

	public final String swtName, goos, piDir;

	Platform(String swtName, String goos, String piDir) {
		this.swtName = swtName;
		this.goos = goos;
		this.piDir = piDir;
	}

	/** gtk is a slot only (TSK-043). */
	public boolean implemented() {
		return this != GTK;
	}

	/** Go package name of this platform's PI bindings (internal/<swtName>). */
	public String piPackage() {
		return swtName;
	}

	public static Platform parse(String name) {
		for (Platform p : values()) if (p.swtName.equals(name) || p.goos.equals(name)) return p;
		throw new IllegalArgumentException("unknown platform " + name + " (cocoa, win32, gtk)");
	}

	/** Roots every platform shares: their output carries no GOOS suffix. */
	public static List<String> commonRoots() {
		return List.of(
				B + "Eclipse SWT/common",
				B + "Eclipse SWT PI/common",
				// org.eclipse.swt.accessibility.Accessible/ACC: not translated (manual.txt), but
				// Control.java declares fields of these types - JDT still needs to resolve them.
				B + "Eclipse SWT Accessibility/common",
				B + "Eclipse SWT Printing/common",
				// org.eclipse.swt.custom: StackLayout/SashForm/SashFormLayout/SashFormData (Round 8).
				B + "Eclipse SWT Custom Widgets/common",
				// org.eclipse.swt.examples.* (Round 10): each example package is its own Go package.
				"examples/org.eclipse.swt.examples/src",
				// Round 12: the SWT JUnit tests (org.eclipse.swt.tests.junit -> tests/swttests).
				"tests/org.eclipse.swt.tests/JUnit Tests");
	}

	/** Roots of this platform; its emulated dirs count here too, each platform picks its own set. */
	public List<String> roots() {
		List<String> r = new ArrayList<>();
		for (String bundle : new String[] { "Eclipse SWT", "Eclipse SWT PI", "Eclipse SWT Accessibility", "Eclipse SWT Printing" })
			r.add(B + bundle + "/" + swtName);
		if (this == COCOA) {
			// BidiUtil: not on win32's real sourcepath for cocoa, but the real cocoa build fragment
			// (binaries/org.eclipse.swt.cocoa.macosx.*/build.properties) pulls this one in too.
			r.add(B + "Eclipse SWT/emulated/bidi");
			// Cocoa has no ToolTip/CoolBar/ExpandBar of its own: the emulated ones (Round 17).
			r.add(B + "Eclipse SWT/emulated/tooltip");
			r.add(B + "Eclipse SWT/emulated/coolbar");
			r.add(B + "Eclipse SWT/emulated/expand");
		}
		return r;
	}
}
