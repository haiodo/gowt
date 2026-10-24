package dev.gowt.j2go;

import dev.gowt.j2go.emit.Emitter;
import org.eclipse.jdt.core.JavaCore;
import org.eclipse.jdt.core.compiler.IProblem;
import org.eclipse.jdt.core.dom.*;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;

/** CLI: --swt <repo root> --out <dir> <java files relative to source roots or absolute...> */
public class Main {

	private static final String SWT_COMMIT = "af630a9093";
	// eclipse.platform.ui, the source of org.eclipse.jface.* (README "Round 22 jface").
	private static final String UI_COMMIT = "618d26a101";

	public static void main(String[] args) throws Exception {
		String swtRoot = null;
		String outDir = null;
		Platform platform = Platform.COCOA;
		String[] classpath = new String[0];
		boolean keepGoing = false;
		List<String> extraRoots = new ArrayList<>();
		List<Path> stubRoots = new ArrayList<>();
		List<String> files = new ArrayList<>();
		// Files after "--" are parsed and modeled (so names/bindings resolve exactly as they did
		// when translated) but not re-emitted - e.g. Widget.java needs OS.java's ClassInfo, not
		// a second copy of cocoa_os.go.
		List<String> refFiles = new ArrayList<>();
		List<String> target = files;
		for (int i = 0; i < args.length; i++) {
			switch (args[i]) {
				case "--swt" -> swtRoot = args[++i];
				case "--out" -> outDir = args[++i];
				case "--platform" -> platform = Platform.parse(args[++i]);
				case "--classpath" -> classpath = args[++i].split(":");
				case "--keep-going" -> keepGoing = true;
				case "--src" -> extraRoots.add(args[++i]);
				case "--config" -> Config.dir = Path.of(args[++i]);
				case "--stubs" -> stubRoots.add(Path.of(args[++i]));
				case "--runtime-import" -> Config.runtimeImport = args[++i];
				case "--junit-import" -> Config.junitImport = args[++i];
				case "--" -> target = refFiles;
				default -> target.add(args[i]);
			}
		}
		// Stubs default to <config>/stubs (Java stubs for test-harness classes outside the SWT repo, org.eclipse.test.Screenshots).
		Config.stubRoots = stubRoots.isEmpty() ? List.of(Config.dir.resolve("stubs")) : stubRoots;
		Profile.load(Config.dir.resolve("packages.txt"));
		if (swtRoot == null || outDir == null || files.isEmpty()) {
			System.err.println("usage: j2go --swt <swt repo root> --out <dir> [--platform cocoa|win32|gtk] [--src <extra source root>]... [--config <dir>] [--stubs <dir>]... [--runtime-import <path>] [--junit-import <path>] <java files...> [-- <reference-only java files...>]");
			System.exit(2);
		}

		if (!platform.implemented()) {
			System.err.println("j2go: platform " + platform.swtName + " is not implemented yet");
			System.exit(2);
		}

		GoTypes.platform = platform;
		GoTypes.piPackage = platform.swtName;
		Path swtRootPath = Path.of(swtRoot).toAbsolutePath().normalize();
		List<String> sourceRoots = new ArrayList<>();
		List<String> platformRoots = new ArrayList<>();
		List<String> allRoots = new ArrayList<>(platform.roots());
		allRoots.addAll(Platform.commonRoots(platform != Platform.GTK));
		// Records (win32) and local classes are rewritten in a mirror that shadows the real root for parsing.
		Path mirrorDir = Files.createTempDirectory("j2go-mirror");
		Map<String, String> mirrorToReal = new HashMap<>();
		for (String r : allRoots) {
			Path p = swtRootPath.resolve(r);
			if (!Files.isDirectory(p)) continue;
			boolean platformRoot = platform.roots().contains(r);
			if (mirrorDir != null) {
				Path mirror = mirrorDir.resolve(Integer.toString(sourceRoots.size()));
				if (SourcePrep.mirror(p, mirror, null)) {
					sourceRoots.add(mirror.toString());
					mirrorToReal.put(mirror.toString(), p.toString());
					if (platformRoot) platformRoots.add(mirror + "/");
				}
			}
			sourceRoots.add(p.toString());
			if (platformRoot) platformRoots.add(p + "/");
		}
		// A non-SWT --src root gets the same record/local class rewrite; a file named by absolute path below it is read from the mirror.
		Map<String, String> extraMirrors = new HashMap<>();
		Set<String> externalMirrorDirs = new HashSet<>();
		List<Path> externalSources = new ArrayList<>();
		for (String r : extraRoots) try (var walk = Files.walk(Path.of(r))) { walk.filter(q -> q.toString().endsWith(".java")).forEach(externalSources::add); }
		for (String f : concat(files, refFiles)) {
			Path q = Path.of(f);
			if (q.isAbsolute() && Files.exists(q) && !q.normalize().startsWith(swtRootPath)) externalSources.add(q);
		}
		Map<String, String> dollars = SourcePrep.dollarMap(externalSources);
		for (String r : extraRoots) {
			Path real = Path.of(r).toAbsolutePath().normalize();
			Path mirror = mirrorDir.resolve(Integer.toString(sourceRoots.size()));
			if (SourcePrep.mirror(real, mirror, dollars)) {
				externalMirrorDirs.add(mirror.toString());
				sourceRoots.add(mirror.toString());
				mirrorToReal.put(mirror.toString(), real.toString());
				extraMirrors.put(real.toString(), mirror.toString());
			}
			sourceRoots.add(real.toString());
		}
		for (Path d : Config.stubRoots) if (Files.isDirectory(d)) sourceRoots.add(d.toAbsolutePath().toString());
		if (platform == Platform.GTK) for (String d : new String[] { "gtkstubs", "gtkstubs-gen" })
			if (Files.isDirectory(Config.dir.resolve(d))) sourceRoots.add(Config.dir.resolve(d).toAbsolutePath().toString());

		List<String> absFiles = resolveSources(files, sourceRoots);
		List<String> absRefFiles = resolveSources(refFiles, sourceRoots);
		// A file named by absolute path is read from its mirror: below a --src root, or (outside every root) a one-file mirror of its own.
		int loneFiles = 0;
		for (List<String> list : List.of(absFiles, absRefFiles)) {
			for (int i = 0; i < list.size(); i++) {
				String f = list.get(i);
				if (sourceRoots.stream().noneMatch(r -> f.startsWith(r + "/"))) {
					Path dir = mirrorDir.resolve("file" + loneFiles++);
					Path real = Path.of(f);
					if (SourcePrep.mirrorFile(real, dir.resolve(real.getFileName()), dollars)) {
						externalMirrorDirs.add(dir.toString());
						mirrorToReal.put(dir.toString(), real.getParent().toString());
						list.set(i, dir.resolve(real.getFileName()).toString());
					}
					continue;
				}
				for (var e : extraMirrors.entrySet()) {
					if (f.startsWith(e.getKey() + "/") && Files.exists(Path.of(e.getValue() + f.substring(e.getKey().length())))) list.set(i, e.getValue() + f.substring(e.getKey().length()));
				}
			}
		}
		List<String> allAbsFiles = new ArrayList<>(absFiles);
		allAbsFiles.addAll(absRefFiles);

		Map<String, CompilationUnit> unitsByPath = new LinkedHashMap<>();
		String[] encodings = new String[allAbsFiles.size()];
		Arrays.fill(encodings, "UTF-8");
		List<CompilationUnit> orderedUnits = new ArrayList<>();
		int problemCount = 0;
		Set<String> badFiles = new HashSet<>();
		// A hoisted local class that captures a variable breaks binding: retry those files with records desugared only.
		for (boolean retry = true; ; retry = false) {
			unitsByPath.clear();
			ASTParser parser = ASTParser.newParser(AST.JLS21);
			Map<String, String> options = JavaCore.getOptions();
			JavaCore.setComplianceOptions(JavaCore.VERSION_21, options);
			parser.setCompilerOptions(options);
			parser.setKind(ASTParser.K_COMPILATION_UNIT);
			parser.setResolveBindings(true);
			parser.setBindingsRecovery(true);
			parser.setStatementsRecovery(true);
			parser.setEnvironment(classpath, sourceRoots.toArray(new String[0]), null, true);
			orderedUnits.clear();
			parser.createASTs(allAbsFiles.toArray(new String[0]), encodings, new String[0], new FileASTRequestor() {
				@Override
				public void acceptAST(String sourceFilePath, CompilationUnit ast) {
					unitsByPath.put(sourceFilePath, ast);
				}
			}, null);
			boolean unhoisted = false;
			List<String> errors = new ArrayList<>();
			for (String f : allAbsFiles) {
				CompilationUnit cu = unitsByPath.get(f);
				if (cu == null) {
					System.err.println("j2go: failed to parse " + f);
					System.exit(1);
				}
				orderedUnits.add(cu);
				boolean bad = false;
				for (IProblem p : cu.getProblems()) {
					if (p.isError()) {
						bad = true;
						errors.add("j2go: [error] " + f + ": " + p);
						badFiles.add(f);
					}
				}
				if (bad && retry) for (var e : mirrorToReal.entrySet())
					if (f.startsWith(e.getKey() + "/")) unhoisted |= SourcePrep.unhoist(Path.of(f), Path.of(e.getValue() + f.substring(e.getKey().length())), externalMirrorDirs.contains(e.getKey()) ? dollars : null);
			}
			if (unhoisted) continue;
			errors.forEach(System.err::println);
			problemCount = errors.size();
			break;
		}
		if (problemCount > 0) {
			System.err.println("j2go: " + problemCount + " binding/compile errors" + (keepGoing ? ", continuing (--keep-going)" : ", aborting"));
			if (!keepGoing) System.exit(1);
		}

		if (Profile.active != null) Profile.collapse(orderedUnits, absFiles.size());
		Names names = new Names();
		names.loadPins(Config.dir.resolve("overloads.properties"));
		names.loadOverrides(Config.dir.resolve("names.properties"));
		dev.gowt.j2go.emit.WinLayout.load(Config.dir.resolve("win32_layout.txt"));
		if (platform == Platform.GTK) names.loadOverrides(Config.dir.resolve("gtkstubs-gen/names.properties"));
		Natives natives = new Natives();
		natives.load(Config.dir.resolve("natives.properties"));
		Selectors selectors = new Selectors();
		selectors.load(Config.dir.resolve("selectors.properties"));
		TypeModel model = new TypeModel();
		Path cascadePins = Config.dir.resolve("cascade.properties");
		if (Files.exists(cascadePins) && System.getenv("J2GO_DUMP_CASCADE") == null) Files.readAllLines(cascadePins).stream().filter(l -> !l.startsWith("#") && !l.isBlank()).forEach(l -> { int eq = l.indexOf('='); if (eq < 0) TypeModel.PINNED_CASCADE.add(l); else TypeModel.PINNED_CASCADE_NAMES.put(l.substring(0, eq), l.substring(eq + 1)); });
		model.build(orderedUnits, names);
		if (System.getenv("J2GO_DUMP_CASCADE") != null) model.dumpCascade(Path.of(System.getenv("J2GO_DUMP_CASCADE")));

		Emitter emitter = new Emitter(model, names, natives, selectors);

		boolean reference = platform == Platform.COCOA;
		boolean emittedSwtPlatformFile = false;
		StringBuilder sharedHelpers = new StringBuilder();
		Set<String> sharedHelperImports = new LinkedHashSet<>();
		if (!reference && Profile.active == null) emitter.generatedHelpers.addAll(definedNames(Path.of(outDir, "swt")));
		// Only the primary (pre "--") files are emitted; reference files only fed the model above.
		for (int i = 0; i < absFiles.size(); i++) {
			CompilationUnit cu = orderedUnits.get(i);
			String absPath = absFiles.get(i);
			// A Java stub only types its Go counterpart (hand-written); it is modeled, never emitted.
			if (Config.stubRoots.stream().anyMatch(r -> absPath.startsWith(r.toAbsolutePath() + "/"))) continue;
			String source = Files.readString(Path.of(absPath), StandardCharsets.UTF_8);

			if (System.getenv("J2GO_TRACE") != null) System.err.println("j2go: emitting " + absPath);
			String unitPkg = cu.getPackage().getName().getFullyQualifiedName();
			if (keepGoing && cu.types().get(0) instanceof RecordDeclaration) { System.err.println("j2go: [skip] " + absPath + ": RecordDeclaration not supported"); continue; }
			String unitType = ((AbstractTypeDeclaration) cu.types().get(0)).getName().getIdentifier();
			String unitDir = GoTypes.goPackageDir(unitPkg, unitType);
			boolean commonSource = !reference && platformRoots.stream().noneMatch(absPath::startsWith) && !unitDir.equals(platform.piDir);
			// A common source the reference platform has no file for (it stubs the class by hand) is this platform's own.
			boolean sharedFile = commonSource && Files.exists(Path.of(outDir, unitDir, lastSegment(unitPkg) + "_" + unitType.toLowerCase(Locale.ROOT) + ".go"));
			emitter.separateHelpers = sharedFile;
			if (keepGoing && badFiles.contains(absPath)) { System.err.println("j2go: [skip] " + absPath + ": binding errors"); continue; }
			Emitter.EmitResult result;
			try {
				result = emitter.emitCompilationUnit(cu);
			} catch (RuntimeException e) {
				if (!keepGoing) throw e;
				StackTraceElement[] st = e.getStackTrace();
				System.err.println("j2go: [crash] " + absPath + ": " + e.getClass().getSimpleName() + " at " + (st.length > 0 ? st[0] : "?"));
				continue;
			}
			// The reference platform owns the shared files; here only the helpers they need are kept.
			if (sharedFile) {
				if (GoTypes.goPackageOf(cu.getPackage().getName().getFullyQualifiedName(), ((AbstractTypeDeclaration) cu.types().get(0)).getName().getIdentifier()).equals("swt")) {
					sharedHelpers.append(result.helpers());
					sharedHelperImports.addAll(result.imports());
				}
				continue;
			}

			String realPath = absPath;
			for (var e : mirrorToReal.entrySet()) if (absPath.startsWith(e.getKey() + "/")) realPath = e.getValue() + absPath.substring(e.getKey().length());
			String relPath = swtRootPath.relativize(Path.of(realPath)).toString();
			String javaPackage = cu.getPackage().getName().getFullyQualifiedName();
			String pkgLastSegment = lastSegment(javaPackage);
			String typeName = ((AbstractTypeDeclaration) cu.types().get(0)).getName().getIdentifier();
			String outDirName = GoTypes.goPackageDir(javaPackage, typeName);
			// swt file built only for this GOOS: read from a platform root (a same-named sibling per platform).
			final String srcPath = absPath;
			boolean piFile = outDirName.equals(platform.piDir);
			// DragSourceEvent/DropTargetEvent print TransferData.type, an int on cocoa and win32 and a long on gtk: one file per platform.
			boolean typeDependent = javaPackage.equals("org.eclipse.swt.dnd") && (typeName.equals("DragSourceEvent") || typeName.equals("DropTargetEvent"));
			boolean platformFile = !piFile && (commonSource || typeDependent || platformRoots.stream().anyMatch(srcPath::startsWith));
			String outName = pkgLastSegment + "_" + typeName.toLowerCase(Locale.ROOT) + (platformFile ? "_" + platform.goos : "") + ".go";
			emittedSwtPlatformFile |= platformFile && outDirName.equals("swt");
			// A PI package is wholly one OS's: a tag, not a rename (swt files share names across platforms, PI files do not).
			String header = buildHeader(relPath, source, cu, piFile ? platform.goos : null);

			StringBuilder file = new StringBuilder();
			file.append(header).append('\n');
			file.append("package ").append(GoTypes.goPackageOf(javaPackage, typeName)).append("\n\n");
			if (!result.imports().isEmpty()) {
				file.append("import (\n");
				for (String imp : result.imports()) file.append('\t').append('"').append(imp).append("\"\n");
				file.append(")\n\n");
			}
			file.append(result.body());

			Path outPath = Path.of(outDir, outDirName, outName);
			Files.createDirectories(outPath.getParent());
			Files.writeString(outPath, file.toString(), StandardCharsets.UTF_8);
			System.out.println("wrote " + outPath);
		}

		if (!reference && Profile.active == null && emittedSwtPlatformFile) {
			// The committed shared files also call helpers the reference platform defined in its own files.
			Set<String> called = new HashSet<>();
			var m = java.util.regex.Pattern.compile("\\b((?:upcast|is)[A-Z]\\w*To[A-Z]\\w*|[a-z]\\w*ImplAs\\w+)\\(").matcher(sharedSource(Path.of(outDir, "swt")));
			while (m.find()) called.add(m.group(1));
			called.removeAll(emitter.generatedHelpers);
			sharedHelpers.append(dev.gowt.j2go.emit.HelperFiller.fill(emitter, called));
		}
		if (sharedHelpers.length() > 0) {
			String text = sharedHelpers.toString();
			StringBuilder f = new StringBuilder("// Code generated by j2go. DO NOT EDIT.\n//go:build " + platform.goos + "\n\n// Helpers the committed shared files need and only the reference platform defines in them.\n\npackage swt\n\n");
			sharedHelperImports.removeIf(imp -> imp.endsWith("/swt") || !text.contains(imp.substring(imp.lastIndexOf('/') + 1) + "."));
			if (!sharedHelperImports.isEmpty()) {
				f.append("import (\n");
				for (String imp : sharedHelperImports) f.append("\t\"").append(imp).append("\"\n");
				f.append(")\n\n");
			}
			Path outPath = Path.of(outDir, "swt", "helpers_" + platform.goos + ".go");
			Files.writeString(outPath, f.append(text).toString(), StandardCharsets.UTF_8);
			System.out.println("wrote " + outPath);
		}

		// Written only by the run that emits widgets classes; the other runs leave it alone.
		String registry = emitter.reflectRegistryFile();
		if (registry != null) {
			Path outPath = Path.of(outDir, "swt", "swtreflect", "swtreflect_" + platform.goos + ".go");
			Files.createDirectories(outPath.getParent());
			Files.writeString(outPath, "// Code generated by j2go. DO NOT EDIT.\n\n" + registry, StandardCharsets.UTF_8);
			System.out.println("wrote " + outPath);
		}

		if (System.getenv("J2GO_DUMP_OVERLOADS") != null) names.dumpOverloads(Path.of(System.getenv("J2GO_DUMP_OVERLOADS")));
		if (System.getenv("J2GO_DUMP_PUBLIC") != null) dev.gowt.j2go.emit.PublicApi.dump(emitter, Path.of(System.getenv("J2GO_DUMP_PUBLIC")));
		printSummary(emitter);
	}

