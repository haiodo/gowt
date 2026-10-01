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

	/** `name=manual`: the Go function is hand-written in the PI package (JNI glue with no C symbol of its own). */
	public boolean isManual(String javaMethodName) {
		return "manual".equals(overrides.get(javaMethodName));
	}

	/** The C symbol a native method's Go wrapper should Dlsym for. */
	public String symbolFor(String javaMethodName) {
		String s = overrides.getOrDefault(javaMethodName, javaMethodName);
		return s.startsWith("fn:") ? s.substring(3) : s;
	}

	/** `name=fn:symbol`: a call of a function even where the Java name looks like a constant accessor (GTK_TYPE_X). */
	public boolean isFunction(String javaMethodName) {
		return overrides.getOrDefault(javaMethodName, "").startsWith("fn:");
	}
}
