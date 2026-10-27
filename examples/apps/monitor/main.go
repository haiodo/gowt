// Command monitor shows the load of the machine it runs on: CPU, per-core load, memory, network and
// the busiest processes. The history charts are gfx scenes (scrolling tape, overlay sprites, hover
// crosshair), the cards, labels and the process table are native gowt widgets.
package main

import (
	"cmp"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/examples/apps/monitor/sampler"
	"github.com/haiodo/gowt/gfx"
)

var windows = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}

// segment is one slice of the memory bar and its legend entry.
type segment struct {
	name  string
	color func(p *palette) gfx.Color
	get   func(m sampler.Memory) uint64
}

func memSegments() (used []segment, rest []segment) {
	app := segment{"App", func(p *palette) gfx.Color { return p.accent }, func(m sampler.Memory) uint64 { return m.App }}
	cached := segment{"Cached files", func(p *palette) gfx.Color { return mix(p.card, p.accent, 0.32) }, func(m sampler.Memory) uint64 { return m.Cached }}
	free := segment{"Free", func(p *palette) gfx.Color { return p.track() }, func(m sampler.Memory) uint64 { return m.Free }}
	if runtime.GOOS == "darwin" {
		wired := segment{"Wired", func(p *palette) gfx.Color { return p.orange }, func(m sampler.Memory) uint64 { return m.Wired }}
		comp := segment{"Compressed", func(p *palette) gfx.Color { return p.purple }, func(m sampler.Memory) uint64 { return m.Compressed }}
		return []segment{app, wired, comp}, []segment{cached, free}
	}
	app.name = "Used"
	return []segment{app}, []segment{cached, free}
}

type legendDot struct {
	l     *gowt.LabelView
	color func(p *palette) gfx.Color
}

type monitor struct {
	app    *gowt.App
	host   sampler.Host
	pal    *palette
	accent gfx.Color
	cards  []*gowt.SectionView
	dots   []legendDot

	cpu, mem, net *chart
	cores, bar    *meterView
	usedSegs      []segment
	allSegs       []segment

	cpuValue, memValue, rxValue, txValue, memNote, swapNote, coresNote *gowt.LabelView
	segValues                                                          []*gowt.LabelView
	uptime, load, procCount                                            *gowt.LabelView

	win      *gowt.Window
	table    *gowt.TableView
	cols     []*gowt.Column
	procs    []sampler.Process
	sortCol  int
	sortDesc bool
	cells    [][4]string
	first    bool
}

// snapping is true under GOWT_SNAP, a screenshot run. The charts need about 25 s of samples to show curves,
// so the capture waits for them unless the caller set a longer delay.
var snapping = os.Getenv("GOWT_SNAP") != ""

// demo is set under GOWT_DEMO=1, the README recording: the screen capture loads this process to over
// 100% CPU, so the table leaves it out instead of heading the list with the recorder's cost.
var demo = os.Getenv("GOWT_DEMO") == "1"

