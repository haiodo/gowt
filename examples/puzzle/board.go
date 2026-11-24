package main

import (
	"math/rand/v2"
	"slices"
)

// boardSize is the grid side: 15 tiles and one hole.
const boardSize = 4

// board keeps the tiles in row-major order; 0 is the hole.
type board struct {
	cells [boardSize * boardSize]int
}

func newBoard() *board {
	b := &board{}
	b.shuffle()
	return b
}

// shuffle deals a random solvable layout.
func (b *board) shuffle() {
	for i := range b.cells {
		b.cells[i] = i + 1
	}
	b.cells[len(b.cells)-1] = 0
	for {
		rand.Shuffle(len(b.cells), func(i, j int) { b.cells[i], b.cells[j] = b.cells[j], b.cells[i] })
		if solvable(b.cells) {
			return
		}
	}
}

// solvable reports whether the layout can reach the solved one. On a board of even width the
// inversions and the hole row counted from the bottom must have the same parity.
func solvable(cells [boardSize * boardSize]int) bool {
	inversions, hole := 0, 0
	for i, v := range cells {
		if v == 0 {
			hole = i
			continue
		}
		for _, u := range cells[i+1:] {
			if u != 0 && u < v {
				inversions++
			}
		}
	}
	return inversions%2 == (boardSize-1-hole/boardSize)%2
}

func (b *board) hole() int { return slices.Index(b.cells[:], 0) }

// move slides the whole run between cell and the hole one step towards the hole; a cell in
// another row and column moves nothing.
func (b *board) move(cell int) bool {
	if b.cells[cell] == 0 {
		return false
	}
	hole := b.hole()
	cr, cc := cell/boardSize, cell%boardSize
	hr, hc := hole/boardSize, hole%boardSize
	switch {
	case cr == hr:
		step := 1
		if cc > hc {
			step = -1
		}
		for c := hole; c != cell; c -= step {
			b.cells[c] = b.cells[c-step]
		}
	case cc == hc:
		step := boardSize
		if cr > hr {
			step = -boardSize
		}
		for c := hole; c != cell; c -= step {
			b.cells[c] = b.cells[c-step]
		}
	default:
		return false
	}
	b.cells[cell] = 0
	return true
}

func (b *board) tilesLeft() int {
	n := 0
	for i, v := range b.cells {
		if v != 0 && v != i+1 {
			n++
		}
	}
	return n
}

func (b *board) solved() bool { return b.tilesLeft() == 0 }
