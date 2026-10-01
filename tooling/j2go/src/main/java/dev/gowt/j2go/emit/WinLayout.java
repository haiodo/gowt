package dev.gowt.j2go.emit;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;

/** C size of each Win32 PI struct and offset/size of its fields (win32_layout.txt, made by win32/layout.py). */
public final class WinLayout {

	record Field(int offset, int size) {}

	private static final Map<String, Integer> SIZES = new HashMap<>();
	private static final Map<String, Field> FIELDS = new HashMap<>();

	private WinLayout() {}

	public static void load(Path file) throws IOException {
		if (!Files.exists(file)) return;
		for (String line : Files.readAllLines(file)) {
			if (line.isBlank() || line.startsWith("#")) continue;
			String[] p = line.trim().split("\\s+");
			if (p.length == 2) SIZES.put(p[0], Integer.parseInt(p[1]));
			else FIELDS.put(p[0], new Field(Integer.parseInt(p[1]), Integer.parseInt(p[2])));
		}
	}

	static Integer size(String cls) {
		return SIZES.get(cls);
	}

	static Field field(String cls, String name) {
		return FIELDS.get(cls + "." + name);
	}
}