func main() {
	if snapping {
		if ms, _ := strconv.Atoi(os.Getenv("GOWT_SNAP_DELAY")); ms < 25000 {
			os.Setenv("GOWT_SNAP_DELAY", "25000")
		}
	}
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		smp, err := sampler.New()
		if err != nil {
			app.Window("Monitor", gowt.Label("This system cannot be monitored: "+err.Error())).Show()
			return
		}
		m := &monitor{app: app, host: smp.Host(), sortCol: 1, sortDesc: true, first: true}
		m.build()
		go m.sampleLoop(smp)
		app.Every(10*time.Second, m.refreshPalette)
		app.OnThemeChange(func(bool) { m.refreshPalette() })
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (m *monitor) palette() *palette {
	card := gfx.Color{R: 0.93, G: 0.93, B: 0.94, A: 1}
	if m.app.Dark() {
		card = gfx.Color{R: 0.17, G: 0.17, B: 0.18, A: 1}
	}
	if len(m.cards) > 0 {
		if c := m.cards[0].Unwrap().GetBackground(); c != nil {
			rgb := c.GetRGB()
			card = gfx.Color{R: float64(rgb.Red) / 255, G: float64(rgb.Green) / 255, B: float64(rgb.Blue) / 255, A: 1}
		}
	}
	if r, g, b, ok := systemAccent(); ok {
		m.accent = gfx.RGB(r, g, b)
	} else if m.accent == (gfx.Color{}) {
		m.accent = gfx.RGB(10, 132, 255)
	}
	return newPalette(m.app.Dark(), card, m.accent, gowt.Type.Caption.Px)
}

// refreshPalette reads the card colour, appearance and accent again and restyles what changed.
func (m *monitor) refreshPalette() {
	p := m.palette()
	if m.pal != nil && *p == *m.pal {
		return
	}
	m.pal = p
	for _, c := range []*chart{m.cpu, m.mem, m.net} {
		c.setPalette(p)
	}
	m.cores.setPalette(p)
	m.bar.setPalette(p)
	for _, d := range m.dots {
		d.l.SetImage(kit.Dot(m.app, toRGB(d.color(p)), 10))
	}
}

func toRGB(c gfx.Color) gowt.RGB {
	return gowt.RGB{R: uint8(c.R*255 + 0.5), G: uint8(c.G*255 + 0.5), B: uint8(c.B*255 + 0.5)}
}

func (m *monitor) card(grow gowt.Option, opts ...gowt.Option) gowt.Option {
	return gowt.Section(append([]gowt.Option{grow, gowt.Init(func(s *gowt.SectionView) { m.cards = append(m.cards, s) })}, opts...)...)
}

// dot is a legend dot, recoloured with the accent.
func (m *monitor) dot(color func(p *palette) gfx.Color) gowt.Option {
	return gowt.Label("", gowt.Init(func(l *gowt.LabelView) {
		m.dots = append(m.dots, legendDot{l, color})
	}))
}

func lbl(dst **gowt.LabelView, text string, opts ...gowt.Option) gowt.Option {
	return gowt.Label(text, append(opts, gowt.Init(func(l *gowt.LabelView) { *dst = l }))...)
}

func (m *monitor) legend(name string, color func(p *palette) gfx.Color) gowt.Option {
	return gowt.HBox(gowt.Gap(gowt.Spacing.XS), m.dot(color), gowt.Label(name, kit.Faint(), gowt.Typeface(gowt.Type.Caption)))
}

func (m *monitor) build() {
	app := m.app
	pal := newPalette(app.Dark(), gfx.Color{R: 0.93, G: 0.93, B: 0.94, A: 1}, gfx.RGB(10, 132, 255), gowt.Type.Caption.Px)
	m.pal = pal
	m.cpu = newChart(chartSpec{
		names: []string{"System", "User"}, stacked: true, total: "CPU", fixedMax: 1,
		fracs: []float64{0, 0.25, 0.5, 0.75, 1}, widest: "100%",
		color: func(p *palette, i int) gfx.Color { return []gfx.Color{p.orange, p.accent}[i] },
		axis:  pct, value: pct,
	}, pal, windows[0])
	used, rest := memSegments()
	m.usedSegs, m.allSegs = used, append(append([]segment(nil), used...), rest...)
	total := float64(m.host.Memory)
	names := make([]string, len(used))
	for i, s := range used {
		names[i] = s.name
	}
	m.mem = newChart(chartSpec{
		names: names, stacked: len(used) > 1, total: "Used", fixedMax: 1,
		fracs: []float64{0, 0.25, 0.5, 0.75, 1}, widest: "000 GB",
		color: func(p *palette, i int) gfx.Color { return used[i].color(p) },
		axis:  func(v float64) string { return gbAxis(v * total) },
		value: func(v float64) string { return bytesStr(uint64(v * total)) },
	}, pal, windows[0])
	m.net = newChart(chartSpec{
		names: []string{"Sent", "Received"}, minMax: 100e3, fracs: []float64{0, 0.5, 1}, widest: "000 KB/s",
		color: func(p *palette, i int) gfx.Color { return []gfx.Color{p.orange, p.accent}[i] },
		axis:  rate, value: rate,
	}, pal, windows[0])
	m.cores = newCoreMeters(pal)
	m.bar = newMemBar(pal, func(p *palette) []gfx.Color {
		cols := make([]gfx.Color, len(m.allSegs))
		for i, s := range m.allSegs {
			cols[i] = s.color(p)
		}
		return cols
	})

	w := app.Window("Monitor")
	m.win = w
	kit.Size(w)
	w.SetMinSize(900, 520)
	w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(gowt.Scaled(14)), gowt.Gap(gowt.Spacing.MD),
		m.header(),
		gowt.HBox(gowt.Grow(), gowt.Gap(gowt.Spacing.MD),
			gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.MD), m.cpuCard(), m.memCard(), m.netCard()),
			gowt.VBox(gowt.Size(gowt.Scaled(440), 0), gowt.GrowY(), gowt.Gap(gowt.Spacing.MD), m.coresCard(), m.procCard()))))
	w.Show()
	m.refreshPalette()
}

