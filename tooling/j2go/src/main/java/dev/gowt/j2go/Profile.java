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
	record Merge(String line, String name, List<String> dirs) {}

	/** null = built-in SWT rules. */
	static Profile active;

	String module;
	String prefix;
	boolean layered;
	final List<Rule> classes = new ArrayList<>(), maps = new ArrayList<>();
	Rule fallback;
	final List<Merge> merges = new ArrayList<>();
	/** dir after rules -> final dir, set by collapse. */
	final Map<String, String> remap = new HashMap<>();
	final List<String[]> commits = new ArrayList<>();
	final Map<String, String> dirByPackage = new HashMap<>();
	final Map<String, Integer> layerByPackage = new HashMap<>();

	static void load(Path file) throws IOException {
		active = null;
		if (!Files.exists(file)) return;
		Profile p = new Profile();
		List<String> lines = Files.readAllLines(file);
		for (int n = 1; n <= lines.size(); n++) {
			String line = lines.get(n - 1);
			int hash = line.indexOf('#');
			if (hash >= 0) line = line.substring(0, hash);
			if (line.isBlank()) continue;
			String where = file + ":" + n + ": ";
			List<String> w = new ArrayList<>();
			int layer = 0;
			for (String t : line.trim().split("\\s+")) {
				if (!t.startsWith("layer=")) { w.add(t); continue; }
				try { layer = Integer.parseInt(t.substring(6)); } catch (NumberFormatException e) { throw new IOException(where + "bad layer: " + t); }
				p.layered = true;
			}
			p.directive(w, layer, where, line.trim());
		}
		if (p.module == null) throw new IOException(file + ": no module line");
		active = p;
	}

	private void directive(List<String> w, int layer, String where, String line) throws IOException {
		int min = switch (w.get(0)) {
			case "module", "default", "message-prefix" -> 2;
			case "class", "map", "commit" -> 3;
			case "merge" -> 4;
			default -> throw new IOException(where + "unknown directive: " + line);
		};
		int max = w.get(0).equals("merge") ? Integer.MAX_VALUE : min;
		if (w.size() < min || w.size() > max) throw new IOException(where + "expected " + (min - 1) + (max == min ? "" : " or more") + " arguments: " + line);
		boolean usesLayer = w.get(0).equals("class") || w.get(0).equals("map") || w.get(0).equals("default");
		if (layer != 0 && !usesLayer) throw new IOException(where + "layer= only applies to class, map and default: " + line);
		switch (w.get(0)) {
			case "module" -> module = w.get(1);
			case "class" -> classes.add(new Rule(w.get(1), w.get(2), layer));
			case "map" -> {
				boolean placeholder = w.get(2).contains("{sub}") || w.get(2).contains("{flat}");
				if (placeholder && !w.get(1).endsWith(".*")) throw new IOException(where + "{sub} and {flat} need a pattern ending in .*: " + line);
				maps.add(new Rule(w.get(1), w.get(2), layer));
			}
			case "default" -> fallback = new Rule("", w.get(1), layer);
			case "commit" -> commits.add(new String[] {w.get(1), w.get(2)});
			case "merge" -> merges.add(new Merge(where + line, w.get(1), w.subList(2, w.size())));
			default -> prefix = w.get(1);
		}
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
	 * or "core"; a "merge" line names it). Two dirs ending in the same name, or in a name the generated
	 * code uses itself, get distinct names (parent segment prefixed, else _2). Edges come from the type
	 * bindings of the emitted files, so a reference the emitter spells without a binding is not seen.
	 */
	static void collapse(List<CompilationUnit> units, int primary) throws IOException {
		Profile pr = active;
		Map<String, String> dirOfType = new TreeMap<>();
		Map<String, Set<String>> javaPackages = new TreeMap<>();
		for (CompilationUnit cu : units) {
			String jp = cu.getPackage() == null ? null : cu.getPackage().getName().getFullyQualifiedName();
			for (Object o : jp == null ? List.of() : cu.types()) {
				String top = ((AbstractTypeDeclaration) o).getName().getIdentifier();
				String dir = GoTypes.goPackageDir(jp, top);
				dirOfType.put(jp + "." + top, dir);
				javaPackages.computeIfAbsent(dir, k -> new TreeSet<>()).add(jp);
			}
		}
		Map<String, Set<String>> edges = new TreeMap<>();
		for (String d : javaPackages.keySet()) edges.put(d, new TreeSet<>());
		for (int i = 0; i < primary && i < units.size(); i++) {
			CompilationUnit cu = units.get(i);
			if (cu.getPackage() == null) continue;
			String jp = cu.getPackage().getName().getFullyQualifiedName();
			for (Object o : cu.types()) {
				AbstractTypeDeclaration td = (AbstractTypeDeclaration) o;
				td.accept(new EdgeVisitor(dirOfType.get(jp + "." + td.getName().getIdentifier()), dirOfType, edges));
			}
		}
		Map<String, String> forcedName = pr.forceMerges(javaPackages.keySet(), edges);
		Map<String, String> merged = mergeCycles(edges, javaPackages, forcedName);
		Map<String, String> renamed = renameClashes(merged, javaPackages.keySet());
		for (String d : javaPackages.keySet()) {
			String f = merged.getOrDefault(d, d);
			f = renamed.getOrDefault(f, f);
			if (!f.equals(d)) pr.remap.put(d, f);
		}
		pr.dirByPackage.clear();
		pr.layerByPackage.clear();
	}

	/** Adds the edges of the "merge" lines; returns dir -> forced name. */
	private Map<String, String> forceMerges(Set<String> dirs, Map<String, Set<String>> edges) throws IOException {
		Map<String, String> forcedName = new TreeMap<>();
		for (Merge m : merges) {
			for (String d : m.dirs) {
				if (!dirs.contains(d)) throw new IOException("merge: no such Go dir " + d + ": " + m.line);
				String old = forcedName.put(d, m.name);
				if (old != null && !old.equals(m.name)) throw new IOException("merge: " + d + " is named both " + old + " and " + m.name + ": " + m.line);
				edges.get(d).addAll(m.dirs);
				edges.get(d).remove(d);
			}
		}
		return forcedName;
	}

	private static Map<String, String> mergeCycles(Map<String, Set<String>> edges, Map<String, Set<String>> javaPackages, Map<String, String> forcedName) throws IOException {
		Map<String, String> merged = new TreeMap<>();
		List<List<String>> comps = new ArrayList<>();
		Set<String> inComp = new TreeSet<>();
		for (List<String> comp : new Tarjan(edges).components()) {
			if (comp.size() < 2) continue;
			Collections.sort(comp);
			comps.add(comp);
			inComp.addAll(comp);
		}
		comps.sort(Comparator.comparing(c -> c.get(0)));
		// A merged package never takes the name of an unmerged one or of another merged one: the union could close a new cycle.
		Set<String> usedNames = new TreeSet<>();
		for (String d : edges.keySet()) if (!inComp.contains(d)) { usedNames.add(d); usedNames.add(d.substring(d.lastIndexOf('/') + 1)); }
		for (List<String> comp : comps) {
			Set<String> forced = new TreeSet<>();
			for (String d : comp) if (forcedName.containsKey(d)) forced.add(forcedName.get(d));
			if (forced.size() > 1) throw new IOException("merge: Go packages " + comp + " form one cycle but are named " + forced);
			String name = forced.isEmpty() ? commonName(comp, javaPackages) : forced.iterator().next();
			String unique = name;
			for (int n = 2; !usedNames.add(unique); n++) unique = name + "_" + n;
			for (String d : comp) merged.put(d, unique);
			System.err.println("j2go: merged Go packages (import cycle) " + comp + " -> " + unique);
		}
		return merged;
	}

	private static String commonName(List<String> comp, Map<String, Set<String>> javaPackages) {
		List<String> common = null;
		for (String d : comp) for (String jp : javaPackages.get(d)) {
			List<String> segs = Arrays.asList(jp.split("\\."));
			if (common == null) { common = new ArrayList<>(segs); continue; }
			int k = 0;
			while (k < common.size() && k < segs.size() && common.get(k).equals(segs.get(k))) k++;
			common = new ArrayList<>(common.subList(0, k));
		}
		return common == null || common.isEmpty() ? "core" : common.get(common.size() - 1);
	}

	/** Names the generated code or Go itself already uses: a package of that name would shadow them. */
	private static final Set<String> RESERVED = Set.of(
			"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto",
			"if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var",
			"any", "bool", "byte", "cap", "complex", "error", "false", "float32", "float64", "int", "int8", "int16", "int32", "int64",
			"len", "make", "new", "nil", "panic", "print", "recover", "rune", "string", "true", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"main", "init", "jrt", "junit", "errors", "fmt", "io", "math", "bits", "os", "reflect", "runtime", "slices", "strconv", "strings",
			"sync", "syscall", "time", "unicode", "unsafe");

	private static boolean reserved(String name) {
		return RESERVED.contains(name) || name.equals(lastSegment(Config.runtimeImport)) || name.equals(lastSegment(Config.junitImport));
	}

	private static String lastSegment(String path) {
		return path.substring(path.lastIndexOf('/') + 1);
	}

	/** merged dir -> new dir for every dir whose last segment is reserved or already taken. */
	private static Map<String, String> renameClashes(Map<String, String> merged, Set<String> dirs) {
		Set<String> finalDirs = new TreeSet<>();
		for (String d : dirs) finalDirs.add(merged.getOrDefault(d, d));
		Set<String> taken = new TreeSet<>();
		for (String d : finalDirs) taken.add(lastSegment(d));
		Map<String, String> renamed = new TreeMap<>();
		Set<String> seen = new TreeSet<>();
		for (String d : finalDirs) {
			String last = lastSegment(d);
			boolean isReserved = reserved(last);
			if (GoTypes.isPiGoPackage(last) || seen.add(last) && !isReserved) continue;
			int slash = d.lastIndexOf('/');
			String parent = slash < 0 ? "" : d.substring(0, slash);
			String base = isReserved ? last + "_" : parent.isEmpty() ? last : lastSegment(parent) + "_" + last;
			String cand = base;
			for (int n = 2; reserved(cand) || !taken.add(cand); n++) cand = base + "_" + n;
			String nd = parent.isEmpty() ? cand : parent + "/" + cand;
			renamed.put(d, nd);
			System.err.println("j2go: Go package name " + last + " is " + (isReserved ? "reserved" : "taken") + ", " + d + " -> " + nd);
		}
		return renamed;
	}

	/** Records an edge from one Go dir to every other dir whose types an AST subtree names, calls or inherits from. */
	private static final class EdgeVisitor extends ASTVisitor {
		final String from;
		final Map<String, String> dirOfType;
		final Map<String, Set<String>> edges;

		EdgeVisitor(String from, Map<String, String> dirOfType, Map<String, Set<String>> edges) {
			this.from = from;
			this.dirOfType = dirOfType;
			this.edges = edges;
		}

		void ref(ITypeBinding t) {
			if (t == null) return;
			if (t.isArray()) { ref(t.getElementType()); return; }
			if (t.isWildcardType()) { ref(t.getBound()); return; }
			for (ITypeBinding a : t.getTypeArguments()) ref(a);
			if (t.isPrimitive() || t.isTypeVariable() || t.isCapture()) return;
			t = t.getTypeDeclaration();
			while (t.getDeclaringClass() != null) t = t.getDeclaringClass();
			String to = dirOfType.get(t.getQualifiedName());
			if (to != null && !to.equals(from)) edges.get(from).add(to);
		}

		void hierarchy(ITypeBinding t) {
			if (t == null) return;
			ref(t);
			hierarchy(t.getSuperclass());
			for (ITypeBinding i : t.getInterfaces()) hierarchy(i);
		}

		@Override public boolean visit(SimpleName n) {
			IBinding b = n.resolveBinding();
			if (b instanceof ITypeBinding t) ref(t);
			if (b instanceof IMethodBinding m) {
				ref(m.getDeclaringClass());
				ref(m.getReturnType());
				for (ITypeBinding a : m.getParameterTypes()) ref(a);
			}
			if (b instanceof IVariableBinding v) {
				ref(v.getDeclaringClass());
				ref(v.getType());
			}
			return true;
		}

		@Override public boolean visit(TypeDeclaration n) { hierarchy(n.resolveBinding()); return true; }
		@Override public boolean visit(EnumDeclaration n) { hierarchy(n.resolveBinding()); return true; }
		@Override public boolean visit(RecordDeclaration n) { hierarchy(n.resolveBinding()); return true; }
		@Override public boolean visit(AnnotationTypeDeclaration n) { hierarchy(n.resolveBinding()); return true; }
		@Override public boolean visit(AnonymousClassDeclaration n) { hierarchy(n.resolveBinding()); return true; }
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
