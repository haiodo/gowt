package main

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/swt"
)

func rexWindow(app *gowt.App, openSettings func()) *gowt.Window {
	w := app.Window("Rex")
	w.SetSize(1280, 800)
	w.SetMinSize(900, 600)
	w.SetTitle("")
	w.SetBackground(bgApp)
	look.SetFullSizeContent(w, true)
	look.SetSidebarFullHeight(w, true)
	more := gowt.PopupMenu(w)
	more.Item("Activity", func() {})
	more.Item("Commands", func() {})
	more.Item("New Session", func() {})
	tab := newSessionTab(app)
	chrome := &rexChrome{w: w, more: more, tab: tab, variant: -1}
	var split *gowt.SplitView
	// The toolbar must exist before titleInset measures the strip it takes.
	chrome.update()
	topInset := titleInset(w)
	var sidebar, content *gowt.Panel
	w.Add(rexSidebar(app, topInset, &sidebar, openSettings), rexContent(app, w, &content, openSettings, tab, &split))
	look.SetSidebar(w, sidebar, content, gowt.Scaled(264))
	look.SetSidebarMinWidth(w, gowt.Scaled(200))
	look.SetContentFullHeight(w, true)
	chrome.update()
	tab.expanded = look.SidebarWidth(w)
	look.OnSidebarResize(w, func(visible bool, width int) {
		tab.follow(visible, width)
		chrome.update()
	})
	look.OnSidebarChange(w, func(visible bool, width int) {
		tab.settle(visible, width)
		chrome.update()
	})
	qa.Handle("sidebar-width", func(a qa.Args) error {
		look.SetSidebarWidth(w, a.Int("width"))
		return nil
	})
	qa.Handle("rex-size", func(a qa.Args) error {
		w.SetSize(a.Int("width"), a.Int("height"))
		return nil
	})
	qa.HandleValue("sidebar-metrics", func(qa.Args) (any, error) {
		return look.SidebarWidth(w), nil
	})
	qa.HandleValue("sidebar-sash", func(a qa.Args) (any, error) {
		for _, control := range w.Unwrap().GetChildren() {
			sash, ok := control.Impl().(*swt.Sash)
			if !ok {
				continue
			}
			e := swt.NewEvent()
			e.X, e.Y, e.Button = int32(a.Int("x")), sash.GetBounds().Y, 1
			sash.NotifyListeners(int32(a.Int("event")), e)
			return map[string]any{"x": sash.GetBounds().X, "inset": tab.inset}, nil
		}
		return nil, fmt.Errorf("no sidebar sash")
	})
	qa.Handle("split", func(a qa.Args) error {
		split.SetWeights(a.Int("left"), a.Int("right"))
		return nil
	})
	qa.Handle("toggle-sidebar", func(qa.Args) error {
		chrome.toggle()
		return nil
	})
	// qatest keeps one app per test package: undo the sidebar, split and toggle state a test left.
	qa.Handle("reset", func(qa.Args) error {
		chrome.hidden = false
		look.SetSidebarVisible(w, true)
		look.SetSidebarWidth(w, gowt.Scaled(264))
		split.SetWeights(50, 50)
		chrome.update()
		return nil
	})
	qa.Handle("appearance", func(a qa.Args) error {
		mode := gowt.AppearanceSystem
		switch a.String("mode") {
		case "light":
			mode = gowt.AppearanceLight
		case "dark":
			mode = gowt.AppearanceDark
		}
		setAppearance(app, mode)
		return nil
	})
	w.OnKey("Cmd+,", openSettings)
	return w
}

type rexChrome struct {
	w        *gowt.Window
	more     *gowt.Menu
	tab      *sessionTab
	variant  int
	minWidth int
	hidden   bool
}

func (c *rexChrome) toggle() {
	c.hidden = !c.hidden
	look.SetSidebarVisibleAnimated(c.w, !c.hidden)
	c.update()
}