func (m *monitor) header() gowt.Option {
	stat := func(name string, dst **gowt.LabelView) gowt.Option {
		return gowt.VBox(gowt.Gap(2), gowt.Label(name, kit.Faint(), gowt.Typeface(gowt.Type.Caption)),
			lbl(dst, "-", gowt.Typeface(gowt.Type.BodyStrong)))
	}
	sub := m.host.OS
	if m.host.CPU != "" {
		sub += "  -  " + m.host.CPU
	}
	sub += fmt.Sprintf("  -  %d cores  -  %s", m.host.Cores, bytesStr(m.host.Memory))
	items := []gowt.Option{
		gowt.VBox(gowt.Gap(gowt.Spacing.XS), gowt.Label(m.host.Name, gowt.Typeface(gowt.Type.Title)), gowt.Label(sub, kit.Faint())),
		gowt.Spacer(),
		stat("Uptime", &m.uptime),
	}
	if runtime.GOOS != "windows" {
		items = append(items, stat("Load average", &m.load))
	}
	items = append(items, stat("Processes", &m.procCount),
		gowt.Combo([]string{"Last minute", "Last 5 minutes", "Last 15 minutes"}, gowt.ReadOnly(), gowt.Selected(0),
			gowt.OnSelect(func(i int) {
				for _, c := range []*chart{m.cpu, m.mem, m.net} {
					c.setWindow(windows[i])
				}
			})))
	return gowt.HBox(append([]gowt.Option{gowt.GrowX(), gowt.Gap(gowt.Scaled(24))}, items...)...)
}

// head is a card's title row. Every row is as tall as a Title value, with or without one, so the
// titles of cards side by side sit on one line.
func head(title string, extra ...gowt.Option) gowt.Option {
	h := int(math.Ceil(gowt.Type.Title.Px * 1.4))
	return gowt.HBox(append([]gowt.Option{gowt.GrowX(), gowt.Size(0, h), gowt.Gap(gowt.Spacing.MD),
		gowt.Label(title, gowt.Typeface(gowt.Type.Subtitle), gowt.GrowY(), gowt.Place(gowt.AlignStart, gowt.AlignCenter))}, extra...)...)
}

func (m *monitor) cpuCard() gowt.Option {
	return m.card(gowt.Grow(), gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS),
		head("CPU", m.legend("User", func(p *palette) gfx.Color { return p.accent }), m.legend("System", func(p *palette) gfx.Color { return p.orange }),
			gowt.Spacer(), lbl(&m.cpuValue, "-", gowt.Typeface(gowt.Type.Title))),
		m.cpu.option(gowt.Scaled(300), gowt.Scaled(76))))
}

