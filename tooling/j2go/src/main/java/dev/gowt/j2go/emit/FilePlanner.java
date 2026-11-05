package dev.gowt.j2go.emit;

import java.util.*;

/**
 * Deterministic packing of method chunks into part files (roadmap: declaration-aware parts).
 * Chunks group by method name so overloads stay in one file; consecutive groups fill a part up to
 * the target line count; a part never exceeds the ceiling except for a single indivisible chunk.
 * Part suffixes carry the sanitized method name, never an opaque part number.
 */
public final class FilePlanner {

	public static final int DEFAULT_TARGET_LINES = 800;
	public static final int DEFAULT_MAX_LINES = 1200;

	private FilePlanner() {}

	/** Pack {@code chunks} into ordered parts. The input list is not modified; sorting is a total
	 *  order (group, owner, text), so the plan is identical for any input permutation and across
	 *  consecutive generation runs. */
	public static List<OutputModel.Part> plan(List<OutputModel.Chunk> chunks, int targetLines, int maxLines) {
		if (chunks.isEmpty()) return List.of();
		List<OutputModel.Chunk> sorted = new ArrayList<>(chunks);
		sorted.sort(Comparator.comparing((OutputModel.Chunk c) -> c.group)
				.thenComparing(c -> c.owner).thenComparing(c -> c.text.toString()));
		List<OutputModel.Part> parts = new ArrayList<>();
		int i = 0;
		while (i < sorted.size()) {
			String group = sorted.get(i).group;
			int j = i;
			while (j < sorted.size() && sorted.get(j).group.equals(group)) j++;
			List<OutputModel.Chunk> groupChunks = sorted.subList(i, j);
			for (List<OutputModel.Chunk> piece : pieces(groupChunks, maxLines)) {
				OutputModel.Part last = parts.isEmpty() ? null : parts.get(parts.size() - 1);
				if (last != null && last.lines() > 0 && last.lines() + lines(piece) > targetLines) {
					parts.add(new OutputModel.Part(piece, group));
				} else if (last != null) {
					last.chunks.addAll(piece);
				} else {
					parts.add(new OutputModel.Part(piece, group));
				}
			}
			i = j;
		}
		nameParts(parts);
		return parts;
	}

	// A same-name group over the ceiling splits at owner boundaries, then at chunk boundaries so
	// every piece stays within the ceiling; a single chunk over it stays alone (indivisible).
	private static List<List<OutputModel.Chunk>> pieces(List<OutputModel.Chunk> group, int maxLines) {
		if (lines(group) <= maxLines) return List.of(group);
		List<List<OutputModel.Chunk>> out = new ArrayList<>();
		List<OutputModel.Chunk> cur = new ArrayList<>();
		for (OutputModel.Chunk c : group) {
			if (!cur.isEmpty() && (!c.owner.equals(cur.get(0).owner) || lines(cur) + c.lines() > maxLines)) {
				out.add(cur);
				cur = new ArrayList<>();
			}
			cur.add(c);
		}
		if (!cur.isEmpty()) out.add(cur);
		return out;
	}

	private static void nameParts(List<OutputModel.Part> parts) {
		// Parts sharing a sanitized name basis: the largest keeps the plain method name, the rest
		// get their first chunk's owner as a disambiguator, then a counter as the last resort.
		Map<String, List<OutputModel.Part>> byName = new LinkedHashMap<>();
		for (OutputModel.Part p : parts) byName.computeIfAbsent(sanitize(p.nameKey), k -> new ArrayList<>()).add(p);
		Set<String> used = new HashSet<>();
		for (var e : byName.entrySet()) {
			List<OutputModel.Part> same = e.getValue();
			same.sort(Comparator.comparingInt((OutputModel.Part p) -> p.lines()).reversed());
			for (int i = 0; i < same.size(); i++) {
				String s = i == 0 ? e.getKey() : e.getKey() + "_" + sanitize(same.get(i).chunks.get(0).owner);
				for (int n = 2; !used.add(s); n++) s = e.getKey() + "_" + n;
				same.get(i).suffix = s;
			}
		}
	}

	static String sanitize(String name) {
		String lower = name.toLowerCase(Locale.ROOT);
		StringBuilder b = new StringBuilder(lower.length());
		for (int i = 0; i < lower.length(); i++) {
			char c = lower.charAt(i);
			b.append(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' ? c : '_');
		}
		return b.toString();
	}

	private static int lines(List<OutputModel.Chunk> chunks) {
		int n = 0;
		for (OutputModel.Chunk c : chunks) n += c.lines();
		return n;
	}
}
