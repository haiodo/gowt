package dev.gowt.j2go;

import dev.gowt.j2go.emit.FilePlanner;
import dev.gowt.j2go.emit.OutputModel;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/** Packing of method chunks into named part files: grouping by method name, target/ceiling
 *  lines, oversized-group splitting and deterministic names. */
class FilePlannerTest {

	private static OutputModel.Chunk chunk(String owner, String group, int lines) {
		String text = lines <= 1 ? "x" : "x\n".repeat(lines - 1) + "x";
		return new OutputModel.Chunk(owner, group, text);
	}

	private static Map<String, String> bodies(List<OutputModel.Part> parts) {
		Map<String, String> out = new LinkedHashMap<>();
		for (OutputModel.Part p : parts) out.put(p.suffix, p.body());
		return out;
	}

	@Test
	void emptyInputGivesNoParts() {
		assertTrue(FilePlanner.plan(List.of(), 800, 1200).isEmpty());
	}

	@Test
	void overloadsShareOnePartNamedAfterTheMethod() {
		OutputModel.Chunk a = chunk("Point", "setLocation", 10);
		OutputModel.Chunk b = chunk("Point", "setLocation", 12);
		OutputModel.Chunk c = chunk("Point", "translate", 20);
		List<OutputModel.Part> parts = FilePlanner.plan(List.of(a, b, c), 800, 1200);
		assertEquals(1, parts.size());
		assertEquals("setlocation", parts.get(0).suffix);
		assertEquals(42, parts.get(0).lines());
	}

	@Test
	void groupsPackUpToTargetAndSplitBeyondIt() {
		List<OutputModel.Chunk> chunks = new ArrayList<>();
		for (int i = 0; i < 6; i++) chunks.add(chunk("C", "m" + i, 100));
		List<OutputModel.Part> parts = FilePlanner.plan(chunks, 300, 1200);
		assertEquals(2, parts.size());
		assertEquals(300, parts.get(0).lines());
		assertEquals(300, parts.get(1).lines());
		assertEquals("m0", parts.get(0).suffix);
		assertEquals("m3", parts.get(1).suffix);
	}

	@Test
	void chunksStayWithinCeilingExceptASingleIndivisibleChunk() {
		// A group of same-name chunks over the ceiling splits at chunk boundaries, each piece
		// within the ceiling; one chunk alone over the ceiling stays as the only exception.
		List<OutputModel.Chunk> group = new ArrayList<>();
		for (int i = 0; i < 4; i++) group.add(chunk("C", "big", 400));
		List<OutputModel.Part> parts = FilePlanner.plan(group, 800, 1200);
		assertEquals(2, parts.size());
		assertEquals(1200, parts.get(0).lines());
		assertEquals(400, parts.get(1).lines());
		assertEquals("big", parts.get(0).suffix);
		assertEquals("big_c", parts.get(1).suffix);

		OutputModel.Chunk huge = chunk("C", "huge", 5000);
		List<OutputModel.Part> alone = FilePlanner.plan(List.of(huge, chunk("C", "other", 10)), 800, 1200);
		assertEquals(2, alone.size());
		assertEquals(5000, alone.get(0).lines());
		assertEquals("huge", alone.get(0).suffix);
	}

	@Test
	void oversizedGroupSplitsByOwner() {
		List<OutputModel.Chunk> chunks = List.of(
				chunk("Outer", "apply", 700), chunk("Inner", "apply", 700));
		List<OutputModel.Part> parts = FilePlanner.plan(chunks, 800, 1200);
		assertEquals(2, parts.size());
		assertEquals("apply", parts.get(0).suffix);
		assertEquals("apply_outer", parts.get(1).suffix);
	}

	@Test
	void namesAreSanitizedAndDeduplicated() {
		OutputModel.Chunk a = chunk("C", "foo", 900);
		OutputModel.Chunk b = chunk("C", "Foo", 900);
		List<OutputModel.Part> parts = FilePlanner.plan(List.of(a, b), 800, 1200);
		assertEquals(List.of("foo", "foo_c"), List.of(parts.get(0).suffix, parts.get(1).suffix));
		assertEquals(Set.of("foo", "foo_c"), bodies(parts).keySet());
	}

	@Test
	void suffixesDoNotDependOnTheDefaultLocale() {
		// Turkish lowercases 'I' to dotless 'ı': with the default locale a group like "InitSize"
		// would sanitize differently and the part file name would change with the environment.
		Locale saved = Locale.getDefault();
		Locale.setDefault(Locale.forLanguageTag("tr"));
		try {
			List<OutputModel.Part> parts = FilePlanner.plan(
					List.of(chunk("C", "InitSize", 900), chunk("C", "initSize", 900)), 800, 1200);
			assertEquals(List.of("initsize", "initsize_c"), List.of(parts.get(0).suffix, parts.get(1).suffix));
		} finally {
			Locale.setDefault(saved);
		}
	}

	@Test
	void planningIsDeterministicUnderInputOrder() {
		List<OutputModel.Chunk> chunks = new ArrayList<>();
		String[] groups = {"gamma", "alpha", "beta", "alpha"};
		for (int i = 0; i < 40; i++) chunks.add(chunk("C" + (i % 3), groups[i % 4], 50 + i));
		List<OutputModel.Part> first = FilePlanner.plan(chunks, 400, 1200);
		Collections.shuffle(chunks, new Random(42));
		List<OutputModel.Part> second = FilePlanner.plan(chunks, 400, 1200);
		assertEquals(bodies(first), bodies(second));
		List<String> s1 = new ArrayList<>(), s2 = new ArrayList<>();
		first.forEach(p -> s1.add(p.suffix));
		second.forEach(p -> s2.add(p.suffix));
		assertEquals(s1, s2);
	}
}