func (c *rexChrome) update() {
	slots := 0
	width := 0
	itemWidth := sidebarItemWidth(c.w)
	minWidth := sidebarMinWidth(itemWidth)
	if c.minWidth != minWidth {
		c.minWidth = minWidth
		look.SetSidebarMinWidth(c.w, minWidth)
	}
	if look.SidebarVisible(c.w) {
		width = look.SidebarWidth(c.w)
		slots = sidebarToolbarSlots(width, itemWidth)
	}
	variant := slots
	plusFits := slots == 3 && sidebarPlusFits(width, itemWidth)
	if slots == 3 && !plusFits {
		variant = 4 // three items fit, but New Session folds into More
	}
	if variant == c.variant {
		return
	}
	c.variant = variant
	items := []look.ToolbarItem{
		{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, SidebarLead: true, Plain: true, Small: true,
			ID: "toggle-sidebar", OnClick: c.toggle},
	}
	eye := look.ToolbarItem{Title: "Show Activity ⌘.", Symbol: "eye", Sidebar: true, Plain: true, Small: true,
		ID: "toolbar-eye", OnClick: func() {}}
	command := look.ToolbarItem{Title: "Commands", Symbol: "command", Sidebar: true, Plain: true, Small: true,
		ID: "toolbar-command", OnClick: func() {}}
	plus := look.ToolbarItem{Title: "New Session", Symbol: "plus", Sidebar: true, Plain: true, Small: true,
		ID: "toolbar-plus", OnClick: func() {}}
	more := look.ToolbarItem{Title: "More", Symbol: "ellipsis", Sidebar: true, Plain: true, Small: true,
		ID: "more", OnClick: c.more.Show}
	switch slots {
	case 3:
		if plusFits {
			items = append(items, eye, command, plus)
		} else {
			items = append(items, eye, command, more)
		}
	case 2:
		items = append(items, eye, more)
	case 1:
		items = append(items, more)
	}
	look.SetToolbar(c.w, items...)
}

func sidebarMinWidth(itemWidth int) int { return 200 + 2*(itemWidth-36) }

// sidebarToolbarSlots is how many Small Sidebar items fit before the sidebar edge. The two fitted
// spacers also grow when AppKit's item viewer grows from 36 points to itemWidth.
func sidebarToolbarSlots(width, itemWidth int) int {
	lead := 156 + 2*(itemWidth-36)
	for n := 3; n > 0; n-- {
		if lead+itemWidth*n <= width+2 {
			return n
		}
	}
	return 0
}

func sidebarPlusFits(width, itemWidth int) bool {
	return 156+2*(itemWidth-36)+4*itemWidth <= width+2
}

// sidebarTrailingGap is the unused part of the trailing inset after the fitted spacer reaches its
// minimum. It grows from zero at minWidth and is capped by the native layout of the selected variant.
func sidebarTrailingGap(width, itemWidth int) int {
	width = max(width, sidebarMinWidth(itemWidth))
	limit := 16 // spacerGap 4: 20 - 4
	if sidebarToolbarSlots(width, itemWidth) == 3 && !sidebarPlusFits(width, itemWidth) {
		limit = 8 // the three-item More variant has 2*spacerGap
	}
	return min(limit, width-sidebarMinWidth(itemWidth))
}

func rexSidebar(app *gowt.App, topInset int, ref **gowt.Panel, openSettings func()) gowt.Option {
	items := []listItem{
		{title: "This Mac", icon: symbol(app, pathMonitor, 14, fgDim), group: true},
		{title: "bold-savanna", icon: symbol(app, pathMonitor, 16, fgDim), value: 0},
		{title: "platform-go", icon: symbol(app, pathBlocks, 16, blue), value: 1},
		{title: "foundation3", icon: symbol(app, pathLayout, 16, purple), value: 2},
		{title: "foundation", icon: symbol(app, pathBlocks, 16, green), value: 3},
		{title: "summon-age-codex", icon: symbol(app, pathCommand, 16, fgDim), value: 4},
		{title: "summon-age/modeling", icon: symbol(app, pathLayout, 16, purple), value: 5},
		{title: "haiodo.blog", icon: symbol(app, pathGlobe, 16, blue), value: 6},
		{title: "ComfyUI", icon: symbol(app, pathPalette, 16, green), value: 7},
		{title: "Landing", icon: symbol(app, pathTerminal, 16, fgDim), value: 8},
		{title: "Tenniarb", icon: symbol(app, pathLayout, 16, purple), value: 9},
		{title: "FUSIO-1914 | foundation4", icon: symbol(app, pathBlocks, 16, blue), value: 10},
		{title: "Services", icon: symbol(app, pathSettings, 16, green), value: 11},
		{space: 70},
		{title: "2 Disconnected Hosts", group: true},
		{title: "m1max", icon: symbol(app, pathMonitor, 16, fgDim), value: -1},
		{title: "haido@10.0.2.2", icon: symbol(app, pathTerminal, 16, fgDim), value: -1},
	}
	list := newDrawnList(app, bgApp, 26, items, 10, func(int) {})
	list.closable = true
	list.id = "sessions-list"
	return gowt.VBox(gowt.Ref(ref), gowt.ID("rex-sidebar"), gowt.Background(bgApp), gowt.Foreground(fg),
		gowt.Gap(0), gowt.Padding(0),
		gowt.VBox(gowt.Grow(), gowt.Indent(0, topInset), gowt.Gap(0), gowt.Padding(0),
			list.Option(),
			gowt.HBox(gowt.GrowX(), gowt.Padding(16), gowt.Gap(gowt.Spacing.SM),
				iconButton(app, pathKeyboard, gowt.Scaled(24), 14, fgDim, "Keyboard", func() {}),
				gowt.Spacer(),
				iconButton(app, pathGlobe, gowt.Scaled(24), 14, fgDim, "Network", func() {}),
				iconButton(app, pathSettings, gowt.Scaled(24), 14, fgDim, "Settings", openSettings,
					gowt.ID("open-settings-sidebar")))))
}

