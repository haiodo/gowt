//go:build windows

package win32

// The generated OsVersionIS_* initializers run before init() sets OsVersionWIN32_BUILD, so they
// all stay false; this init (file order: after version_osversion.go) recomputes them.
// IS_WIN11_21H2 stays false on purpose: MenuItem's Win11 branch carries an untranslated getInteger call that panics.
func init() {
	OsVersionIS_WIN10_1607 = OsVersionWIN32_BUILD >= OsVersionWIN10_1607
	OsVersionIS_WIN10_1809 = OsVersionWIN32_BUILD >= 17763
	OsVersionIS_WIN10_2004 = OsVersionWIN32_BUILD >= 19041
}
