// Hand-written no-ops for org.eclipse.swt.internal.Platform/Library, which C.java's static
// block calls before loading SWT's own bundled native library - no Go equivalent, see
// tooling/j2go/manual.txt. The frameworks OS.java's natives need are loaded by libs_manual.go.
package cocoa

import "runtime"

func PlatformExitIfNotLoadable() {}

func LibraryLoadLibrary(name string) {}

// JavaOsArch mirrors System.getProperty("os.arch")'s JVM naming (OS.java's IS_X86_64), which
// doesn't match Go's own runtime.GOARCH spelling.
func JavaOsArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}