func rexContent(app *gowt.App, w *gowt.Window, ref **gowt.Panel, openSettings func(), tab *sessionTab, split **gowt.SplitView) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.ID("rex-content"), gowt.Background(bgApp), gowt.Foreground(fg),
		gowt.Gap(0), gowt.Padding(0),
		gowt.VBox(gowt.Size(0, gowt.Scaled(4))),
		gowt.HBox(gowt.Grow(), gowt.Gap(0),
			gowt.VBox(gowt.Size(8, 0)),
			gowt.Split(gowt.Grow(), gowt.Ref(split), gowt.ID("rex-split"), gowt.Background(bgApp),
				gowt.Init(func(s *gowt.SplitView) { s.SetWeights(50, 50) }),
				paneCard(bgCard, codexPane(app, tab)),
				paneCard(bgShell, shellPane(app, openSettings))),
			gowt.VBox(gowt.Size(8, 0))),
		gowt.VBox(gowt.Size(0, 8)))
}

func paneCard(bg gowt.Adaptive, child gowt.Option) gowt.Option {
	return gowt.HBox(gowt.Grow(), gowt.Padding(0), gowt.Gap(0),
		gowt.VBox(gowt.Size(3, 0)),
		gowt.Section(gowt.Grow(), gowt.Background(bg), gowt.CardBorder(cardBorder), gowt.ID("pane-card"),
			gowt.VBox(gowt.Grow(), gowt.Background(bg), gowt.Padding(0), gowt.Gap(0), child)))
}

func codexPane(app *gowt.App, tab *sessionTab) gowt.Option {
	return gowt.VBox(gowt.Grow(), gowt.Background(bgCard), gowt.Gap(0), gowt.Padding(0),
		tab.Option(),
		rexTranscript(),
		rexComposer(app),
		rexStatus())
}

type sessionTab struct {
	icon     *gowt.Image
	canvas   *gowt.CanvasView
	inset    int
	expanded int
}

func newSessionTab(app *gowt.App) *sessionTab {
	t := &sessionTab{}
	themedSymbol(app, pathTerminal, 15, fg, func(img *gowt.Image) {
		t.icon = img
		if t.canvas != nil {
			t.canvas.Redraw()
		}
	})
	return t
}

func (t *sessionTab) Option() gowt.Option {
	return gowt.Canvas(t.draw, gowt.GrowX(), gowt.Size(0, 34), gowt.Ref(&t.canvas), gowt.ID("codex-tab"))
}

// follow keeps the title at the card edge while the sidebar is visible and past the toolbar when hidden.
func (t *sessionTab) follow(visible bool, width int) {
	if visible {
		t.expanded = width
		t.setInset(0)
		return
	}
	t.setInset(t.expanded)
}

// settle returns the title to the card edge when the width is final.
func (t *sessionTab) settle(visible bool, width int) {
	if visible {
		t.expanded = width
		t.setInset(0)
		return
	}
	if width > t.expanded {
		t.expanded = width
	}
	t.setInset(t.expanded)
}

