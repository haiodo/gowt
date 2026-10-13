package dev.gowt.j2go;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeSet;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Stream;

/** Rewrites records into final classes before parsing: the emitter has no record rule. */
final class SourcePrep {

	private static final Pattern RECORD = Pattern.compile("(?m)^([ \\t]*)((?:(?:public|private|protected|static|final)\\s+)*)record\\s+(\\w+)\\s*(<[^>(]*>)?\\s*\\(");

	private SourcePrep() {}

	/** Copies the .java files of root that contain a record, rewritten, into mirror; true if any. */
	static boolean mirror(Path root, Path mirror, Map<String, String> dollars) throws IOException {
		boolean any = false;
		try (Stream<Path> files = Files.walk(root)) {
			for (Path f : (Iterable<Path>) files.filter(p -> p.toString().endsWith(".java"))::iterator) {
				any |= mirrorFile(f, mirror.resolve(root.relativize(f)), dollars);
			}
		}
		return any;
	}

	/** dollars null = an SWT file (rewritten as before); else a non-SWT file: '$' identifiers renamed per the map, records get value methods. */
	static boolean mirrorFile(Path f, Path dst, Map<String, String> dollars) throws IOException {
		String text = Files.readString(f, StandardCharsets.UTF_8);
		String out = prepare(text, dollars, true);
		if (out.equals(text)) return false;
		Files.createDirectories(dst.getParent());
		Files.writeString(dst, out, StandardCharsets.UTF_8);
		return true;
	}

	private static String prepare(String text, Map<String, String> dollars, boolean hoist) {
		if (!text.contains("record ") && !text.contains("class ") && (dollars == null || !text.contains("$"))) return text;
		String src = dollars == null ? text : mapIdents(text, id -> dollars.getOrDefault(id, id));
		src = desugar(anonymizeLocalClasses(src), dollars != null);
		return hoist ? hoistLocalClasses(src) : src;
	}

	/** Rewrites a mirror file without hoisting; true if that changed it. */
	static boolean unhoist(Path mirrorFile, Path realFile, Map<String, String> dollars) throws IOException {
		String plain = prepare(Files.readString(realFile, StandardCharsets.UTF_8), dollars, false);
		if (plain.equals(Files.readString(mirrorFile, StandardCharsets.UTF_8))) return false;
		Files.writeString(mirrorFile, plain, StandardCharsets.UTF_8);
		return true;
	}

	/**
	 * Go has no '$' in identifiers. Every Java identifier with one gets a Go-safe name before parsing, one
	 * name per identifier across all files: a leading '$' becomes "d_" ("D_" before an uppercase letter), so the
	 * first letter, not the '$', decides exported-ness as for any name; other '$' become '_'. A name that
	 * already occurs in the sources, or was handed out, gets '_' appended. Sorted, so the result is deterministic.
	 */
	static Map<String, String> dollarMap(Collection<Path> files) throws IOException {
		Set<String> all = new HashSet<>();
		for (Path f : files) mapIdents(Files.readString(f, StandardCharsets.UTF_8), id -> { all.add(id); return id; });
		Map<String, String> map = new HashMap<>();
		for (String id : new TreeSet<>(all)) {
			if (id.indexOf('$') < 0) continue;
			String name = id.replace('$', '_');
			if (id.charAt(0) == '$') {
				boolean upper = id.length() > 1 && Character.isUpperCase(id.charAt(1));
				name = (upper ? "D_" : "d_") + id.substring(1).replace('$', '_');
			}
			while (all.contains(name)) name += "_";
			all.add(name);
			map.put(id, name);
		}
		return map;
	}

	/** Applies f to every identifier token of Java source, outside comments, string, text block and char literals. */
	static String mapIdents(String src, java.util.function.UnaryOperator<String> f) {
		StringBuilder b = new StringBuilder(src.length());
		int i = 0, n = src.length();
		while (i < n) {
			char c = src.charAt(i);
			int end = i + 1;
			if (src.startsWith("//", i)) {
				end = src.indexOf('\n', i);
			} else if (src.startsWith("/*", i)) {
				end = src.indexOf("*/", i + 2) + 2;
			} else if (src.startsWith("\"\"\"", i)) {
				end = src.indexOf("\"\"\"", i + 3) + 3;
			} else if (c == '"' || c == '\'') {
				while (end < n && src.charAt(end) != c) end += src.charAt(end) == '\\' ? 2 : 1;
				end++;
			} else if (Character.isDigit(c)) {
				while (end < n && (Character.isLetterOrDigit(src.charAt(end)) || src.charAt(end) == '_' || src.charAt(end) == '.')) end++;
			} else if (Character.isJavaIdentifierStart(c)) {
				while (end < n && Character.isJavaIdentifierPart(src.charAt(end))) end++;
				b.append(f.apply(src.substring(i, end)));
				i = end;
				continue;
			} else {
				b.append(c);
				i++;
				continue;
			}
			if (end <= i || end > n) end = n;
			b.append(src, i, end);
			i = end;
		}
		return b.toString();
	}

