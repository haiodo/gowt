package swt

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// The recycle sites are matched by function name; a renamed method would leak its callbacks again.
func TestCallbackRecycleSitesExist(t *testing.T) {
	var names []string
	for _, f := range []any{(*Display).GetMonitors, (*Device).GetFontList, (*FontData).SetLocale,
		(*TextLayout).ShapeGcHdcRun, (*Tree).SortHParentAll} {
		names = append(names, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
	}
	for _, site := range callbackRecycleSites {
		found := false
		for _, n := range names {
			found = found || strings.HasSuffix(n, site)
		}
		if !found {
			t.Errorf("no method %s in %q", site, names)
		}
	}
}
