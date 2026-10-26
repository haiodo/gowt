//go:build gowtqa

package swt

import "os"

// qaClipboardPath is the file of the virtual clipboard in a QA run, "" in any other.
func qaClipboardPath() string {
	if os.Getenv("GOWT_QA_SOCKET") == "" {
		return ""
	}
	return qaClipboardDir(os.Getenv("GOWT_QA_DATA"))
}