	/** Top-level func and type names of the committed shared (GOOS-suffix-free) files of dir. */
	private static Set<String> definedNames(Path dir) throws IOException {
		Set<String> names = new HashSet<>();
		if (!Files.isDirectory(dir)) return names;
		var decl = java.util.regex.Pattern.compile("^(?:func|type) ([A-Za-z_][A-Za-z0-9_]*)", java.util.regex.Pattern.MULTILINE);
		try (var files = Files.list(dir)) {
			for (Path f : (Iterable<Path>) files::iterator) {
				String n = f.getFileName().toString();
				if (!n.endsWith(".go") || n.matches(".*_(darwin|windows|linux)(_test)?\\.go")) continue;
				var m = decl.matcher(Files.readString(f));
				while (m.find()) names.add(m.group(1));
			}
		}
		return names;
	}

	private static String sharedSource(Path dir) throws IOException {
		StringBuilder all = new StringBuilder();
		try (var files = Files.list(dir)) {
			for (Path f : (Iterable<Path>) files::iterator) {
				String n = f.getFileName().toString();
				if (n.endsWith(".go") && !n.matches(".*_(darwin|windows|linux)(_test)?\\.go")) all.append(Files.readString(f));
			}
		}
		return all.toString();
	}

	private static List<String> resolveSources(List<String> relOrAbs, List<String> sourceRoots) {
		List<String> abs = new ArrayList<>();
		for (String f : relOrAbs) {
			Path direct = Path.of(f);
			if (direct.isAbsolute() && Files.exists(direct)) {
				abs.add(direct.toString());
				continue;
			}
			String found = null;
			for (String root : sourceRoots) {
				Path candidate = Path.of(root, f);
				if (Files.exists(candidate)) {
					found = candidate.toString();
					break;
				}
			}
			if (found == null) {
				System.err.println("cannot locate source file: " + f);
				System.exit(2);
			}
			abs.add(found);
		}
		return abs;
	}

