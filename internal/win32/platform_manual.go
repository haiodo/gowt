//go:build windows

// org.eclipse.swt.internal.Platform/Library: SWT loads its own JNI DLLs there, the natives here bind system DLLs lazily.
package win32

func PlatformExitIfNotLoadable() {}

func LibraryLoadLibrary(name string) {}
