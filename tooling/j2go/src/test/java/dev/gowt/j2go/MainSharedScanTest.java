package dev.gowt.j2go;

import org.junit.jupiter.api.Test;

import java.nio.file.*;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/**
 * The shared-file scans behind shared-file detection and helper filling ({@code definedNames},
 * {@code sharedSource}): they must see shared declarations including the parts of shared files,
 * and must not pick up any GOOS-specific file or part. That holds because a platform file keeps
 * its GOOS suffix at the very end of every name (widgets_display_settext_darwin.go), which the
 * {@code .*_(darwin|windows|linux)(_test)?\.go} filter matches - the names Must 1 produces are
 * exactly the names the unchanged regexes select on.
 */
class MainSharedScanTest {

	@Test
	void sharedScansSeeSharedPartsButNoGoosFiles() throws Exception {
		Path dir = Files.createTempDirectory("j2go-shared-scan");
		write(dir, "widgets_button.go", "package swt\n\nfunc ButtonFoo() {}\n\ntype Button struct {\n}\n");
		write(dir, "widgets_button_settext.go", "package swt\n\nfunc ButtonSetText() {}\n");
		write(dir, "widgets_display_darwin.go", "package swt\n\nfunc DisplayCreate() {}\n");
		// Parts of platform files, every GOOS at the very end:
		write(dir, "widgets_display_settext_darwin.go", "package swt\n\nfunc DisplaySetText() {}\n");
		write(dir, "widgets_display_settext_windows.go", "package swt\n\nfunc DisplaySetTextW() {}\n");
		write(dir, "helpers_windows.go", "package swt\n\nfunc winHelper() {}\n");
		write(dir, "button_linux_test.go", "package swt\n\nfunc TestButtonX() {}\n");

		Set<String> names = Main.definedNames(dir);
		assertTrue(names.contains("ButtonFoo"), names.toString());
		assertTrue(names.contains("Button"), names.toString());
		assertTrue(names.contains("ButtonSetText"), "a shared file's part is shared too: " + names);
		assertFalse(names.contains("DisplayCreate"), names.toString());
		assertFalse(names.contains("DisplaySetText"), "foreign-OS part leaked into the shared scan: " + names);
		assertFalse(names.contains("DisplaySetTextW"), "own-OS platform part is not a shared declaration: " + names);
		assertFalse(names.contains("winHelper"), names.toString());
		assertFalse(names.contains("TestButtonX"), names.toString());

		String source = Main.sharedSource(dir);
		assertTrue(source.contains("ButtonSetText"));
		assertFalse(source.contains("DisplayCreate"));
		assertFalse(source.contains("DisplaySetText("), "foreign-OS part leaked into the shared source");
		assertFalse(source.contains("DisplaySetTextW("));
		assertFalse(source.contains("winHelper("));
	}

	private static void write(Path dir, String name, String text) throws Exception {
		Files.writeString(dir.resolve(name), text);
	}
}
