package dev.gowt.j2go;

import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/** Per-project settings of one run, set by Main before any other class reads them. */
public final class Config {
	/** Directory with the *.properties pins, manual.txt, win32_layout.txt and the gtk stubs. */
	public static Path dir = Path.of("tooling/j2go");
	/** Java-only roots that type the hand-written Go counterparts: modeled, never emitted. */
	public static List<Path> stubRoots = new ArrayList<>(List.of(Path.of("tooling/j2go/stubs")));
	public static String runtimeImport = "github.com/haiodo/gowt/internal/jrt";
	public static String junitImport = "github.com/haiodo/gowt/internal/junit";

	private Config() {}
}
