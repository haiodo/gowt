package gowt

import (
	"os"
	"testing"
	"time"
)

type scrollCall struct {
	x, y int
	at   time.Time
}

func TestScrollOnScroll(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var calls []scrollCall
	var wheelSent bool
	var wheelFrom int
	var burstMoved []time.Time
	check := func(phase string, want int) {
		if len(calls) != want {
			t.Errorf("%s: %d calls %v, want %d", phase, len(calls), calls, want)
		}
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			var s *ScrollView
			w := app.Window("scroll")
			w.SetSize(240, 160)
			w.Add(Scroll(Grow(), Ref(&s), Label("tall", Size(600, 3000))))
			s.OnScroll(func(x, y int) { calls = append(calls, scrollCall{x, y, time.Now()}) })
			w.Show()
			step := 150 * time.Millisecond
			app.After(step, func() {
				check("before any scroll", 0)
				s.SetOrigin(0, 50) // four moves in one turn: one call, with the last origin
				s.SetOrigin(10, 80)
				s.SetOrigin(20, 100)
				s.SetOrigin(30, 120)
			})
			app.After(2*step, func() {
				check("SetOrigin four times", 1)
				if len(calls) == 1 && (calls[0].x != 30 || calls[0].y != 120) {
					t.Errorf("SetOrigin: got %d,%d, want 30,120", calls[0].x, calls[0].y)
				}
				s.SetOrigin(30, 120) // no move, no call
				// A move every 4 ms, faster than a frame: at most one call per frame. Chained:
				// Windows runs timers that expire in one tick in any order, so five Afters could end
				// at 130. burstMoved records when the moves reached the loop, so a runner too slow for
				// 4 ms does not fail a coalescing it never had to do.
				var move func(i int)
				move = func(i int) {
					s.SetOrigin(30, 120+10*i)
					burstMoved = append(burstMoved, time.Now())
					if i < 5 {
						app.After(4*time.Millisecond, func() { move(i + 1) })
					}
				}
				app.After(4*time.Millisecond, func() { move(1) })
			})
			app.After(3*step, func() {
				burst := calls[1:]
				if len(burst) == 0 {
					t.Errorf("five moves 4 ms apart: no call")
				} else if len(burstMoved) > 1 {
					if span := burstMoved[len(burstMoved)-1].Sub(burstMoved[0]); len(burst) >= 5 && span < 4*scrollFrame {
						t.Errorf("five moves %v apart: %d calls %v, want fewer than 5", span, len(burst), burst)
					}
				}
				for i := 1; i < len(calls); i++ {
					if gap := calls[i].at.Sub(calls[i-1].at); gap < scrollFrame-2*time.Millisecond {
						t.Errorf("calls %d and %d are %v apart, less than a frame", i-1, i, gap)
					}
				}
				if last := calls[len(calls)-1]; last.y != 170 {
					t.Errorf("last call after the burst reports y %d, want 170", last.y)
				}
				wheelFrom = len(calls)
				wheelSent = postWheelDown(s)
			})
			app.After(5*step, func() {
				x, y := s.Origin()
				switch {
				case !wheelSent:
					t.Log("no wheel event can be sent on this platform; the wheel is not checked")
				case len(calls) == wheelFrom:
					t.Errorf("wheel: no call; origin %d,%d", x, y)
				case calls[len(calls)-1].y <= 170 || calls[len(calls)-1].y != y:
					t.Errorf("wheel: last call %v, origin %d,%d, want a y past 170 equal to the origin", calls[len(calls)-1], x, y)
				}
				t.Logf("calls %v", calls)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