	// The licence comment is a "/***" block, which JDT takes for the package's Javadoc, so the package node starts at 0:
	// the text up to the package name (minus the keyword) is the original header.
	private static String buildHeader(String relPath, String source, CompilationUnit cu, String buildTag) {
		String leading = source.substring(0, cu.getPackage().getName().getStartPosition()).replaceFirst("package\\s*$", "").strip();
		StringBuilder b = new StringBuilder();
		String rev = Profile.active == null ? (relPath.contains("eclipse.platform.ui") ? UI_COMMIT : SWT_COMMIT) : Profile.active.revision(relPath);
		b.append("// Code generated by j2go from ").append(relPath).append(rev == null ? "" : " @ " + rev)
				.append(". DO NOT EDIT.\n");
		if (buildTag != null) b.append("//go:build ").append(buildTag).append("\n\n");
		if (!leading.isEmpty()) b.append(buildTag == null ? "//\n" : "").append(lineComments(leading)).append('\n');
		return b.toString();
	}

	// Block comments become // lines (the "/****" and "****/" rulers and the leading " * " are dropped); // comments stay.
	private static String lineComments(String comments) {
		StringBuilder out = new StringBuilder();
		boolean block = false;
		for (String line : comments.split("\n")) {
			String t = line.strip();
			if (!block && t.startsWith("//")) { out.append(t).append('\n'); continue; }
			if (!block && t.startsWith("/*")) { block = true; t = t.replaceFirst("^/\\*+", ""); }
			else if (block) t = t.replaceFirst("^\\*(?!/)", "");
			boolean end = t.endsWith("*/");
			if (end) { t = t.substring(0, t.length() - 2).replaceFirst("\\*+$", ""); block = false; }
			t = t.stripTrailing();
			if (t.startsWith(" ")) t = t.substring(1);
			if (t.isBlank() && (end || out.length() == 0)) continue;
			out.append(t.isEmpty() ? "//" : "// " + t).append('\n');
		}
		return out.toString().stripTrailing();
	}

	private static <T> List<T> concat(List<T> a, List<T> b) {
		List<T> all = new ArrayList<>(a);
		all.addAll(b);
		return all;
	}

	private static String lastSegment(String dotted) {
		int idx = dotted.lastIndexOf('.');
		return idx < 0 ? dotted : dotted.substring(idx + 1);
	}

	private static void printSummary(Emitter emitter) {
		System.out.println();
		System.out.println("=== j2go summary ===");
		if (emitter.unsupported.isEmpty()) {
			System.out.println("unsupported markers: none");
			return;
		}
		Map<String, Integer> byKind = new TreeMap<>();
		for (String u : emitter.unsupported) {
			String kind = u.substring(0, u.indexOf(':'));
			byKind.merge(kind, 1, Integer::sum);
		}
		System.out.println("unsupported markers by kind:");
		for (var e : byKind.entrySet()) System.out.println("  " + e.getKey() + ": " + e.getValue());
		System.out.println("details:");
		for (String u : emitter.unsupported) System.out.println("  " + u);
	}
}
