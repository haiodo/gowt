// Hand-written stub for org.eclipse.swt.SWT (manual.txt), until the class itself is translated:
// at 5000+ lines it is out of v0 scope. Only the members the translated graphics classes
// reference are provided, named exactly as the translator would name them if SWT were
// translated (no-prefix static members, see the translation contract).
package swt

import "fmt"

const (
	ERROR_NULL_ARGUMENT    int32 = 4
	ERROR_INVALID_ARGUMENT int32 = 5
)

// Error mirrors SWT.error(int): it never returns normally.
func Error(code int32) {
	panic(fmt.Sprintf("SWT error %d", code))
}
