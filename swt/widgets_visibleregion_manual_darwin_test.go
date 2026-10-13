package swt

import "testing"

func TestVisibleRegionsIdle(t *testing.T) {
	saved := visibleRgnCount
	t.Cleanup(func() { visibleRgnCount = saved })
	visibleRgnCount = 0
	d := &Display{}
	if !visibleRegionsIdle(d) {
		t.Fatal("no regions and no contexts must be idle")
	}
	visibleRgnCount = 1
	if visibleRegionsIdle(d) {
		t.Fatal("a cached region must not be idle")
	}
	visibleRgnCount = 0
	d.contexts = make([]*GCData, 1)
	if visibleRegionsIdle(d) {
		t.Fatal("an open GC context must not be idle")
	}
}
