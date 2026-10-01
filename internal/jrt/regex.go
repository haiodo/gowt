package jrt

import (
	"regexp"
	"strings"
	"unicode/utf16"
)

// Pattern is java.util.regex.Pattern over Go's regexp: only the syntax the translated sources use
// (\p{Punct} is rewritten, \z is common to both).
type Pattern struct{ re *regexp.Regexp }

func PatternCompile(s string) *Pattern {
	return &Pattern{regexp.MustCompile(strings.ReplaceAll(s, `\p{Punct}`, `[:punct:]`))}
}

func (p *Pattern) Matcher(s string) *Matcher { return &Matcher{p: p, s: s} }

// Matcher is java.util.regex.Matcher; Start/End are UTF-16 offsets like Java's.
type Matcher struct {
	p     *Pattern
	s     string
	m     []int
	start int
}

func (m *Matcher) Find() bool {
	if m.start > len(m.s) {
		return false
	}
	m.m = m.p.re.FindStringSubmatchIndex(m.s[m.start:])
	if m.m == nil {
		return false
	}
	for i := range m.m {
		m.m[i] += m.start
	}
	m.start = m.m[1]
	if m.m[1] == m.m[0] {
		m.start++
	}
	return true
}

func (m *Matcher) units(byteOffset int) int32 { return int32(len(utf16.Encode([]rune(m.s[:byteOffset])))) }

func (m *Matcher) Start() int32 { return m.units(m.m[0]) }
func (m *Matcher) End() int32   { return m.units(m.m[1]) }

// ReplaceAll is String.replaceAll(regex, replacement).
func ReplaceAll(s, regex, repl string) string {
	return PatternCompile(regex).re.ReplaceAllString(s, strings.ReplaceAll(repl, "$", "$$"))
}
