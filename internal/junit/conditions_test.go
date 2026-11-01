package junit

import (
	"runtime"
	"testing"
)

func TestSkipOnOS(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		systems []string
		want    string
	}{
		{"disabled current", false, []string{runtime.GOOS}, "condition"},
		{"disabled other", false, []string{"other"}, ""},
		{"enabled current", true, []string{runtime.GOOS}, ""},
		{"enabled other", true, []string{"other"}, "condition"},
		{"disabled multiple", false, []string{"other", runtime.GOOS}, "condition"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SkipOnOS(tc.enabled, "condition", tc.systems...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFirstSkip(t *testing.T) {
	if got := FirstSkip("", ""); got != "" {
		t.Fatalf("no conditions matched: got %q", got)
	}
	if got := FirstSkip("", "disabled", "later"); got != "disabled" {
		t.Fatalf("later condition matched: got %q, want disabled", got)
	}
}
