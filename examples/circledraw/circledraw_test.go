package main

import (
	"slices"
	"testing"
)

func TestCircleHistory(t *testing.T) {
	empty := circleState{sel: -1}
	one := circleState{circles: []circle{{x: 10, y: 20, d: 30}}, sel: 0}
	resized := circleState{circles: []circle{{x: 10, y: 20, d: 50}}, sel: 0}
	cases := []struct {
		name       string
		past       []circleState
		future     []circleState
		current    circleState
		undo       bool
		want       circleState
		wantOK     bool
		wantPast   int
		wantFuture int
	}{
		{"undo add", []circleState{empty}, nil, one, true, empty, true, 0, 1},
		{"undo resize", []circleState{one}, nil, resized, true, one, true, 0, 1},
		{"redo add", nil, []circleState{one}, empty, false, one, true, 1, 0},
		{"redo resize", nil, []circleState{resized}, one, false, resized, true, 1, 0},
		{"undo with empty stack", nil, nil, one, true, one, false, 0, 0},
		{"redo with empty stack", nil, nil, one, false, one, false, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := circleHistory{past: tc.past, future: tc.future}
			var got circleState
			var ok bool
			if tc.undo {
				got, ok = h.undoOnce(tc.current)
			} else {
				got, ok = h.redoOnce(tc.current)
			}
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !slices.Equal(got.circles, tc.want.circles) || got.sel != tc.want.sel {
				t.Fatalf("state = %+v, want %+v", got, tc.want)
			}
			if len(h.past) != tc.wantPast || len(h.future) != tc.wantFuture {
				t.Fatalf("stacks = %d/%d, want %d/%d", len(h.past), len(h.future), tc.wantPast, tc.wantFuture)
			}
		})
	}
}

func TestCircleHistoryRecordClearsRedo(t *testing.T) {
	h := circleHistory{future: []circleState{{circles: []circle{{x: 1, y: 2, d: 30}}}}}
	h.record(circleState{sel: -1})
	if len(h.future) != 0 {
		t.Fatalf("redo stack has %d states after a record", len(h.future))
	}
}

func TestCircleHistoryCopiesState(t *testing.T) {
	h := circleHistory{}
	cur := circleState{circles: []circle{{x: 1, y: 2, d: 30}}, sel: 0}
	h.record(cur)
	cur.circles[0].d = 99
	got, ok := h.undoOnce(circleState{sel: -1})
	if !ok || got.circles[0].d != 30 {
		t.Fatalf("undo after mutating the recorded circle = %+v, want d 30", got)
	}
}

func TestTopmost(t *testing.T) {
	circles := []circle{{x: 50, y: 50, d: 30}, {x: 55, y: 50, d: 30}}
	cases := []struct {
		name string
		x, y int
		want int
	}{
		{"overlap returns the last drawn", 55, 50, 1},
		{"inside the first only", 37, 50, 0},
		{"outside every circle", 10, 10, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := topmost(circles, tc.x, tc.y); got != tc.want {
				t.Fatalf("topmost(%d, %d) = %d, want %d", tc.x, tc.y, got, tc.want)
			}
		})
	}
}
