// Command app is the fixture of qatest's end-to-end test: controls with IDs and the QA handlers.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

type keyFunc func(*swt.Event)

func (f keyFunc) HandleEvent(e *swt.Event) { f(e) }

func main() {
	err := gowt.Run(func(app *gowt.App) {
		var events []string
		var w *gowt.Window
		var gv *view.View
		var popBtn *gowt.ButtonView
		var pop *gowt.Popover
		var resizes [][2]int
		scene := gfx.NewScene()
		red := gfx.NewNode(1)
		rec := gfx.NewRecorder()
		rec.FillRRect(gfx.R(0, 0, 400, 400), 0, gfx.RGB(255, 0, 0))
		// a black square of alpha 1/3 over the red: 255 * 2/3 = 170 once, 113 when composited twice
		veil := gfx.NewNode(2)
		vrec := gfx.NewRecorder()
		vrec.FillRRect(gfx.R(0, 0, 100, 100), 0, gfx.RGBA(0, 0, 0, 1.0/3))
		scene.Update(func(tx *gfx.Tx) {
			tx.Add(nil, red)
			tx.SetPicture(red, rec.Finish())
			tx.Add(nil, veil)
			tx.SetPicture(veil, vrec.Finish())
			tx.Move(veil, 100, 50)
		})
		vetoClose := false
		note := func(format string, a ...any) { events = append(events, fmt.Sprintf(format, a...)) }
		clicks := rx.New(0)
		off := rx.New(0)
		name := rx.New("")
		agree := rx.New(false)
		pick := rx.New(-1)
		report := rx.New("")
		text := rx.Compute(func() string { return fmt.Sprintf("clicks: %d", clicks.Get()) })
		w = app.Window("QA fixture",
			gowt.Label("", gowt.BindText(text), gowt.ID("count")),
			gowt.Button("Click", gowt.ID("click"), gowt.OnClick(func() { clicks.Set(clicks.Get() + 1) })),
			gowt.Button("Off", gowt.ID("off"), gowt.Disabled(), gowt.OnClick(func() { off.Set(off.Get() + 1) })),
			view.New(scene, func(v *view.View) {
				gv = v
				v.OnPointer(func(p *view.Pointer) {
					note("gfx kind=%d button=%d count=%d at %.0f,%.0f drag=%v node=%v mods=%v", p.Kind, p.Button, p.Count, p.Pos.X, p.Pos.Y, p.Drag, p.Node != nil, p.Mods.Shift)
					if p.Kind == view.PointerDown && p.Count == 2 {
						v.Edit(view.EditOptions{Rect: gfx.R(10, 10, 120, 24), Text: "title", OnCommit: func(s string) { note("commit %s", s) },
							OnCancel: func() { note("cancel edit") }, Opts: []gowt.Option{gowt.ID("gfx-edit")}})
					}
				})
				v.OnResize(func(w, h int) { resizes = append(resizes, [2]int{w, h}) })
			}, gowt.ID("gfx"), gowt.Size(300, 200)),
			gowt.Canvas(nil, gowt.ID("plain"), gowt.Size(120, 40), gowt.Init(func(c *gowt.CanvasView) {
				c.OnMouseDown(func(m gowt.Mouse) { note("plain down %d,%d b%d c%d shift=%v", m.X, m.Y, m.Button, m.Count, m.Shift) })
				c.OnMouseUp(func(m gowt.Mouse) { note("plain up %d,%d b%d", m.X, m.Y, m.Button) })
				c.OnMouseMove(func(m gowt.Mouse) { note("plain move %d,%d", m.X, m.Y) })
			})),
			gowt.Button("Menu host", gowt.ID("menuhost"), gowt.Init(func(b *gowt.ButtonView) {
				m := gowt.PopupMenu(b)
				m.OnShow(func() { note("menu show") })
				m.Item("Rename", func() { note("menu rename") })
				m.Item("Wrap", func() { note("menu wrap") }, gowt.Check())
				m.Submenu("More").Item("Deep", func() { note("menu deep") })
			})),
			gowt.Button("Pop", gowt.ID("pop"), gowt.Ref(&popBtn), gowt.OnClick(func() { pop.ShowAt(popBtn, gowt.PopoverBelow) })),
			gowt.Text(gowt.ID("name"), gowt.OnChange(func(s string) { name.Set(s) }),
				gowt.OnActivate(func() { note("activate name") }),
				gowt.OnFocus(func(on bool) { note("focus name %v", on) }),
				gowt.Init(func(t *gowt.TextView) {
					t.Unwrap().AddListener(swt.KeyDown, keyFunc(func(e *swt.Event) { note("keydown name %d", e.KeyCode) }))
				})),
			gowt.Button("Agree", gowt.Check(), gowt.ID("agree"), gowt.OnClick(func() { agree.Set(!agree.Get()) })),
			gowt.List([]string{"a", "b", "c"}, gowt.ID("list"), gowt.OnSelect(func(i int) { pick.Set(i) }),
				gowt.OnActivate(func(i int) { note("activate list %d", i) })),
			gowt.Table(gowt.ID("tbl"), gowt.OnActivate(func(i int) { note("activate table %d", i) }),
				gowt.Init(func(t *gowt.TableView) {
					t.Column("Name", 80)
					t.Row("alpha")
					t.Row("beta")
				})),
			gowt.Tree(gowt.ID("lazy"), gowt.OnActivate(func(n *gowt.Node) { note("activate tree %s", n.Text()) }),
				gowt.Init(func(t *gowt.TreeView) {
					t.Node("remote").Node("loading")
					t.OnExpand(func(n *gowt.Node) {
						note("expand %s", n.Text())
						for _, c := range n.Children() {
							c.Remove()
						}
						n.Node("disk")
						n.Node("net")
					})
					t.OnCollapse(func(n *gowt.Node) { note("collapse %s", n.Text()) })
				})),
			gowt.Button("Delete", gowt.ID("delete"), gowt.OnClick(func() {
				if w.Confirm("Delete", "Delete the file?") {
					note("deleted")
				}
			})),
			gowt.Button("Open", gowt.ID("open"), gowt.OnClick(func() {
				for _, p := range w.FileDialog(gowt.FileDialog{Title: "Open", Multi: true, Filters: []gowt.FileFilter{{Name: "Text", Pattern: "*.txt"}}}) {
					note("opened %s", p)
				}
			})),
			gowt.Button("Popup", gowt.ID("popup"), gowt.OnClick(func() {
				p := app.Window("Popup", gowt.Label("popup"),
					gowt.Text(gowt.ID("pname"), gowt.OnFocus(func(on bool) { note("focus pname %v", on) })))
				p.OnClose(func() bool { note("close popup"); return !vetoClose })
				p.Show()
			})),
			gowt.Scroll(gowt.ID("feed"), gowt.Size(300, 120), gowt.Init(func(s *gowt.ScrollView) {
				for i := range 60 {
					s.Add(gowt.Label(fmt.Sprintf("row %d", i), gowt.ID(fmt.Sprintf("row-%d", i))))
				}
				s.OnScroll(func(x, y int) { note("scroll %d", y) })
			})),
			gowt.Button("Load", gowt.ID("load"), gowt.OnClick(func() {
				app.Job("Download report", func(ctx context.Context, p *gowt.JobProgress) error {
					p.Total(3)
					for range 3 {
						time.Sleep(500 * time.Millisecond)
						p.Step(1)
					}
					return nil
				}).Then(func(error) { report.Set("report ready") })
			})),
			gowt.Label("", gowt.BindText(report), gowt.ID("report")),
			gowt.Button("Twin"), gowt.Button("Twin"))
		pop = w.Popover(gowt.ID("popover"))
		pop.Add(gowt.Label("popover body"), gowt.Button("Pop OK", gowt.ID("pop-ok"), gowt.OnClick(func() { note("pop ok") })))
		pop.OnClose(func() { note("pop closed") })
		w.OnKey("Cmd+K", func() { note("key mod+k") })
		w.MenuBar().Submenu("File").Item("Run", func() { note("menu run") }, gowt.Shortcut("Cmd+R"))
		w.Show()
		// Perpetual work: listed by Jobs, never waited for by settle.
		app.Job("poll", func(ctx context.Context, p *gowt.JobProgress) error {
			<-ctx.Done()
			return nil
		}, gowt.BackgroundJob())
		qa.Handle("reset", func(qa.Args) error {
			clicks.Set(0)
			off.Set(0)
			report.Set("")
			events, vetoClose = nil, false
			pop.Close()
			gv.Compositor().SetCamera(gfx.Camera{Zoom: 1})
			return nil
		})
		qa.Handle("camera-set", func(a qa.Args) error {
			gv.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: float64(a.Int("x")), Y: float64(a.Int("y"))}, Zoom: 1})
			return nil
		})
		qa.HandleValue("camera", func(qa.Args) (any, error) {
			c := gv.Compositor().Camera()
			return map[string]float64{"x": c.Origin.X, "y": c.Origin.Y, "zoom": c.Zoom}, nil
		})
		qa.HandleValue("gfx-size", func(qa.Args) (any, error) {
			w, h := gv.Canvas().Size()
			return map[string]any{"w": w, "h": h, "resizes": resizes}, nil
		})
		qa.HandleValue("events", func(qa.Args) (any, error) { return events, nil })
		qa.Handle("veto", func(a qa.Args) error { vetoClose = a.Bool("on"); return nil })
		qa.Handle("add", func(a qa.Args) error { clicks.Set(clicks.Get() + a.Int("n")); return nil })
		qa.HandleValue("get", func(qa.Args) (any, error) { return clicks.Get(), nil })
		qa.HandleValue("form", func(qa.Args) (any, error) {
			return map[string]any{"off": off.Get(), "name": name.Get(), "agree": agree.Get(), "pick": pick.Get()}, nil
		})
		qa.Handle("fail", func(qa.Args) error { return fmt.Errorf("asked to fail") })
		qa.Handle("panic", func(qa.Args) error { panic("asked to panic") })
	})
	if err != nil {
		log.Fatal(err)
	}
}