func (t *sessionTab) setInset(inset int) {
	if inset == t.inset {
		return
	}
	t.inset = inset
	if t.canvas != nil {
		t.canvas.Redraw()
	}
}

func (t *sessionTab) draw(g *gowt.GC) {
	w, h := g.Size()
	g.SetFill(colour(bgCard))
	g.FillRect(0, 0, w, h)
	x := 6 + t.inset
	g.Image(t.icon, x, (h-15)/2)
	x += 21
	g.SetFont(gowt.Type.Body)
	g.SetColor(colour(fg))
	text := "давай пример такого UI | gowt"
	_, th := g.TextSize(text)
	g.Text(text, x, (h-th)/2)
	cx, cy := w-18, h/2
	g.SetColor(colour(fgDim))
	g.Line(cx-4, cy-4, cx+4, cy+4)
	g.Line(cx-4, cy+4, cx+4, cy-4)
}

func rexTranscript() gowt.Option {
	content := gowt.VBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Padding(16), gowt.Gap(2),
		gowt.HBox(gowt.GrowX(), mono(">_", gowt.Foreground(teal)), mono(" OpenAI Codex (v0.172.0)")),
		monoLine("   ~/Develop/private/gowt", gowt.Foreground(fgDim)),
		monoLine(" "),
		monoLine("> сделай пример такого UI на gowt"),
		monoLine(" "),
		gowt.HBox(gowt.GrowX(), mono("• ", gowt.Foreground(teal)),
			mono("Читаю docs/code-map.md и facade docs.", gowt.GrowX())),
		gowt.HBox(gowt.GrowX(), mono("• ", gowt.Foreground(teal)),
			mono("Собираю окно Rex: сессии, transcript, статус-бар.", gowt.GrowX())),
		monoLine(" "),
		monoLine("Ran ./node_modules/.bin/jest --runInBand --silent --coverage=false --forceExit"),
		monoLine("PASS src/billing/license.spec.ts", gowt.Foreground(green)),
		monoLine("Tests: 410 passed, 410 total"),
		monoLine("Snapshots: 0 total", gowt.Foreground(fgDim)),
		monoLine("Time: 5.982 s", gowt.Foreground(fgDim)),
		monoLine(" "),
		monoLine("Recap: FUSIO-1914 adds a license cap for active accounts.", gowt.Foreground(fgDim)),
		monoLine("Next: pass LICENSE_KEY to the services and check it with one local method.", gowt.Foreground(teal)))
	return gowt.Scroll(gowt.Grow(), gowt.NoHScroll(), gowt.Background(bgCard), content)
}

func rexComposer(app *gowt.App) gowt.Option {
	row := gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
		gowt.Place(gowt.AlignFill, gowt.AlignCenter),
		label(">", gowt.Foreground(teal), gowt.Typeface(monoFont())),
		gowt.Text(gowt.Hint("Ask Codex to do anything"), gowt.NoBorder(),
			gowt.Background(bgCard), gowt.Foreground(fg), gowt.GrowX()),
		iconButton(app, pathSend, gowt.Scaled(24), 14, fgDim, "Send", func() {}),
		iconButton(app, pathAttach, gowt.Scaled(24), 14, fgDim, "Attach", func() {}))
	return gowt.VBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Padding(10), gowt.Gap(gowt.Spacing.XS),
		row,
		label("← for agents   ? for shortcuts", faint(), gowt.Typeface(monoFont())))
}

func rexStatus() gowt.Option {
	return gowt.HBox(gowt.GrowX(), gowt.Size(0, 30), gowt.Background(bgCard),
		gowt.Padding(8), gowt.Gap(gowt.Spacing.SM),
		label("DeepSeek V4.1 Flash 1017 high", gowt.Typeface(semibold(gowt.Type.Caption))),
		label(" · ", faint(), gowt.Typeface(gowt.Type.Caption)),
		label("Ready", gowt.Foreground(teal), gowt.Typeface(gowt.Type.Caption)),
		label(" · Context 100% left", faint(), gowt.Typeface(gowt.Type.Caption)))
}

func shellPane(app *gowt.App, openSettings func()) gowt.Option {
	return gowt.VBox(gowt.Grow(), gowt.ID("shell-pane"), gowt.Background(bgShell), gowt.Gap(0), gowt.Padding(0),
		shellHeader(app, openSettings),
		shellTranscript(),
		shellStatus())
}

