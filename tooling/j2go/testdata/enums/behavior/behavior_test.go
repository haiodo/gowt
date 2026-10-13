package app

import "testing"

func TestEnums(t *testing.T) {
	u := NewEnumUse()
	if u.Score(LevelLOW) != 1 || u.Score(LevelHIGH) != 3 || u.Arrow(LevelMID) != "m" {
		t.Fatal("switch")
	}
	if u.Parse("HIGH") != LevelHIGH || !u.Same(LevelLOW, LevelLOW) || u.Mixed(LevelLOW, PlanetEARTH) || !u.ViaObject(LevelLOW, LevelLOW, nil) || u.Cmp(LevelHIGH, LevelLOW) != 2 {
		t.Fatal("ops")
	}
	if u.Show(LevelMID) != "MID1MID" || LevelHIGH.Next() != LevelLOW {
		t.Fatal(u.Show(LevelMID))
	}
	u.Fill()
	if PlanetHeaviest() != PlanetEARTH || PlanetEARTH.Label() != "earth" {
		t.Fatal("planet")
	}
}