func (m *monitor) memCard() gowt.Option {
	var legend []gowt.Option
	m.segValues = make([]*gowt.LabelView, len(m.allSegs))
	for i, s := range m.allSegs {
		legend = append(legend, gowt.HBox(gowt.Gap(gowt.Spacing.SM), m.dot(s.color),
			gowt.VBox(gowt.Gap(1), gowt.Label(s.name, kit.Faint(), gowt.Typeface(gowt.Type.Caption)),
				lbl(&m.segValues[i], "-", gowt.Typeface(gowt.Type.BodyStrong)))))
	}
	legend = append(legend, gowt.Spacer(), lbl(&m.swapNote, "", kit.Faint(), gowt.Typeface(gowt.Type.Caption)))
	return m.card(gowt.Grow(), gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS),
		head("Memory", gowt.Spacer(), lbl(&m.memNote, "", kit.Faint()), lbl(&m.memValue, "-", gowt.Typeface(gowt.Type.Title))),
		m.mem.option(gowt.Scaled(300), gowt.Scaled(40)),
		m.bar.strip(gowt.Scaled(16)),
		gowt.HBox(append([]gowt.Option{gowt.GrowX(), gowt.Gap(gowt.Scaled(20))}, legend...)...)))
}

func (m *monitor) netCard() gowt.Option {
	return m.card(gowt.Grow(), gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS),
		head("Network", gowt.Spacer(),
			m.legend("Received", func(p *palette) gfx.Color { return p.accent }), lbl(&m.rxValue, "-", gowt.Typeface(gowt.Type.Subtitle)),
			m.legend("Sent", func(p *palette) gfx.Color { return p.orange }), lbl(&m.txValue, "-", gowt.Typeface(gowt.Type.Subtitle))),
		m.net.option(gowt.Scaled(300), gowt.Scaled(32))))
}

func (m *monitor) coresCard() gowt.Option {
	return m.card(gowt.GrowX(), gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS), gowt.Size(0, gowt.Scaled(190)),
		head("Cores", gowt.Spacer(), lbl(&m.coresNote, "", kit.Faint())),
		m.cores.option(gowt.Scaled(200), gowt.Scaled(110))))
}

var columns = []struct {
	title string
	width int
	right bool
}{{"Process", 184, false}, {"% CPU", 64, true}, {"Memory", 80, true}, {"PID", 56, true}}

func (m *monitor) procCard() gowt.Option {
	return m.card(gowt.Grow(), gowt.VBox(gowt.Grow(), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS),
		head("Processes", gowt.Spacer(), gowt.Label("CPU is percent of one core", kit.Faint(), gowt.Typeface(gowt.Type.Caption))),
		gowt.Table(gowt.Grow(), gowt.Size(0, gowt.Scaled(110)), gowt.Tooltip("100% is one core fully busy, as in Activity Monitor"), gowt.Init(func(t *gowt.TableView) {
			m.table = t
			for i, c := range columns {
				opts := []gowt.Option{}
				if c.right {
					opts = append(opts, gowt.Right())
				}
				col := t.Column(c.title, gowt.Scaled(c.width), opts...)
				col.OnClick(func() { m.sortBy(i) })
				m.cols = append(m.cols, col)
			}
			m.table.SetSort(m.cols[m.sortCol], gowt.SortDescending)
		}))))
}

// sortBy sorts by column i; a second click on the same column reverses it.
func (m *monitor) sortBy(i int) {
	if i == m.sortCol {
		m.sortDesc = !m.sortDesc
	} else {
		m.sortCol, m.sortDesc = i, i != 0 // names ascend first, numbers descend first
	}
	dir := gowt.SortAscending
	if m.sortDesc {
		dir = gowt.SortDescending
	}
	m.table.SetSort(m.cols[i], dir)
	m.fillTable()
}

const maxRows = 40

