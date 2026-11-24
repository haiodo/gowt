package main

import "testing"

// solved is the layout a finished puzzle shows.
var solved = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 0}

func layout(v ...int) [boardSize * boardSize]int {
	var cells [boardSize * boardSize]int
	copy(cells[:], v)
	return cells
}

func TestSolvable(t *testing.T) {
	cases := []struct {
		name  string
		cells [boardSize * boardSize]int
		want  bool
	}{
		{"solved", layout(solved...), true},
		{"14 and 15 swapped", layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 14, 0), false},
		{"hole first, tiles ordered", layout(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15), false},
		{"column slide from solved", layout(1, 2, 3, 0, 5, 6, 7, 4, 9, 10, 11, 8, 13, 14, 15, 12), true},
	}
	for _, c := range cases {
		if got := solvable(c.cells); got != c.want {
			t.Errorf("%s: solvable = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMove(t *testing.T) {
	cases := []struct {
		name  string
		start [boardSize * boardSize]int
		cell  int
		after [boardSize * boardSize]int
		want  bool
	}{
		{
			"row run slides right",
			layout(solved...), 12,
			layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 13, 14, 15), true,
		},
		{
			"column run slides down",
			layout(solved...), 3,
			layout(1, 2, 3, 0, 5, 6, 7, 4, 9, 10, 11, 8, 13, 14, 15, 12), true,
		},
		{
			"one tile",
			layout(solved...), 14,
			layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0, 15), true,
		},
		{
			"another row and column",
			layout(solved...), 5,
			layout(solved...), false,
		},
		{
			"click on the hole",
			layout(solved...), 15,
			layout(solved...), false,
		},
	}
	for _, c := range cases {
		b := &board{cells: c.start}
		if got := b.move(c.cell); got != c.want {
			t.Errorf("%s: move = %v, want %v", c.name, got, c.want)
		}
		if b.cells != c.after {
			t.Errorf("%s: board = %v, want %v", c.name, b.cells, c.after)
		}
	}
}

func TestTilesLeft(t *testing.T) {
	cases := []struct {
		name  string
		cells [boardSize * boardSize]int
		want  int
	}{
		{"solved", layout(solved...), 0},
		{"hole moved", layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0, 15), 1},
		{"two tiles swapped", layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 14, 0), 2},
	}
	for _, c := range cases {
		b := &board{cells: c.cells}
		if got := b.tilesLeft(); got != c.want {
			t.Errorf("%s: tilesLeft = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestSolved(t *testing.T) {
	cases := []struct {
		name  string
		cells [boardSize * boardSize]int
		want  bool
	}{
		{"solved", layout(solved...), true},
		{"hole moved", layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0, 15), false},
		{"two tiles swapped", layout(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 14, 0), false},
	}
	for _, c := range cases {
		b := &board{cells: c.cells}
		if got := b.solved(); got != c.want {
			t.Errorf("%s: solved = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestShuffle(t *testing.T) {
	b := &board{}
	for range 200 {
		b.shuffle()
		var seen [boardSize * boardSize]bool
		for _, v := range b.cells {
			if v < 0 || v >= len(seen) || seen[v] {
				t.Fatalf("bad tile %d in %v", v, b.cells)
			}
			seen[v] = true
		}
		if !solvable(b.cells) {
			t.Fatalf("unsolvable layout %v", b.cells)
		}
	}
}

func TestCellAtBoardEdges(t *testing.T) {
	x, y, size := geometry(200, 200)
	for _, p := range [][2]int{{x - 1, y}, {x, y - 1}, {x + size*boardSize, y}, {x, y + size*boardSize}} {
		if got := cellAt(200, 200, p[0], p[1]); got != -1 {
			t.Errorf("outside %v: %d", p, got)
		}
	}
	if got := cellAt(200, 200, x, y); got != 0 {
		t.Errorf("top left: %d", got)
	}
	if got := cellAt(200, 200, x+size*boardSize-1, y+size*boardSize-1); got != 15 {
		t.Errorf("bottom right: %d", got)
	}
}