	static String desugar(String src, boolean valueMethods) {
		StringBuilder out = new StringBuilder();
		StringBuilder hoisted = new StringBuilder();
		int pos = 0;
		Matcher m = RECORD.matcher(src);
		while (m.find(pos)) {
			if (insideCommentOrString(src, m.start())) {
				out.append(src, pos, m.end());
				pos = m.end();
				continue;
			}
			int open = m.end() - 1;
			int close = matching(src, open, '(', ')');
			int bodyOpen = src.indexOf('{', close);
			String header = src.substring(close + 1, bodyOpen).trim();
			int bodyClose = matching(src, bodyOpen, '{', '}');
			String body = src.substring(bodyOpen + 1, bodyClose);
			List<String[]> comps = components(src.substring(open + 1, close));
			out.append(src, pos, m.start());
			StringBuilder rec = new StringBuilder();
			String mods = m.group(2).trim();
			boolean nested = depthAt(src, m.start()) > 0;
			// A record is implicitly static; a local one (inside a method body) cannot say so.
			boolean local = nested && !isMemberPosition(src, m.start());
			// A local record moves to the end of the file, as a static member: the emitter has no local classes.
			if (nested && !mods.contains("static")) mods = (mods + " static").trim();
			rec.append(m.group(1)).append(mods.isEmpty() ? "" : mods + " ").append("final class ").append(m.group(3))
					.append(m.group(4) == null ? "" : m.group(4)).append(header.isEmpty() ? "" : " " + header).append(" {\n");
			for (String[] c : comps) rec.append(m.group(1)).append("\tprivate final ").append(c[0]).append(' ').append(c[1]).append(";\n");
			String params = String.join(", ", comps.stream().map(c -> c[0] + " " + c[1]).toList());
			Matcher cm = Pattern.compile("(?m)^\\s*(?:public\\s+)?" + m.group(3) + "\\s*\\{").matcher(body);
			String ctorBody = "";
			if (cm.find()) {
				int cOpen = cm.end() - 1;
				int cClose = matching(body, cOpen, '{', '}');
				ctorBody = body.substring(cOpen + 1, cClose);
				body = body.substring(0, cm.start()) + body.substring(cClose + 1);
			}
			String access = mods.contains("public") ? "public " : mods.contains("protected") ? "protected " : mods.contains("private") ? "private " : "";
			rec.append(m.group(1)).append('\t').append(access).append(m.group(3)).append('(').append(params).append(") {\n").append(ctorBody);
			for (String[] c : comps) rec.append(m.group(1)).append("\t\tthis.").append(c[1]).append(" = ").append(c[1]).append(";\n");
			rec.append(m.group(1)).append("\t}\n");
			for (String[] c : comps) {
				if (Pattern.compile("\\b" + c[1] + "\\s*\\(\\s*\\)\\s*\\{").matcher(body).find()) continue;
				rec.append(m.group(1)).append("\tpublic ").append(c[0]).append(' ').append(c[1]).append("() { return ").append(c[1]).append("; }\n");
			}
			if (valueMethods) rec.append(valueMethods(m.group(1), m.group(3), m.group(4), comps, body));
			rec.append(body).append(m.group(1)).append("}");
			if (local) hoisted.append("\n\t").append(rec.toString().strip()).append("\n");
			else out.append(rec);
			pos = bodyClose + 1;
		}
		out.append(src.substring(pos));
		if (hoisted.length() > 0) out.insert(out.lastIndexOf("}"), hoisted);
		return out.toString();
	}

