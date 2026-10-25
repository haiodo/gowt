package gowt

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// A chat feed row as a chat client builds it: avatar Canvas, header Label, wrapped body. Numbers have
// four digits, so that every row of the same i%4 has the same natural width.
func feedRow(i int, kind string) Option {
	text := fmt.Sprintf("Message %04d: %s", i, strings.Repeat("some wrapped text of a chat message ", 1+i%4))
	var body Option
	switch kind {
	case "label":
		body = Label(text, Wrap(), GrowX())
	case "none":
		body = Label("", Size(1, 1))
	default:
		body = StyledText(ReadOnly(), Wrap(), GrowX(), Init(func(ed *StyledTextView) { ed.SetText(text) }))
	}
	return HBox(GrowX(), Gap(10), Padding(0), Indent(0, 16),
		Canvas(func(g *GC) {
			g.SetFill(RGB{R: 0x70, G: 0x70, B: 0xC0})
			g.FillOval(0, 0, 36, 36)
		}, Size(36, 36), Place(AlignStart, AlignStart)),
		VBox(GrowX(), Gap(6), Padding(0),
			Label(fmt.Sprintf("Author %04d  12:%02d", i, i%60)), body))
}

func feedRows(from, n int, kind string) []Option {
	rows := make([]Option, n)
	for i := range rows {
		rows[i] = feedRow(from+i, kind)
	}
	return rows
}

// feedWindow shows a 900x700 window with a ScrollView of n rows, laid out.
func feedWindow(app *App, n int, kind string) (*Window, *ScrollView) {
	var sv *ScrollView
	w := app.Window("feed")
	w.SetSize(900, 700)
	w.Add(Scroll(Grow(), Ref(&sv)))
	sv.Add(feedRows(0, n, kind)...)
	w.Show()
	app.Flush()
	return w, sv
}

// One row created in the content of a shown ScrollView, without the layout Add does after it.
func BenchmarkFeedRowCreate(b *testing.B) {
	for _, kind := range []string{"styled", "label"} {
		b.Run(kind, func(b *testing.B) {
			guiBench(b, func(app *App) {
				w, sv := feedWindow(app, 0, kind)
				b.ResetTimer()
				for i := range b.N {
					add(sv, []Option{feedRow(i, kind)})
				}
				b.StopTimer()
				w.Close()
			})
		})
	}
}

// ScrollView.Add of 10 rows to a feed of 300 rows.
func BenchmarkFeedAdd10(b *testing.B) {
	for _, n := range []int{60, 300} {
		b.Run(fmt.Sprint("rows=", n), func(b *testing.B) {
			guiBench(b, func(app *App) {
				w, sv := feedWindow(app, n, "styled")
				b.ResetTimer()
				for i := range b.N {
					sv.Add(feedRows(n+i*10, 10, "styled")...)
					b.StopTimer()
					kids := sv.Content().Unwrap().GetChildren()
					for _, k := range kids[n:] {
						k.Dispose()
					}
					b.StartTimer()
				}
				b.StopTimer()
				w.Close()
			})
		})
	}
}

// The relayout of the whole content: Content().Layout() and the ComputeSize of Fit.
func BenchmarkFeedLayout(b *testing.B) {
	for _, n := range []int{60, 300} {
		b.Run(fmt.Sprint("rows=", n), func(b *testing.B) {
			guiBench(b, func(app *App) {
				w, sv := feedWindow(app, n, "styled")
				c := sv.Content().Unwrap()
				b.ResetTimer()
				for range b.N {
					c.Layout()
				}
				b.StopTimer()
				w.Close()
			})
		})
	}
}