// fillTable shows the top processes in the sort order, rewriting rows in place so the selection and
// the scroll position stay, and tracking the selected process by PID as the order changes.
func (m *monitor) fillTable() {
	ps := make([]sampler.Process, 0, len(m.procs))
	for _, p := range m.procs {
		if demo && p.PID == os.Getpid() {
			continue
		}
		ps = append(ps, p)
	}
	sort.Slice(ps, func(a, b int) bool {
		x, y := ps[a], ps[b]
		c := 0
		switch m.sortCol {
		case 0:
			c = strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
		case 1:
			c = cmp.Compare(x.CPU, y.CPU)
		case 2:
			c = cmp.Compare(x.RSS, y.RSS)
		}
		if c == 0 {
			c = cmp.Compare(x.PID, y.PID)
		}
		if m.sortDesc {
			return c > 0
		}
		return c < 0
	})
	if len(ps) > maxRows {
		ps = ps[:maxRows]
	}
	t := m.table
	selPID := -1
	if i := t.Index(); i >= 0 && i < t.Len() {
		if pid, ok := t.RowAt(i).Data().(int); ok {
			selPID = pid
		}
	}
	t.Batch(func() {
		for i, p := range ps {
			cell := [4]string{p.Name, fmt.Sprintf("%.1f", p.CPU), bytesStr(p.RSS), fmt.Sprint(p.PID)}
			if i >= t.Len() {
				t.Row(cell[:]...).SetData(p.PID)
				m.cells = append(m.cells, cell)
				continue
			}
			row := t.RowAt(i)
			for c := range cell {
				if m.cells[i][c] != cell[c] {
					row.SetTextAt(c, cell[c])
				}
			}
			row.SetData(p.PID)
			m.cells[i] = cell
		}
		for t.Len() > len(ps) {
			t.Remove(t.Len() - 1)
			m.cells = m.cells[:len(m.cells)-1]
		}
	})
	if selPID >= 0 {
		for i, p := range ps {
			if p.PID == selPID {
				if t.Index() != i {
					t.Select(i)
				}
				return
			}
		}
		t.Deselect()
	}
}

// sampleLoop reads the host every second and hands each reading to the UI thread.
func (m *monitor) sampleLoop(smp *sampler.Sampler) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for n := 0; ; n++ {
		snap, err := smp.Sample(n%2 == 0)
		if err == nil {
			m.app.Async(func() { m.apply(snap) })
		}
		<-tick.C
	}
}

func (m *monitor) apply(s sampler.Snapshot) {

	if m.first { // the first reading has no rates: only the process names are of use
		m.first = false
		m.procs = s.Procs
		m.fillTable()
		return
	}
	m.cpu.push(s.Time, s.System, s.User)
	m.cpuValue.SetText(pct(s.CPU))

	total := float64(s.Mem.Total)
	vals := make([]float64, len(m.usedSegs))
	for i, seg := range m.usedSegs {
		vals[i] = float64(seg.get(s.Mem)) / total
	}
	m.mem.push(s.Time, vals...)
	m.memValue.SetText(bytesStr(s.Mem.Used))
	m.memNote.SetText("of " + bytesStr(s.Mem.Total))
	frac := make([]float64, len(m.allSegs))
	for i, seg := range m.allSegs {
		v := seg.get(s.Mem)
		frac[i] = float64(v) / total
		m.segValues[i].SetText(bytesStr(v))
	}
	m.bar.setValues(frac, 1)
	if s.Mem.SwapTotal > 0 {
		m.swapNote.SetText(fmt.Sprintf("Swap %s of %s", bytesStr(s.Mem.SwapUsed), bytesStr(s.Mem.SwapTotal)))
	} else {
		m.swapNote.SetText("No swap")
	}

	m.net.push(s.Time, s.TxRate, s.RxRate)
	m.rxValue.SetText(rate(s.RxRate))
	m.txValue.SetText(rate(s.TxRate))

	m.cores.setValues(s.Cores, len(s.Cores))
	m.coresNote.SetText(fmt.Sprintf("%d logical", len(s.Cores)))

	m.uptime.SetText(uptimeStr(s.Uptime))
	if s.HasLoad && m.load != nil {
		m.load.SetText(fmt.Sprintf("%.2f  %.2f  %.2f", s.Load[0], s.Load[1], s.Load[2]))
	}
	if s.Procs != nil {
		m.procs = s.Procs
		m.procCount.SetText(fmt.Sprint(len(s.Procs)))
		m.fillTable()
	}
	m.repaintCards()
}