	/** equals, hashCode and toString of a record (JLS 8.10.3 / Record's contract), unless the body declares them. */
	private static String valueMethods(String ind, String name, String typeParams, List<String[]> comps, String body) {
		StringBuilder b = new StringBuilder();
		String self = typeParams == null ? name : name + "<?>";
		if (!Pattern.compile("\\bboolean\\s+equals\\s*\\(").matcher(body).find()) {
			b.append(ind).append("\t@Override\n").append(ind).append("\tpublic boolean equals(Object o) {\n");
			b.append(ind).append("\t\tif (this == o) return true;\n").append(ind).append("\t\tif (!(o instanceof ").append(self).append(")) return false;\n");
			b.append(ind).append("\t\t").append(self).append(" r = (").append(self).append(") o;\n").append(ind).append("\t\treturn true");
			for (String[] c : comps) {
				String x = c[1];
				b.append(switch (c[0]) {
					case "float" -> " && Float.compare(" + x + ", r." + x + ") == 0";
					case "double" -> " && Double.compare(" + x + ", r." + x + ") == 0";
					case "int", "long", "short", "byte", "char", "boolean" -> " && " + x + " == r." + x;
					default -> " && java.util.Objects.equals(" + x + ", r." + x + ")";
				});
			}
			b.append(";\n").append(ind).append("\t}\n");
		}
		if (!Pattern.compile("\\bint\\s+hashCode\\s*\\(").matcher(body).find()) {
			b.append(ind).append("\t@Override\n").append(ind).append("\tpublic int hashCode() {\n").append(ind).append("\t\tint h = 0;\n");
			for (String[] c : comps) {
				String x = c[1];
				b.append(ind).append("\t\th = 31 * h + ").append(switch (c[0]) {
					case "int", "short", "byte", "char" -> "(int) " + x;
					case "boolean" -> "(" + x + " ? 1231 : 1237)";
					case "long" -> "(int) (" + x + " ^ (" + x + " >>> 32))";
					case "float" -> "Float.hashCode(" + x + ")";
					case "double" -> "Double.hashCode(" + x + ")";
					default -> "(" + x + " == null ? 0 : " + x + ".hashCode())";
				}).append(";\n");
			}
			b.append(ind).append("\t\treturn h;\n").append(ind).append("\t}\n");
		}
		if (!Pattern.compile("\\bString\\s+toString\\s*\\(").matcher(body).find()) {
			b.append(ind).append("\t@Override\n").append(ind).append("\tpublic String toString() {\n").append(ind).append("\t\treturn \"").append(name).append("[\"");
			for (int k = 0; k < comps.size(); k++) b.append(" + \"").append(k == 0 ? "" : ", ").append(comps.get(k)[1]).append("=\" + ").append(comps.get(k)[1]);
			b.append(" + \"]\";\n").append(ind).append("\t}\n");
		}
		return b.toString();
	}

	private static final Pattern LOCAL_EXTENDING = Pattern.compile("(?m)^([ \\t]+)(?:final\\s+)?class\\s+(\\w+)\\s+extends\\s+([\\w.]+)\\s*\\{");

	/**
	 * A local class whose constructor does nothing but super(...) becomes an anonymous class at each
	 * `new Name(args)`: the emitter has anonymous classes with captured variables, but no local classes
	 * that capture. The constructor's parameters are replaced by the call's arguments in the super call.
	 * A class that is used any other way stays as it is.
	 */
	static String anonymizeLocalClasses(String src) {
		Matcher m = LOCAL_EXTENDING.matcher(src);
		for (int from = 0; m.find(from); ) {
			int start = m.start();
			if (insideCommentOrString(src, start) || depthAt(src, start) == 0 || isMemberPosition(src, start)) {
				from = m.end();
				continue;
			}
			String out = anonymizeOne(src, m);
			if (out == null) {
				from = m.end();
				continue;
			}
			src = out;
			m = LOCAL_EXTENDING.matcher(src);
			from = 0;
		}
		return src;
	}

