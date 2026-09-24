package jrt

import "testing"

func TestPropertiesAndMessageFormat(t *testing.T) {
	p := parseProperties([]byte("# comment\nwindow.title = SWT Controls\nWrap\t= Jack and \\\n    Jill\nesc=a\\tb\\u0041\nkey:value\n"))
	want := map[string]string{"window.title": "SWT Controls", "Wrap": "Jack and Jill", "esc": "a\tbA", "key": "value"}
	for k, v := range want {
		if p[k] != v {
			t.Errorf("%s = %q, want %q", k, p[k], v)
		}
	}
	if got := MessageFormatFormat("{0}   e.g. {1}, it''s {2}", []any{"int", 5}); got != "int   e.g. 5, it's {2}" {
		t.Errorf("MessageFormatFormat = %q", got)
	}
	if got := IndexFrom("a|b|c", "|", 2); got != 3 {
		t.Errorf("IndexFrom = %d", got)
	}
}
