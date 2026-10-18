package swt

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// SWTMessages.properties is copied from SWT common (org/eclipse/swt/internal, EPL-2.0).
//
//go:embed SWTMessages.properties
var swtMessagesSource string

var swtMessages = sync.OnceValue(func() map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(swtMessagesSource, "\n") {
		line = strings.TrimLeft(strings.TrimRight(line, "\r"), " \t")
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(key)] = strings.ReplaceAll(value, `\t`, "\t")
	}
	return m
})

// Compatibility.getMessage: the English SWTMessages bundle, the key itself when it has no entry.
// Arguments fill {n} and {n,...} like MessageFormat, without its number and date formatting.
func CompatibilityGetMessage(key string, args ...[]any) string {
	msg, ok := swtMessages()[key]
	if !ok {
		return key
	}
	if len(args) == 0 {
		return msg
	}
	var b strings.Builder
	for {
		open := strings.IndexByte(msg, '{')
		if open < 0 {
			break
		}
		end := strings.IndexByte(msg[open:], '}')
		if end < 0 {
			break
		}
		spec := msg[open+1 : open+end]
		index, _, _ := strings.Cut(spec, ",")
		b.WriteString(msg[:open])
		n, err := strconv.Atoi(index)
		if err != nil || n < 0 || n >= len(args[0]) {
			b.WriteString(msg[open : open+end+1])
		} else if c, ok := args[0][n].(uint16); ok {
			// a Java char arrives as uint16 and MessageFormat prints it as a character
			b.WriteRune(rune(c))
		} else {
			fmt.Fprint(&b, args[0][n])
		}
		msg = msg[open+end+1:]
	}
	b.WriteString(msg)
	return b.String()
}
