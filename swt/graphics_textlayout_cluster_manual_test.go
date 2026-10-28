package swt

import (
	"slices"
	"testing"

	"github.com/haiodo/gowt/internal/jrt"
)

func TestClusterJoined(t *testing.T) {
	joined := func(s string) []int32 {
		var out []int32
		for i := int32(0); i <= jrt.StringLength(s); i++ {
			if clusterJoined(s, i) {
				out = append(out, i)
			}
		}
		return out
	}
	for _, c := range []struct {
		s    string
		want []int32
	}{
		{"x" + tZWJ + "z", []int32{3, 4, 6, 7}}, // the stops inside the family
		{"a\U0001F44D\U0001F3FDb", []int32{3}},  // before the skin tone
		{"❤️!", []int32{1}},                     // before VS16
		{"ab" + tSmile + tFlag, nil},
	} {
		if got := joined(c.s); !slices.Equal(got, c.want) {
			t.Errorf("%q: joined at %v, want %v", c.s, got, c.want)
		}
	}
}
