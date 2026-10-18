package swt

import (
	"embed"

	"github.com/haiodo/gowt/internal/jrt"
)

// SWTMessages.properties is copied from SWT common (org/eclipse/swt/internal, EPL-2.0).
//
//go:embed SWTMessages.properties
var swtMessagesFS embed.FS

func init() { jrt.RegisterResources(swtMessagesFS) }

var swtMessages = jrt.ResourceBundleGetBundle("SWTMessages")

// Compatibility.getMessage: the English SWTMessages bundle, the key itself when it has no entry.
func CompatibilityGetMessage(key string, args ...[]any) (msg string) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, ok := r.(*jrt.MissingResourceException); !ok {
			panic(r)
		}
		msg = key
	}()
	msg = swtMessages.GetString(key)
	if len(args) > 0 {
		msg = jrt.MessageFormatFormat(msg, args[0])
	}
	return msg
}
