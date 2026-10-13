package dev.gowt.j2go;

import org.eclipse.jdt.core.dom.*;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

/**
 * Java package -> Go package layout, import paths and file-header settings of a project, read from
 * packages.txt in the config dir. Without the file the SWT rules built into GoTypes apply.
 * Lines (# starts a comment):
 *   module &lt;go module path&gt;
 *   class &lt;java class, fully qualified&gt; &lt;dir&gt; [layer=N]
 *   map &lt;java package&gt; &lt;dir&gt; [layer=N]     ("pkg.*" = pkg and its subpackages, {sub} = the rest of the package as a path, {flat} = the same with _; Go package names are the last dir segment and must stay unique)
 *   default &lt;dir&gt; [layer=N]                    (no rule matched; else the Java package as a path)
 *   commit &lt;path substring or *&gt; &lt;sha&gt;        (header "@ sha"; none = no revision in the header)
 *   merge &lt;name&gt; &lt;dir&gt;...                      (these dirs are one Go package; cycles are merged automatically, see collapse)
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
	final List<List<String>> merges = new ArrayList<>();
	/** dir after rules -> final dir, set by collapse. */
	final Map<String, String> remap = new HashMap<>();
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
				case "merge" -> p.merges.add(new ArrayList<>(Arrays.asList(w).subList(1, w.length)));
				case "message-prefix" -> p.prefix = w[1];
				default -> throw new IOException(file + ": unknown directive: " + line);
			}
		}
		if (p.module == null) throw new IOException(file + ": no module line");
		active = p;
	}

	public static boolean isActive() {
		return active != null;
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
				String rest = javaPackage.length() > head.length() ? javaPackage.substring(head.length() + 1) : "";
				return record(r, r.dir.replace("{sub}", rest.replace('.', '/')).replace("{flat}", rest.replace('.', '_')).replaceAll("/+$", ""));
			}
			if (pat.equals(javaPackage)) return record(r, r.dir);
		}
		if (fallback != null) return record(fallback, fallback.dir);
		return javaPackage.replace('.', '/');
	}

	private String record(Rule r, String dir) {
		// "pkg.*" matched pkg itself: nothing left for {sub}, take pkg's own last segment.
		if (dir.isEmpty()) dir = r.pattern.substring(0, r.pattern.length() - 2).replaceAll(".*\\.", "");
		dir = remap.getOrDefault(dir, dir);
		String name = dir.substring(dir.lastIndexOf('/') + 1);
		dirByPackage.put(name, dir);
		layerByPackage.put(name, r.layer);
		return dir;
	}

	String revision(String relPath) {
		for (String[] c : commits) if (c[0].equals("*") || relPath.contains(c[0])) return c[1];
		return null;
	}

	/**
	 * Java packages are routinely cyclic, Go packages may not be: Go packages whose types reference each
	 * other in a cycle become one package (named after the last segment of the common Java package prefix,
	 * or "core"; a "merge" line names it). Two dirs ending in the same name get distinct names (parent
	 * segment prefixed, else _2). Edges come from the type bindings of the emitted files, so a reference
	 * the emitter spells without a binding is not seen.
	 */
	static void collapse(List<CompilationUnit> units, int primary) {
		Profile pr = active;
		Map<String, String> dirOfType = new TreeMap<>();
		Map<String, Set<String>> javaPackages = new TreeMap<>();
		for (CompilationUnit cu : units) {
			if (cu.getPackage() == null || cu.types().isEmpty()) continue;
			String jp = cu.getPackage().getName().getFullyQualifiedName();
			String top = ((AbstractTypeDeclaration) cu.types().get(0)).getName().getIdentifier();
			String dir = GoTypes.goPackageDir(jp, top);
			dirOfType.put(jp + "." + top, dir);
			javaPackages.computeIfAbsent(dir, k -> new TreeSet<>()).add(jp);
		}
		Map<String, Set<String>> edges = new TreeMap<>();
		for (String d : javaPackages.keySet()) edges.put(d, new TreeSet<>());
		for (int i = 0; i < primary && i < units.size(); i++) {
			CompilationUnit cu = units.get(i);
			if (cu.getPackage() == null || cu.types().isEmpty()) continue;
			String from = dirOfType.get(cu.getPackage().getName().getFullyQualifiedName() + "." + ((AbstractTypeDeclaration) cu.types().get(0)).getName().getIdentifier());
			cu.accept(new ASTVisitor() {
				void ref(ITypeBinding t) {
					for (int guard = 0; t != null && guard < 8; guard++) {
						if (t.isArray()) t = t.getElementType();
						else if (t.isParameterizedType() || t.isRawType() || t.isGenericType()) { for (ITypeBinding a : t.getTypeArguments()) ref(a); t = t.getErasure(); if (guard > 6) return; }
						else break;
					}
					if (t == null || t.isPrimitive() || t.isTypeVariable() || t.isCapture() || t.isWildcardType()) return;
					t = t.getTypeDeclaration();
					while (t.getDeclaringClass() != null) t = t.getDeclaringClass();
					String to = dirOfType.get(t.getQualifiedName());
					if (to != null && !to.equals(from)) edges.get(from).add(to);
				}
				@Override public boolean visit(SimpleName n) {
					IBinding b = n.resolveBinding();
					if (b instanceof ITypeBinding t) ref(t);
					else if (b instanceof IMethodBinding m) {
						ref(m.getDeclaringClass()); ref(m.getReturnType());
						for (ITypeBinding a : m.getParameterTypes()) ref(a);
					} else if (b instanceof IVariableBinding v) { if (v.getDeclaringClass() != null) ref(v.getDeclaringClass()); ref(v.getType()); }
					return true;
				}
				@Override public boolean visit(TypeDeclaration n) {
					ITypeBinding t = n.resolveBinding();
					for (; t != null; t = t.getSuperclass()) {
						ref(t);
						for (ITypeBinding i : t.getInterfaces()) ref(i);
					}
					return true;
				}
			});
		}
		// Forced groups first: edges both ways make them one component.
		Map<String, String> forcedName = new TreeMap<>();
		for (List<String> m : pr.merges) {
			String name = m.get(0);
			for (String d : m.subList(1, m.size())) {
				forcedName.put(d, name);
				for (String e : m.subList(1, m.size())) if (edges.containsKey(d) && edges.containsKey(e) && !d.equals(e)) edges.get(d).add(e);
			}
		}
		Map<String, String> merged = new TreeMap<>();
		Set<String> usedNames = new TreeSet<>();
		for (List<String> comp : new Tarjan(edges).components()) {
			if (comp.size() < 2) continue;
			Collections.sort(comp);
			String name = null;
			for (String d : comp) if (forcedName.containsKey(d)) name = forcedName.get(d);
			if (name == null) {
				List<String> common = null;
				for (String d : comp) for (String jp : javaPackages.get(d)) {
					List<String> segs = Arrays.asList(jp.split("\\."));
					if (common == null) common = new ArrayList<>(segs);
					else { int k = 0; while (k < common.size() && k < segs.size() && common.get(k).equals(segs.get(k))) k++; common = new ArrayList<>(common.subList(0, k)); }
				}
				name = common == null || common.isEmpty() ? "core" : common.get(common.size() - 1);
			}
			// Two components never share a package: the union could close a new cycle.
			String unique = name;
			for (int n = 2; !usedNames.add(unique); n++) unique = name + "_" + n;
			name = unique;
			for (String d : comp) merged.put(d, name);
			System.err.println("j2go: merged Go packages (import cycle) " + comp + " -> " + name);
		}
		// Distinct package names among the final dirs.
		Set<String> finalDirs = new TreeSet<>();
		for (String d : javaPackages.keySet()) finalDirs.add(merged.getOrDefault(d, d));
		Map<String, String> renamed = new TreeMap<>();
		Set<String> taken = new TreeSet<>();
		for (String d : finalDirs) taken.add(d.substring(d.lastIndexOf('/') + 1));
		Set<String> seen = new TreeSet<>();
		for (String d : finalDirs) {
			int slash = d.lastIndexOf('/');
			String last = d.substring(slash + 1);
			if (seen.add(last) || GoTypes.isPiGoPackage(last)) continue;
			String parent = slash < 0 ? "" : d.substring(0, slash);
			String base = parent.isEmpty() ? last : parent.substring(parent.lastIndexOf('/') + 1) + "_" + last;
			String cand = base;
			for (int n = 2; !taken.add(cand); n++) cand = base + "_" + n;
			String nd = parent.isEmpty() ? cand : parent + "/" + cand;
			renamed.put(d, nd);
			System.err.println("j2go: Go package name " + last + " is taken, " + d + " -> " + nd);
		}
		for (String d : javaPackages.keySet()) {
			String f = merged.getOrDefault(d, d);
			f = renamed.getOrDefault(f, f);
			if (!f.equals(d)) pr.remap.put(d, f);
		}
		pr.dirByPackage.clear();
		pr.layerByPackage.clear();
	}

	private static final class Tarjan {
		final Map<String, Set<String>> g;
		final Map<String, Integer> idx = new HashMap<>(), low = new HashMap<>();
		final Deque<String> stack = new ArrayDeque<>();
		final Set<String> on = new HashSet<>();
		final List<List<String>> out = new ArrayList<>();

		Tarjan(Map<String, Set<String>> g) { this.g = g; }

		List<List<String>> components() {
			for (String v : g.keySet()) if (!idx.containsKey(v)) visit(v);
			return out;
		}

		void visit(String v) {
			idx.put(v, idx.size()); low.put(v, idx.get(v)); stack.push(v); on.add(v);
			for (String w : g.getOrDefault(v, Set.of())) {
				if (!idx.containsKey(w)) { visit(w); low.put(v, Math.min(low.get(v), low.get(w))); }
				else if (on.contains(w)) low.put(v, Math.min(low.get(v), idx.get(w)));
			}
			if (low.get(v).equals(idx.get(v))) {
				List<String> c = new ArrayList<>();
				String w;
				do { w = stack.pop(); on.remove(w); c.add(w); } while (!w.equals(v));
				out.add(c);
			}
		}
	}
}
