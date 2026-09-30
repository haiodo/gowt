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

	// Java stubs for test-harness classes outside the SWT repo (org.eclipse.test.Screenshots).
	private static final String STUB_ROOT = "tooling/j2go/stubs";

	private static final String SWT_COMMIT = "af630a9093";

	public static void main(String[] args) throws Exception {
		String swtRoot = null;
		String outDir = null;
		Platform platform = Platform.COCOA;
		String[] classpath = new String[0];
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
				case "--" -> target = refFiles;
				default -> target.add(args[i]);
			}
		}
		if (swtRoot == null || outDir == null || files.isEmpty()) {
			System.err.println("usage: j2go --swt <swt repo root> --out <dir> [--platform cocoa|win32|gtk] <java files...> [-- <reference-only java files...>]");
			System.exit(2);
		}

		if (!platform.implemented()) {
			System.err.println("j2go: platform " + platform.swtName + " is not implemented yet");
			System.exit(2);
		}

		Path swtRootPath = Path.of(swtRoot).toAbsolutePath().normalize();
		List<String> sourceRoots = new ArrayList<>();
		List<String> platformRoots = new ArrayList<>();
		List<String> allRoots = new ArrayList<>(platform.roots());
		allRoots.addAll(Platform.commonRoots());
		for (String r : allRoots) {
			Path p = swtRootPath.resolve(r);
			if (!Files.isDirectory(p)) continue;
			sourceRoots.add(p.toString());
			if (platform.roots().contains(r)) platformRoots.add(p + "/");
		}
		if (Files.isDirectory(Path.of(STUB_ROOT))) sourceRoots.add(Path.of(STUB_ROOT).toAbsolutePath().toString());

		List<String> absFiles = resolveSources(files, sourceRoots);
		List<String> absRefFiles = resolveSources(refFiles, sourceRoots);
		List<String> allAbsFiles = new ArrayList<>(absFiles);
		allAbsFiles.addAll(absRefFiles);

		ASTParser parser = ASTParser.newParser(AST.JLS21);
		Map<String, String> options = JavaCore.getOptions();
		JavaCore.setComplianceOptions(JavaCore.VERSION_21, options);
		parser.setCompilerOptions(options);
		parser.setKind(ASTParser.K_COMPILATION_UNIT);
		parser.setResolveBindings(true);
		parser.setBindingsRecovery(true);
		parser.setStatementsRecovery(true);
		parser.setEnvironment(classpath, sourceRoots.toArray(new String[0]), null, true);

		Map<String, CompilationUnit> unitsByPath = new LinkedHashMap<>();
		String[] encodings = new String[allAbsFiles.size()];
		Arrays.fill(encodings, "UTF-8");
		parser.createASTs(allAbsFiles.toArray(new String[0]), encodings, new String[0], new FileASTRequestor() {
			@Override
			public void acceptAST(String sourceFilePath, CompilationUnit ast) {
				unitsByPath.put(sourceFilePath, ast);
			}
		}, null);

		List<CompilationUnit> orderedUnits = new ArrayList<>();
		int problemCount = 0;
		for (String f : allAbsFiles) {
			CompilationUnit cu = unitsByPath.get(f);
			if (cu == null) {
				System.err.println("j2go: failed to parse " + f);
				System.exit(1);
			}
			orderedUnits.add(cu);
			for (IProblem p : cu.getProblems()) {
				if (p.isError()) {
					System.err.println("j2go: [error] " + f + ": " + p);
					problemCount++;
				}
			}
		}
		if (problemCount > 0) {
			System.err.println("j2go: " + problemCount + " binding/compile errors, aborting");
			System.exit(1);
		}

		Names names = new Names();
		names.loadOverrides(Path.of("tooling/j2go/names.properties"));
		Natives natives = new Natives();
		natives.load(Path.of("tooling/j2go/natives.properties"));
		Selectors selectors = new Selectors();
		selectors.load(Path.of("tooling/j2go/selectors.properties"));
		TypeModel model = new TypeModel();
		model.build(orderedUnits, names);

		Emitter emitter = new Emitter(model, names, natives, selectors);

		// Only the primary (pre "--") files are emitted; reference files only fed the model above.
		for (int i = 0; i < absFiles.size(); i++) {
			CompilationUnit cu = orderedUnits.get(i);
			String absPath = absFiles.get(i);
			String source = Files.readString(Path.of(absPath), StandardCharsets.UTF_8);

			if (System.getenv("J2GO_TRACE") != null) System.err.println("j2go: emitting " + absPath);
			Emitter.EmitResult result = emitter.emitCompilationUnit(cu);

			String relPath = swtRootPath.relativize(Path.of(absPath)).toString();
			String javaPackage = cu.getPackage().getName().getFullyQualifiedName();
			String pkgLastSegment = lastSegment(javaPackage);
			String typeName = ((TypeDeclaration) cu.types().get(0)).getName().getIdentifier();
			String outDirName = GoTypes.goPackageDir(javaPackage, typeName);
			// swt file built only for this GOOS: read from a platform root (a same-named sibling per platform).
			final String srcPath = absPath;
			boolean piFile = outDirName.equals(platform.piDir);
			boolean platformFile = !piFile && platformRoots.stream().anyMatch(srcPath::startsWith);
			String outName = pkgLastSegment + "_" + typeName.toLowerCase(Locale.ROOT) + (platformFile ? "_" + platform.goos : "") + ".go";
			String header = buildHeader(relPath, source, cu);
			// A PI package is wholly one OS's: a tag, not a rename (swt files share names across platforms, PI files do not).
			if (piFile) header = header.replaceFirst("\n", "\n//go:build " + platform.goos + "\n\n");

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

		// Written only by the run that emits widgets classes; the other runs leave it alone.
		String registry = emitter.reflectRegistryFile();
		if (registry != null) {
			Path outPath = Path.of(outDir, "swt", "swtreflect", "swtreflect_" + platform.goos + ".go");
			Files.createDirectories(outPath.getParent());
			Files.writeString(outPath, "// Code generated by j2go. DO NOT EDIT.\n\n" + registry, StandardCharsets.UTF_8);
			System.out.println("wrote " + outPath);
		}

		printSummary(emitter);
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

	private static String buildHeader(String relPath, String source, CompilationUnit cu) {
		int pkgStart = cu.getPackage().getStartPosition();
		String leading = source.substring(0, pkgStart).strip();
		StringBuilder b = new StringBuilder();
		b.append("// Code generated by j2go from ").append(relPath).append(" @ ").append(SWT_COMMIT)
				.append(". DO NOT EDIT.\n");
		if (!leading.isEmpty()) b.append(leading).append('\n');
		return b.toString();
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
