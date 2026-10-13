package dev.gowt.j2go;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Java package -> Go package layout, import paths and file-header settings of a project, read from
 * packages.txt in the config dir. Without the file the SWT rules built into GoTypes apply.
 * Lines (# starts a comment):
 *   module &lt;go module path&gt;
 *   class &lt;java class, fully qualified&gt; &lt;dir&gt; [layer=N]
 *   map &lt;java package&gt; &lt;dir&gt; [layer=N]     ("pkg.*" = pkg and its subpackages, {sub} = the rest of the package as a path)
 *   default &lt;dir&gt; [layer=N]                    (no rule matched; else the Java package as a path)
 *   commit &lt;path substring or *&gt; &lt;sha&gt;        (header "@ sha"; none = no revision in the header)
 *   message-prefix &lt;word&gt;                       (stderr text of generated code; none = no prefix)
 * Rules are tried in file order. Layers are optional: without any, packages may reference each other freely.
 */
public final class Profile {
	record Rule(String pattern, String dir, int layer) {}

	/** null = built-in SWT rules. */
	static Profile active;

	String module;
	String prefix;
	boolean layered;
	final List<Rule> classes = new ArrayList<>(), maps = new ArrayList<>();
	Rule fallback;
	final List<String[]> commits = new ArrayList<>();
	final Map<String, String> dirByPackage = new HashMap<>();
	final Map<String, Integer> layerByPackage = new HashMap<>();

	static void load(Path file) throws IOException {
		active = null;
		if (!Files.exists(file)) return;
		Profile p = new Profile();
		for (String line : Files.readAllLines(file)) {
			int hash = line.indexOf('#');
			if (hash >= 0) line = line.substring(0, hash);
			String[] w = line.trim().split("\\s+");
			if (w[0].isEmpty()) continue;
			int layer = 0;
			for (String s : w) if (s.startsWith("layer=")) { layer = Integer.parseInt(s.substring(6)); p.layered = true; }
			switch (w[0]) {
				case "module" -> p.module = w[1];
				case "class" -> p.classes.add(new Rule(w[1], w[2], layer));
				case "map" -> p.maps.add(new Rule(w[1], w[2], layer));
				case "default" -> p.fallback = new Rule("", w[1], layer);
				case "commit" -> p.commits.add(new String[] {w[1], w[2]});
				case "message-prefix" -> p.prefix = w[1];
				default -> throw new IOException(file + ": unknown directive: " + line);
			}
		}
		if (p.module == null) throw new IOException(file + ": no module line");
		active = p;
	}

	/** Text before "deferred init" in the stderr messages of generated code. */
	public static String messagePrefix() {
		if (active == null) return "gowt: ";
		return active.prefix == null ? "" : active.prefix + ": ";
	}

	/** Go dir of a top-level Java class, or null when no rule applies. */
	String dirOf(String javaPackage, String topLevelName) {
		for (Rule r : classes) if (r.pattern.equals(javaPackage + "." + topLevelName)) return record(r, r.dir);
		return null;
	}

	String dirOfPackage(String javaPackage) {
		for (Rule r : maps) {
			String pat = r.pattern;
			if (pat.endsWith(".*")) {
				String head = pat.substring(0, pat.length() - 2);
				if (!javaPackage.equals(head) && !javaPackage.startsWith(head + ".")) continue;
				String sub = javaPackage.length() > head.length() ? javaPackage.substring(head.length() + 1).replace('.', '/') : "";
				return record(r, r.dir.replace("{sub}", sub).replaceAll("/+$", ""));
			}
			if (pat.equals(javaPackage)) return record(r, r.dir);
		}
		if (fallback != null) return record(fallback, fallback.dir);
		return javaPackage.replace('.', '/');
	}

	private String record(Rule r, String dir) {
		String name = dir.substring(dir.lastIndexOf('/') + 1);
		dirByPackage.put(name, dir);
		layerByPackage.put(name, r.layer);
		return dir;
	}

	String revision(String relPath) {
		for (String[] c : commits) if (c[0].equals("*") || relPath.contains(c[0])) return c[1];
		return null;
	}
}
