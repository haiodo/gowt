package dev.gowt.j2go;

import org.junit.jupiter.api.DynamicTest;
import org.junit.jupiter.api.TestFactory;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;
import java.util.stream.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * testdata/&lt;rule&gt;/in/*.java is translated in one j2go run; each emitted file must equal testdata/&lt;rule&gt;/want/&lt;name&gt;.go.
 * -Dj2go.update=true rewrites want/ from the current output.
 */
class GoldenTest {

	private static final Path TESTDATA = Path.of("testdata");
	// Under a common SWT root, so SourcePrep (records, local classes) applies as it does to real sources.
	private static final String ROOT = "bundles/org.eclipse.swt/Eclipse SWT/common/fixture";

	@TestFactory
	Stream<DynamicTest> rules() throws IOException {
		List<Path> rules;
		try (Stream<Path> s = Files.list(TESTDATA)) {
			rules = s.filter(Files::isDirectory).sorted().collect(Collectors.toList());
		}
		return rules.stream().map(r -> DynamicTest.dynamicTest(r.getFileName().toString(), () -> run(r)));
	}

	private void run(Path rule) throws Exception {
		Path in = rule.resolve("in"), want = rule.resolve("want");
		List<String> names;
		try (Stream<Path> s = Files.list(in)) {
			names = s.map(p -> p.getFileName().toString()).filter(n -> n.endsWith(".java")).sorted().collect(Collectors.toList());
		}
		assertFalse(names.isEmpty(), "no fixtures in " + in);
		Path work = Files.createTempDirectory("j2go-golden");
		Path src = work.resolve("swt"), out = work.resolve("out"), cwd = Files.createDirectories(work.resolve("cwd"));
		Files.createDirectories(src.resolve(ROOT));
		for (String n : names) Files.copy(in.resolve(n), src.resolve(ROOT).resolve(n));

		// A fresh JVM per rule: j2go keeps its state in statics. cwd is empty, so no repo config is picked up.
		List<String> cmd = new ArrayList<>(List.of(Path.of(System.getProperty("java.home"), "bin", "java").toString(),
				"-cp", System.getProperty("java.class.path"), "dev.gowt.j2go.Main", "--swt", src.toString(), "--out", out.toString()));
		for (String n : names) cmd.add("fixture/" + n);
		Process p = new ProcessBuilder(cmd).directory(cwd.toFile()).redirectErrorStream(true).start();
		String log = new String(p.getInputStream().readAllBytes(), StandardCharsets.UTF_8);
		assertEquals(0, p.waitFor(), log);

		Map<String, String> got = new TreeMap<>();
		Path gen = out.resolve("swt");
		if (Files.isDirectory(gen)) try (Stream<Path> s = Files.list(gen)) {
			for (Path f : (Iterable<Path>) s::iterator) got.put(f.getFileName().toString(), Files.readString(f, StandardCharsets.UTF_8));
		}
		if (Boolean.getBoolean("j2go.update")) {
			if (Files.isDirectory(want)) try (Stream<Path> s = Files.list(want)) { for (Path f : (Iterable<Path>) s::iterator) Files.delete(f); }
			Files.createDirectories(want);
			for (var e : got.entrySet()) Files.writeString(want.resolve(e.getKey()), e.getValue(), StandardCharsets.UTF_8);
			return;
		}
		Set<String> wanted = new TreeSet<>();
		try (Stream<Path> s = Files.list(want)) { s.forEach(f -> wanted.add(f.getFileName().toString())); }
		assertEquals(wanted, got.keySet(), "emitted files differ from want/");
		for (var e : got.entrySet())
			assertEquals(Files.readString(want.resolve(e.getKey()), StandardCharsets.UTF_8), e.getValue(), e.getKey() + " differs from want/" + e.getKey() + "\n" + log);
	}
}
