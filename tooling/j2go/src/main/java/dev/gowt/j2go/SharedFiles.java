package dev.gowt.j2go;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

/**
 * Generated Go files from a common (or example/test) source are shared by the platforms while their text agrees. Cocoa
 * owns the unsuffixed file; another platform whose text differs (impl dispatch names depend on its widget tree) writes
 * {@code name_<goos>.go} and the unsuffixed file then carries {@code //go:build !<goos>} for it.
 */
final class SharedFiles {

	private static final Pattern BUILD = Pattern.compile("(?m)^//go:build [^\\n]*\\n\\n?");
	private static final String[] GOOS = { "darwin", "windows", "linux" };

	private SharedFiles() {}

	private static String plain(String text) {
		return BUILD.matcher(text).replaceFirst("");
	}

	private static Path variant(Path shared, String goos) {
		String n = shared.getFileName().toString();
		return shared.resolveSibling(n.substring(0, n.length() - 3) + "_" + goos + ".go");
	}

	private static String withConstraint(String text, List<String> excluded) {
		String plain = plain(text);
		if (excluded.isEmpty()) return plain;
		StringBuilder c = new StringBuilder("//go:build");
		for (int i = 0; i < excluded.size(); i++) c.append(i == 0 ? " !" : " && !").append(excluded.get(i));
		return plain.replaceFirst("\n", "\n" + c + "\n\n");
	}

	private static List<String> variantsOf(Path shared, String except) {
		List<String> v = new ArrayList<>();
		for (String g : GOOS) if (!g.equals(except) && Files.exists(variant(shared, g))) v.add(g);
		return v;
	}

	/** Writes the output of a common-derived source for platform p (goos) into the shared file or its variant. */
	static void write(Path shared, String text, Platform p) throws IOException {
		Files.createDirectories(shared.getParent());
		if (p == Platform.COCOA) {
			Files.writeString(shared, withConstraint(text, variantsOf(shared, "darwin")), StandardCharsets.UTF_8);
			return;
		}
		Path mine = variant(shared, p.goos);
		if (!Files.exists(shared)) {
			Files.writeString(shared, plain(text), StandardCharsets.UTF_8);
			return;
		}
		String existing = Files.readString(shared, StandardCharsets.UTF_8);
		if (plain(existing).equals(plain(text))) {
			Files.deleteIfExists(mine);
			Files.writeString(shared, withConstraint(existing, variantsOf(shared, "darwin")), StandardCharsets.UTF_8);
			return;
		}
		Files.writeString(mine, plain(text), StandardCharsets.UTF_8);
		Files.writeString(shared, withConstraint(existing, variantsOf(shared, "darwin")), StandardCharsets.UTF_8);
	}
}