	private static String anonymizeOne(String src, Matcher m) {
		String name = m.group(2), sup = m.group(3);
		int open = m.end() - 1, close = matching(src, open, '{', '}');
		String body = src.substring(open + 1, close);
		List<String> params = new ArrayList<>();
		String superArgs = "";
		Matcher cm = Pattern.compile("(?m)^\\s*" + name + "\\s*\\(([^)]*)\\)\\s*\\{").matcher(body);
		if (cm.find()) {
			int cOpen = cm.end() - 1, cClose = matching(body, cOpen, '{', '}');
			Matcher sm = Pattern.compile("(?s)^super\\s*\\((.*)\\)\\s*;$").matcher(body.substring(cOpen + 1, cClose).trim());
			if (!sm.matches()) return null;
			superArgs = sm.group(1);
			for (String p : splitArgs(cm.group(1))) {
				String[] words = p.trim().split("\\s+");
				params.add(words[words.length - 1]);
			}
			body = body.substring(0, cm.start()) + body.substring(cClose + 1);
		}
		int scopeEnd = blockEnd(src, close + 1);
		String scope = src.substring(close + 1, scopeEnd);
		Matcher um = Pattern.compile("\\bnew\\s+" + name + "\\s*\\(").matcher(scope);
		List<int[]> uses = new ArrayList<>();
		List<String> repl = new ArrayList<>();
		while (um.find()) {
			int argsOpen = um.end() - 1, argsClose = matching(scope, argsOpen, '(', ')');
			List<String> args = splitArgs(scope.substring(argsOpen + 1, argsClose));
			if (args.size() != params.size()) return null;
			String sa = superArgs;
			for (int i = 0; i < params.size(); i++) sa = sa.replaceAll("\\b" + params.get(i) + "\\b", Matcher.quoteReplacement("(" + args.get(i).trim() + ")"));
			uses.add(new int[] { um.start(), argsClose + 1 });
			repl.add("new " + sup + "(" + sa + ") {" + body + "}");
		}
		// Any other mention of the class (a variable of its type, instanceof) needs the class.
		int mentions = 0;
		for (Matcher w = Pattern.compile("\\b" + name + "\\b").matcher(scope); w.find(); ) mentions++;
		if (uses.isEmpty() || mentions != uses.size()) return null;
		StringBuilder out = new StringBuilder(scope);
		for (int i = uses.size() - 1; i >= 0; i--) out.replace(uses.get(i)[0], uses.get(i)[1], repl.get(i));
		return src.substring(0, m.start()) + out + src.substring(scopeEnd);
	}

	/** Index of the brace that closes the block a position is inside of. */
	private static int blockEnd(String s, int from) {
		int depth = 0;
		for (int i = from; i < s.length(); i++) {
			char ch = s.charAt(i);
			if (ch == '"' || ch == '\'') i = skipLiteral(s, i);
			else if (ch == '/' && s.charAt(i + 1) == '/') i = s.indexOf('\n', i);
			else if (ch == '/' && s.charAt(i + 1) == '*') i = s.indexOf("*/", i + 2) + 1;
			else if (ch == '{') depth++;
			else if (ch == '}' && depth-- == 0) return i;
		}
		throw new IllegalStateException("unbalanced {");
	}

	/** Splits a call's or declaration's argument list at its top-level commas. */
	private static List<String> splitArgs(String list) {
		List<String> r = new ArrayList<>();
		if (list.isBlank()) return r;
		int depth = 0, start = 0;
		for (int i = 0; i < list.length(); i++) {
			char c = list.charAt(i);
			if (c == '"' || c == '\'') i = skipLiteral(list, i);
			else if (c == '(' || c == '<' || c == '[' || c == '{') depth++;
			else if (c == ')' || c == '>' || c == ']' || c == '}') depth--;
			else if (c == ',' && depth == 0) {
				r.add(list.substring(start, i));
				start = i + 1;
			}
		}
		r.add(list.substring(start));
		return r;
	}

	private static final Pattern LOCAL_CLASS = Pattern.compile("(?m)^([ \\t]+)((?:final|abstract)\\s+)?class\\s+(\\w+)");

