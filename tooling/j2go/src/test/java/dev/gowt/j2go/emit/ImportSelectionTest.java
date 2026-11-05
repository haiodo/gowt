package dev.gowt.j2go.emit;

import org.junit.jupiter.api.Test;

import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/** Per-part import selection: an identifier merely ending in a package name (CaretPos.addr)
 *  must not keep the import, while real selectors must - at any position, case-exact. */
class ImportSelectionTest {

	@Test
	void identifierEndingInPackageNameDoesNotKeepTheImport() {
		assertFalse(Emitter.referencesPackage("r, _, _ := syscall.SyscallN(proc_OSGetCaretPos.addr(), uintptr(p))", "os"));
		assertFalse(Emitter.referencesPackage("y := proc_OSCaretPos.addr()", "os"));
		assertFalse(Emitter.referencesPackage("x := jrtNew.Run()", "jrt"), "a longer identifier is not the package");
		assertFalse(Emitter.referencesPackage("a := proc_OS.Addr()", "os"), "case must match exactly");
	}

	@Test
	void realSelectorsKeepTheImportAnywhere() {
		assertTrue(Emitter.referencesPackage("os.Getenv(name)", "os"), "start of the text");
		assertTrue(Emitter.referencesPackage("\tdata, _ := os.ReadFile(name)", "os"), "after whitespace");
		assertTrue(Emitter.referencesPackage("if e := fmt.Fprintln(os.Stderr, x); e != nil", "os"), "after an opening paren");
		assertTrue(Emitter.referencesPackage("return jrt.NewIllegalArgumentException(msg)", "jrt"));
		assertTrue(Emitter.referencesPackage("y := swt.DEFAULT", "swt"));
	}

	@Test
	void importSetSelectionDropsFalsePositivesOnly() {
		Set<String> imps = Emitter.referencedImports(Set.of("os", "fmt"),
				"r, _, _ := syscall.SyscallN(proc_OSGetCaretPos.addr(), uintptr(p))\n\tfmt.Println(r)\n");
		assertEquals(Set.of("fmt"), imps);
		Set<String> both = Emitter.referencedImports(Set.of("os", "fmt"),
				"\tif e := recover(); e != nil {\n\t\tfmt.Fprintln(os.Stderr, e)\n\t}\n");
		assertEquals(Set.of("os", "fmt"), both);
	}
}