func shellStatus() gowt.Option {
	return gowt.HBox(gowt.GrowX(), gowt.Size(0, 24), gowt.Background(bgShell), gowt.Padding(6),
		gowt.Gap(gowt.Spacing.SM),
		gowt.Spacer(),
		label("4 warnings", gowt.Foreground(amber), gowt.Typeface(gowt.Type.Caption)),
		label(" · f2 to view", faint(), gowt.Typeface(gowt.Type.Caption)))
}

func shellHeader(app *gowt.App, openSettings func()) gowt.Option {
	return gowt.HBox(gowt.GrowX(), gowt.Size(0, 34), gowt.Background(bgShell),
		gowt.Padding(6), gowt.Indent(2, 0), gowt.Gap(gowt.Spacing.SM),
		label("Go", gowt.Foreground(teal), gowt.Typeface(gowt.Type.BodyStrong)),
		shellPath(app),
		iconButton(app, pathSplit, gowt.Scaled(20), 12, fgDim, "Split", func() {},
			gowt.ID("shell-split")),
		iconButton(app, pathSettings, gowt.Scaled(20), 12, fgDim, "Settings", openSettings,
			gowt.ID("open-settings-right")),
		iconButton(app, pathClose, gowt.Scaled(20), 12, fgDim, "Close", func() {},
			gowt.ID("shell-close")))
}

func shellPath(app *gowt.App) gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		_, h := g.Size()
		g.SetFont(gowt.Type.Body)
		g.SetColor(colour(shellFG))
		text := "~/Develop/private/gowt"
		_, th := g.TextSize(text)
		g.Text(text, 0, (h-th)/2)
	}, gowt.GrowX(), gowt.Size(0, 34), gowt.ID("shell-path"),
		gowt.Init(func(c *gowt.CanvasView) { app.OnThemeChange(func(bool) { c.Redraw() }) }))
}

func shellTranscript() gowt.Option {
	err := func(text string) gowt.Option { return monoLine(text, gowt.Foreground(red)) }
	line := func(text string) gowt.Option { return monoLine(text, gowt.Foreground(shellFG)) }
	content := gowt.VBox(gowt.GrowX(), gowt.Background(bgShell), gowt.Padding(16), gowt.Gap(1),
		line("$ go run examples/rex/main.go"),
		err("# command-line-arguments"),
		err("examples/rex/main.go:16:10: undefined: rexWindow"),
		err("examples/rex/main.go:20:14: undefined: settingsWindow"),
		line("$ go run examples/rex/mainwindow.go"),
		err("# command-line-arguments"),
		err("examples/rex/mainwindow.go:13:18: undefined: bgApp"),
		err("examples/rex/mainwindow.go:25:13: undefined: listItem"),
		err("examples/rex/mainwindow.go:26:29: undefined: symbol"),
		err("examples/rex/mainwindow.go:26:41: undefined: pathMonitor"),
		err("examples/rex/mainwindow.go:26:58: undefined: fgDim"),
		err("examples/rex/mainwindow.go:27:33: undefined: symbol"),
		err("examples/rex/mainwindow.go:27:45: undefined: pathMonitor"),
		err("examples/rex/mainwindow.go:27:62: undefined: fgDim"),
		err("examples/rex/mainwindow.go:28:32: undefined: symbol"),
		err("examples/rex/mainwindow.go:28:44: undefined: pathBlocks"),
		err("examples/rex/mainwindow.go:28:44: too many errors"),
		line("$ go run ./examples/rex/"),
		err("package examples/rex is not in std (/opt/homebrew/Cellar/go/1.27.1/libexec/src/examples/rex)"),
		line("$ go run ./examples/rex"),
		line(" "),
		monoLine("▶ running...", gowt.Foreground(teal)))
	return gowt.Scroll(gowt.Grow(), gowt.NoHScroll(), gowt.Background(bgShell), content)
}

func monoFont() gowt.Font {
	f := gowt.Type.Mono
	f.Px = 14
	return f
}

func mono(text string, opts ...gowt.Option) gowt.Option {
	return label(text, append([]gowt.Option{gowt.Typeface(monoFont())}, opts...)...)
}

func monoLine(text string, opts ...gowt.Option) gowt.Option {
	return mono(text, append([]gowt.Option{gowt.GrowX()}, opts...)...)
}