	/** A local class (declared in a method body) moves to the end of the file as a static member: the emitter has
	 * no local classes. Only for one that captures no local variable and no outer instance. */
	static String hoistLocalClasses(String src) {
		StringBuilder hoisted = new StringBuilder();
		java.util.Set<String> hoistedNames = new java.util.HashSet<>();
		Matcher m = LOCAL_CLASS.matcher(src);
		for (int from = 0; m.find(from); ) {
			if (insideCommentOrString(src, m.start()) || depthAt(src, m.start()) == 0 || isMemberPosition(src, m.start())) {
				from = m.end();
				continue;
			}
			int close = matching(src, src.indexOf('{', m.end()), '{', '}');
			String decl = src.substring(m.start() + m.group(1).length(), close + 1);
			int scopeEnd = blockEnd(src, close + 1);
			String scope = src.substring(close + 1, scopeEnd);
			// Two methods may each declare a local class of the same name; the file then holds one member of each.
			String name = m.group(3);
			if (!hoistedNames.add(name)) {
				String renamed = name + "_" + hoistedNames.size();
				decl = decl.replaceAll("\\b" + name + "\\b", renamed);
				scope = scope.replaceAll("\\b" + name + "\\b", renamed);
				hoistedNames.add(renamed);
			}
			hoisted.append("\n\tstatic ").append(decl).append("\n");
			src = src.substring(0, m.start()) + scope + src.substring(scopeEnd);
			m = LOCAL_CLASS.matcher(src);
			from = 0;
		}
		if (hoisted.length() > 0) src = src.substring(0, src.lastIndexOf('}')) + hoisted + "}\n";
		return src;
	}

	private static List<String[]> components(String list) {
		List<String[]> r = new ArrayList<>();
		int depth = 0;
		int start = 0;
		for (int i = 0; i <= list.length(); i++) {
			char c = i < list.length() ? list.charAt(i) : ',';
			if (c == '<') depth++;
			else if (c == '>') depth--;
			else if (c == ',' && depth == 0) {
				String part = list.substring(start, i).trim();
				start = i + 1;
				if (part.isEmpty()) continue;
				int sp = part.lastIndexOf(' ');
				r.add(new String[] { part.substring(0, sp).trim(), part.substring(sp + 1) });
			}
		}
		return r;
	}

	/** Index of the bracket closing the one at open, skipping strings, chars and comments. */
	private static int matching(String s, int open, char o, char c) {
		int depth = 0;
		for (int i = open; i < s.length(); i++) {
			char ch = s.charAt(i);
			if (ch == '"' || ch == '\'') {
				i = skipLiteral(s, i);
			} else if (ch == '/' && i + 1 < s.length() && s.charAt(i + 1) == '/') {
				i = s.indexOf('\n', i);
				if (i < 0) break;
			} else if (ch == '/' && i + 1 < s.length() && s.charAt(i + 1) == '*') {
				i = s.indexOf("*/", i + 2) + 1;
			} else if (ch == o) {
				depth++;
			} else if (ch == c && --depth == 0) {
				return i;
			}
		}
		throw new IllegalStateException("unbalanced " + o);
	}

	private static int skipLiteral(String s, int i) {
		char q = s.charAt(i);
		for (int j = i + 1; j < s.length(); j++) {
			if (s.charAt(j) == '\\') j++;
			else if (s.charAt(j) == q) return j;
		}
		return s.length();
	}

	private static int depthAt(String s, int end) {
		int depth = 0;
		for (int i = 0; i < end; i++) {
			char ch = s.charAt(i);
			if (ch == '"' || ch == '\'') i = skipLiteral(s, i);
			else if (ch == '/' && s.charAt(i + 1) == '/') i = s.indexOf('\n', i);
			else if (ch == '/' && s.charAt(i + 1) == '*') i = s.indexOf("*/", i + 2) + 1;
			else if (ch == '{') depth++;
			else if (ch == '}') depth--;
		}
		return depth;
	}

	/** A record is a class member unless its enclosing block belongs to a method (a parenthesis in the header). */
	private static boolean isMemberPosition(String s, int pos) {
		int depth = 0;
		for (int i = pos - 1; i >= 0; i--) {
			char ch = s.charAt(i);
			if (ch == '}') depth++;
			else if (ch == '{' && depth-- == 0) {
				int from = Math.max(s.lastIndexOf(';', i), Math.max(s.lastIndexOf('}', i - 1), s.lastIndexOf('{', i - 1)));
				String head = s.substring(from + 1, i).replaceAll("(?s)/\\*.*?\\*/", "").replaceAll("//[^\\n]*", "");
				return head.matches("(?s).*\\b(class|interface|enum|record)\\s.*");
			}
		}
		return true;
	}

	private static boolean insideCommentOrString(String s, int pos) {
		int ls = s.lastIndexOf('\n', pos - 1) + 1;
		String line = s.substring(ls, pos);
		if (line.contains("//")) return true;
		String t = line.trim();
		return t.startsWith("*") || t.startsWith("/*");
	}
}
