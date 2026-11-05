package dev.gowt.j2go;

import org.junit.jupiter.api.Test;

import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;
import java.util.regex.Pattern;
import java.util.stream.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * The declaration-aware parts layout (roadmap TSK-03): two consecutive generations produce
 * identical trees, and splitting a unit into a base file plus method-named parts does not change
 * the exported API (every func/type declaration survives verbatim).
 */
class PartsLayoutTest {

	private static final Path IN = Path.of("testdata/parts/in");
	private static final String ROOT = "bundles/org.eclipse.swt/Eclipse SWT/common/fixture";

	@Test
	void twoConsecutiveGenerationsProduceIdenticalTrees() throws Exception {
		Map<String, String> first = generate(List.of("--part-target", "25"));
		Map<String, String> second = generate(List.of("--part-target", "25"));
		assertEquals(first, second);
		assertTrue(first.size() > 1, "expected a base file plus method parts, got " + first.keySet());
		assertTrue(first.containsKey("fixture_parts.go"), "base file keeps the current name");
	}

	@Test
	void eofConventionIsPinnedPerLayout() throws Exception {
		// Flag-less legacy output keeps its historical trailing blank line (byte-identical to
		// every committed single-file unit); split output ends with one newline, as gofmt
		// enforces on committed generated files.
		Map<String, String> legacy = generate(List.of());
		assertTrue(legacy.values().iterator().next().endsWith("}\n\n"), "legacy EOF changed");
		Map<String, String> split = generate(List.of("--part-target", "25"));
		assertFalse(split.isEmpty());
		split.values().forEach(text -> assertTrue(text.endsWith("}\n") && !text.endsWith("}\n\n"),
				"split output must end with exactly one newline"));
	}

	@Test
	void noPartFlagKeepsTheLegacySingleFile() throws Exception {
		// Splitting is opt-in: a run without --part-target must reproduce the previous
		// single-file layout exactly, whatever the unit's size.
		Map<String, String> legacy = generate(List.of());
		Map<String, String> whole = generate(List.of("--part-target", "100000"));
		assertEquals(1, legacy.size(), legacy.keySet().toString());
		assertEquals(whole, legacy);
	}

	@Test
	void splittingDoesNotChangeTheExportedApi() throws Exception {
		Map<String, String> split = generate(List.of("--part-target", "25"));
		Map<String, String> whole = generate(List.of("--part-target", "100000"));
		assertEquals(1, whole.size(), "a huge target must keep the legacy single file");
		assertEquals(apiOf(whole), apiOf(split));
	}

	/** Sorted func/type declaration lines of every generated file - the exported API surface. */
	private static Set<String> apiOf(Map<String, String> tree) {
		Pattern decl = Pattern.compile("^(func|type) .*$", Pattern.MULTILINE);
		Set<String> api = new TreeSet<>();
		tree.values().forEach(text -> {
			var m = decl.matcher(text);
			while (m.find()) api.add(m.group());
		});
		return api;
	}

	/** Run j2go on the parts fixture in a fresh JVM (j2go keeps state in statics), keyed by path
	 *  under the output dir, mirroring GoldenTest's harness. */
	private static Map<String, String> generate(List<String> extraArgs) throws Exception {
		List<String> names;
		try (Stream<Path> s = Files.list(IN)) {
			names = s.map(p -> p.getFileName().toString()).filter(n -> n.endsWith(".java")).sorted().collect(Collectors.toList());
		}
		Path work = Files.createTempDirectory("j2go-parts");
		Path src = work.resolve("swt"), out = work.resolve("out"), cwd = Files.createDirectories(work.resolve("cwd"));
		Path inputs = src.resolve(ROOT);
		Files.createDirectories(inputs);
		for (String n : names) Files.copy(IN.resolve(n), inputs.resolve(n));
		List<String> cmd = new ArrayList<>(List.of(Path.of(System.getProperty("java.home"), "bin", "java").toString(),
				"-cp", System.getProperty("java.class.path"), "dev.gowt.j2go.Main", "--swt", src.toString(), "--out", out.toString()));
		cmd.addAll(extraArgs);
		for (String n : names) cmd.add("fixture/" + n);
		Process p = new ProcessBuilder(cmd).directory(cwd.toFile()).redirectErrorStream(true).start();
		String log = new String(p.getInputStream().readAllBytes(), StandardCharsets.UTF_8);
		assertEquals(0, p.waitFor(), log);
		Map<String, String> got = new TreeMap<>();
		try (Stream<Path> s = Files.walk(out)) {
			for (Path f : (Iterable<Path>) s.filter(Files::isRegularFile)::iterator) {
				got.put(out.relativize(f).toString().replaceFirst("^swt/", ""), Files.readString(f, StandardCharsets.UTF_8));
			}
		}
		return got;
	}
}
