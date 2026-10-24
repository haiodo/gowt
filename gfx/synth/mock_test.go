package synth

import "testing"

// The real design screens of the tracker have 150 to 540 nodes and 40 to 150 texts; the generator
// must stay in that range or the benchmarks stop standing for them.
func TestMockDensity(t *testing.T) {
	m := NewMock(6, 1)
	if len(m.Screens) != 6 || m.Scene.Len() == 0 {
		t.Fatalf("screens %d, scene nodes %d", len(m.Screens), m.Scene.Len())
	}
	total := 0
	for _, s := range m.Screens[:3] {
		t.Logf("%-5s nodes %3d texts %3d paths %3d frames %3d", s.Kind, s.Nodes, s.Texts, s.Paths, s.Frames)
		if s.Nodes < 140 || s.Nodes > 600 || s.Texts < 40 || s.Texts > 170 || s.Paths < 40 {
			t.Errorf("%s screen out of range: %+v", s.Kind, *s)
		}
		total += s.Nodes
	}
	if m.Scene.Len() < total {
		t.Errorf("scene holds %d nodes", m.Scene.Len())
	}
}