// A scroll frame as the client measures it: SetOrigin 40 points, then the event loop paints and fires
// a 1 ms timer; a frame is the interval between timer calls. The first 50 frames go up from the bottom
// over rows not painted before (fresh), the rest move back and forth over the same 2000 points (painted).
// Every frame moves the content, with 60 rows too.
func BenchmarkFeedScrollFrame(b *testing.B) {
	for _, kind := range []string{"styled", "none"} {
		for _, n := range []int{60, 300} {
			b.Run(fmt.Sprintf("%s/rows=%d", kind, n), func(b *testing.B) {
				guiBench(b, func(app *App) {
					w, sv := feedWindow(app, n, kind)
					top := int(sv.Unwrap().GetMinHeight()) - 700
					y, dy := top, -40
					var fresh, painted []time.Duration
					var last time.Time
					var run func()
					step := 0
					run = func() {
						now := time.Now()
						if !last.IsZero() {
							if step <= 50 {
								fresh = append(fresh, now.Sub(last))
							} else {
								painted = append(painted, now.Sub(last))
							}
						}
						last = now
						if step == b.N {
							b.StopTimer()
							p50 := func(d []time.Duration) float64 {
								if len(d) == 0 {
									return 0
								}
								slices.Sort(d)
								return float64(d[len(d)/2].Microseconds())
							}
							b.ReportMetric(p50(fresh), "fresh-p50-us")
							b.ReportMetric(p50(painted), "painted-p50-us")
							w.Close()
							return
						}
						step++
						if step > 50 && (y+dy < top-2000 || y+dy > top) {
							dy = -dy
						}
						y += dy
						sv.SetOrigin(0, y)
						app.After(time.Millisecond, run)
					}
					app.After(200*time.Millisecond, func() { b.ResetTimer(); run() })
				})
			})
		}
	}
}

// bounds lists x, y, width, height of c and of everything inside it, depth first.
func bounds(c *swt.Control) [][4]int32 {
	b := c.GetBounds()
	out := [][4]int32{{b.X, b.Y, b.Width, b.Height}}
	if comp, ok := c.Impl().(interface{ GetChildren() []*swt.Control }); ok {
		for _, k := range comp.GetChildren() {
			out = append(out, bounds(k)...)
		}
	}
	return out
}

// ScrollView.Add lays out only the new rows, Panel.Add of a column inside it only the column; after
// the ScrollView.Add the result must equal a full flushed layout of the content.
func TestFeedAddSameAsFullLayout(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			var col *Panel
			w, sv := feedWindow(app, 30, "styled")
			sv.Add(VBox(GrowX(), Ref(&col)))
			for i := range 4 {
				col.Add(feedRows(200+i*3, 3, "styled")...)
				sv.Add(feedRows(100+i*7, 7, "label")...)
			}
			app.Flush()
			c := sv.Content().Unwrap()
			got := bounds(&c.Control)
			gotW, gotH := sv.Unwrap().GetMinWidth(), sv.Unwrap().GetMinHeight()
			c.LayoutOverload2(true, true)
			sv.Fit()
			want := bounds(&c.Control)
			if w, h := sv.Unwrap().GetMinWidth(), sv.Unwrap().GetMinHeight(); w != gotW || h != gotH {
				t.Errorf("min size after Add %dx%d, after a full layout %dx%d", gotW, gotH, w, h)
			}
			if !slices.Equal(got, want) {
				bad := 0
				for i := range min(len(got), len(want)) {
					if got[i] != want[i] && bad < 5 {
						bad++
						t.Errorf("control %d of %d: %v after Add, %v after a full layout", i, len(want), got[i], want[i])
					}
				}
				if len(got) != len(want) {
					t.Errorf("%d controls after Add, %d after a full layout", len(got), len(want))
				}
			}
			w.Close()
		})
		if err != nil {
			t.Error(err)
		}
	})
}

// A setter that grows a row lays out on the next turn; an Add in the same turn must measure that row
// again, so the result equals a full layout before the turn ends.
func TestFeedAddAfterSetText(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			var head *LabelView
			w, sv := feedWindow(app, 20, "label")
			sv.Add(HBox(GrowX(), Label("last row", Ref(&head))))
			app.Flush()
			head.SetText("the last row\ngrew\nto four\nlines")
			sv.Add(feedRows(500, 3, "label")...)
			c := sv.Content().Unwrap()
			got := bounds(&c.Control)
			gotH := sv.Unwrap().GetMinHeight()
			c.LayoutOverload2(true, true)
			sv.Fit()
			if h := sv.Unwrap().GetMinHeight(); h != gotH {
				t.Errorf("min height after SetText and Add %d, after a full layout %d", gotH, h)
			}
			if want := bounds(&c.Control); !slices.Equal(got, want) {
				t.Error("bounds after SetText and Add differ from a full layout")
			}
			w.Close()
		})
		if err != nil {
			t.Error(err)
		}
	})
}
