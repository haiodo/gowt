//go:build gowtqa

package swt

import "os"

// qaNoActivate: a QA run (package qa) must not take the focus from the user's work.
func qaNoActivate() bool { return os.Getenv("GOWT_QA_SOCKET") != "" }
