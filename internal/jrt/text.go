package jrt

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// ResourceBundle is java.util.ResourceBundle backed by a <name>.properties file in the registered
// resource FS (RegisterResources). No locale fallback chain: only the base bundle is read.
type ResourceBundle struct {
	values map[string]string
}

// MissingResourceException is java.util.MissingResourceException.
type MissingResourceException struct{ Key string }

func (e *MissingResourceException) Error() string { return "MissingResourceException: " + e.Key }

// NumberFormatException is java.lang.NumberFormatException.
type NumberFormatException struct{ Input string }

func (e *NumberFormatException) Error() string {
	return "NumberFormatException: For input string: \"" + e.Input + "\""
}

// MessageFormat only names the static MessageFormatFormat (java.text.MessageFormat.format).
type MessageFormat struct{}

func ResourceBundleGetBundle(name string) *ResourceBundle {
	if resourceFS == nil {
		panic(&MissingResourceException{Key: name})
	}
	data, err := fs.ReadFile(resourceFS, name+".properties")
	if err != nil {
		panic(&MissingResourceException{Key: name})
	}
	return &ResourceBundle{values: parseProperties(data)}
}

func (b *ResourceBundle) GetString(key string) string {
	v, ok := b.values[key]
	if !ok {
		panic(&MissingResourceException{Key: key})
	}
	return v
}

// parseProperties reads the java.util.Properties text format: comments, key=value/key:value/
// key value, backslash line continuation and escapes (\n, \t, \uXXXX).
func parseProperties(data []byte) map[string]string {
	values := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	logical := ""
	for sc.Scan() {
		line := strings.TrimLeft(sc.Text(), " \t\f")
		if logical == "" && (line == "" || line[0] == '#' || line[0] == '!') {
			continue
		}
		if trailingBackslashes(line)%2 == 1 {
			logical += line[:len(line)-1]
			continue
		}
		logical += line
		key, value := splitProperty(logical)
		values[unescapeProperty(key)] = unescapeProperty(value)
		logical = ""
	}
	return values
}

func trailingBackslashes(s string) int {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n
}

func splitProperty(line string) (string, string) {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '=', ':', ' ', '\t', '\f':
			rest := strings.TrimLeft(line[i:], " \t\f")
			if rest != "" && (rest[0] == '=' || rest[0] == ':') {
				rest = strings.TrimLeft(rest[1:], " \t\f")
			}
			return line[:i], rest
		}
	}
	return line, ""
}

func unescapeProperty(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'f':
			b.WriteByte('\f')
		case 'u':
			if r, err := strconv.ParseUint(s[i+1:min(i+5, len(s))], 16, 32); err == nil && i+4 < len(s) {
				b.WriteRune(rune(r))
				i += 4
				continue
			}
			b.WriteByte('u')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// MessageFormatFormat is MessageFormat.format(pattern, args): {n} is replaced by fmt.Sprint(args[n])
// and '' is a literal quote. Not supported: {n,number,...} format types, quoted literal sections.
func MessageFormatFormat(pattern string, args []any) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\'' && i+1 < len(pattern) && pattern[i+1] == '\'' {
			b.WriteByte('\'')
			i++
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		if c != '{' || end < 0 {
			b.WriteByte(c)
			continue
		}
		n, err := strconv.Atoi(pattern[i+1 : i+end])
		if err != nil || n < 0 || n >= len(args) {
			b.WriteString(pattern[i : i+end+1])
		} else {
			b.WriteString(fmt.Sprint(args[n]))
		}
		i += end
	}
	return b.String()
}

// ParseInt is Integer.parseInt: a NumberFormatException panic on bad input or int overflow.
func ParseInt(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		panic(&NumberFormatException{Input: s})
	}
	return int32(n)
}

// IndexFrom is String.indexOf(str, fromIndex): byte offsets, like the rest of this port's strings.
func IndexFrom(s, needle string, from int32) int32 {
	if from < 0 {
		from = 0
	}
	if int(from) > len(s) {
		return -1
	}
	i := strings.Index(s[from:], needle)
	if i < 0 {
		return -1
	}
	return from + int32(i)
}

// ClassGetResourceAsStream is Class.getResourceAsStream: the class is ignored, names resolve
// against the one registered resource FS.
func ClassGetResourceAsStream(class any, name string) InputStream {
	return GetResourceAsStream(name)
}
