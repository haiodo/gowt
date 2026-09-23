package dev.gowt.j2go;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;
import java.util.Properties;

/** javaMethodBaseName -> C symbol override for the rare case it isn't the Java name (see README). */
public class Natives {

	private final Map<String, String> overrides = new HashMap<>();

	void load(Path propsFile) throws IOException {
		if (!Files.exists(propsFile)) return;
		Properties p = new Properties();
		try (var in = Files.newInputStream(propsFile)) {
			p.load(in);
		}
		for (String key : p.stringPropertyNames()) {
			overrides.put(key, p.getProperty(key).trim());
		}
	}

	/** The C symbol a native method's Go wrapper should Dlsym for. */
	public String symbolFor(String javaMethodName) {
		return overrides.getOrDefault(javaMethodName, javaMethodName);
	}
}
