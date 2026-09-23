package dev.gowt.j2go;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;
import java.util.Properties;

/** enumConstName -> selector string, extracted from Selector.java (elided, see README). */
public class Selectors {

	private final Map<String, String> byConstName = new HashMap<>();

	void load(Path propsFile) throws IOException {
		if (!Files.exists(propsFile)) return;
		Properties p = new Properties();
		try (var in = Files.newInputStream(propsFile)) {
			p.load(in);
		}
		for (String key : p.stringPropertyNames()) {
			byConstName.put(key, p.getProperty(key));
		}
	}

	/** The selector string (e.g. "sendSelection:") for enum constant name (e.g. "sel_sendSelection_"). */
	public String stringFor(String constName) {
		return byConstName.get(constName);
	}
}
