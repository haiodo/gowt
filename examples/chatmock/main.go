// Command chatmock is a standalone visual demo of a team chat window built after
// .work/ui-mockup/reference.png: a translucent sidebar with grouped channels and direct
// messages, a white content card with a message feed, a NEW divider and a glass composer.
// Channel selection switches the header and the composer hint; sending appends a message.
// All data is hardcoded, the window is fixed at 1160x780.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

// person is a chat participant: name, avatar initials and avatar colour.
type person struct {
	name, initials string
	color          gowt.RGB
}

var (
	me      = person{name: "Alex Rivera", initials: "AR", color: gowt.RGB{R: 30, G: 30, B: 40}}
	maya    = person{name: "Maya Chen", initials: "MC", color: gowt.RGB{R: 130, G: 80, B: 230}}
	jordan  = person{name: "Jordan Brooks", initials: "JB", color: gowt.RGB{R: 245, G: 140, B: 10}}
	camille = person{name: "Camille Dubois", initials: "CD", color: gowt.RGB{R: 40, G: 170, B: 190}}
	fizz    = person{name: "Fizz", initials: "FZ", color: gowt.RGB{R: 48, G: 176, B: 78}}
	priya   = person{name: "Priya Shah", initials: "PS", color: gowt.RGB{R: 230, G: 76, B: 60}}
	honey   = person{name: "Honey", initials: "HN", color: gowt.RGB{R: 250, G: 190, B: 40}}
)

// reaction is one pill under a message: a symbol icon and a count.
type reaction struct {
	icon  string
	color gowt.RGB
	count string
}

// message is one feed entry.
type message struct {
	who       person
	time      string
	paras     []string
	reactions []reaction
}

// initialFeed returns the feed of the selected channel. The same transcript stands in for
// every channel: the demo is about the layout, not per-channel data.
func initialFeed() []message {
	return []message{
		{who: me, time: "3:24 PM", paras: []string{"Perfect. That's the move."}, reactions: []reaction{{icon: pathHexagon, color: honey.color, count: "2"}}},
		{who: maya, time: "3:25 PM", paras: []string{"Small thing: the desktop-to-mobile handoff still feels a little fast."}},
		{who: jordan, time: "3:25 PM", paras: []string{"Yeah - I want one extra beat on the sent message."}},
		{who: camille, time: "3:25 PM", paras: []string{"That would give the camera somewhere to land too."}},
		{who: maya, time: "3:26 PM", paras: []string{"@Fizz can you turn that into a clean three-beat capture plan?"}, reactions: []reaction{{icon: pathThumbsUp, count: "1"}, {icon: pathEye, count: "1"}}},
		{who: fizz, time: "3:26 PM", paras: []string{
			"Absolutely - clean three-beat capture plan:",
			"1. Desktop compose",
			"   - Start on the project view.",
			"   - One clean cursor move into the update field.",
			"   - Type the short update without rushing.",
			"2. Project header settle",
			"   - After the transition, hold on the project header for a half beat.",
			"   - Let the context change become legible before moving on.",
			"3. Mobile handoff / sent message",
			"   - Cut to mobile.",
			"   - Let the sent message land and hold one extra beat.",
			"   - This gives the camera a clear final resting point and makes the handoff feel intentional.",
			"Tiny rule of thumb: cursor moves once, transition breathes once, sent state lands once. Nice and buzzy.",
		}},
		{who: honey, time: "3:28 PM", paras: []string{"@Honey, over to you for the final capture pass."}, reactions: []reaction{{icon: pathThumbsUp, count: "1"}, {icon: pathEye, count: "1"}}},
	}
}

// channel is one sidebar row; group is the section header it sits under, dm marks a
// direct message row (no # prefix in the header).
type channel struct {
	id, name, group string
	badge           string
	dm              bool
	who             *person
}

var channels = []channel{
	{id: "inbox", name: "Inbox", group: ""},
	{id: "agents", name: "Agents", group: ""},
	{id: "announcements", name: "announcements", group: "The Hive"},
	{id: "general", name: "general", group: "The Hive", badge: "dot"},
	{id: "design", name: "design", group: "Product", badge: "dot"},
	{id: "flight-path", name: "flight-path", group: "Product", badge: "4"},
	{id: "mobile", name: "mobile", group: "Product", badge: "dot"},
	{id: "marketing", name: "marketing", group: "Launch Swarm", badge: "dot"},
	{id: "queen-bee-launch", name: "queen-bee-launch", group: "Launch Swarm", badge: "dot"},
	{id: "welcome", name: "Welcome", group: "Channels"},
	{id: "jordan", name: "Jordan Brooks", group: "Direct messages", dm: true, who: &jordan, badge: "1"},
	{id: "maya", name: "Maya Chen", group: "Direct messages", dm: true, who: &maya, badge: "1"},
	{id: "priya", name: "Priya Shah", group: "Direct messages", dm: true, who: &priya, badge: "1"},
}

var groups = []string{"", "The Hive", "Product", "Launch Swarm", "Channels", "Direct messages"}

var (
	feed  = initialFeed()
	split = 1 // messages before the NEW divider: the first one only
)

// uiErrors collects what App.OnError reports (a misplaced option among it), for the QA
// "errors" handler; the default logging is kept.
var uiErrors []string

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		app.OnError(func(err error) {
			uiErrors = append(uiErrors, err.Error())
			log.Printf("%v\n%s", err, debug.Stack())
		})
		qa.HandleValue("errors", func(qa.Args) (any, error) { return uiErrors, nil })
		theApp = app
		// Without a window-level glass backdrop the window stays opaque, so the warm
		// tone shows in the title bar strip and behind the sidebar.
		w := app.Window(" ", gowt.Padding(0), gowt.Background(sidebarBg))
		w.SetMinSize(880, 560)
		w.SetSize(1160, 780)
		// CHATMOCK_SIZE="WxH" starts the window at another size (QA: the min-size check).
		if sz := os.Getenv("CHATMOCK_SIZE"); sz != "" {
			var x, y int
			if _, err := fmt.Sscanf(sz, "%dx%d", &x, &y); err == nil {
				w.SetSize(x, y)
			}
		}
		look.SetRoundedCorners(w, true)
		look.SetFullSizeContent(w, true) // the inset must be measurable before the sidebar builds

		header := rx.New("# flight-path")
		sel := rx.New("flight-path")

		// Flush full-height sidebar to the window edges, content filling the rest; the
		// sidebar children start below the title bar strip, the material runs on to the top.
		var side, cont *gowt.Panel
		w.Add(
			gowt.VBox(gowt.ID("sidebar"), gowt.Ref(&side),
				sidebar(app, header, sel, contentTopInset(w))),
			gowt.VBox(gowt.Ref(&cont),
				content(app, header)))
		look.SetSidebar(w, side, cont, gowt.Scaled(280))
		look.SetSidebarFullHeight(w, true)
		// SetGlass clears the window background; restore the warm tone under the glass, or the strip
		// refracts the grey default. The window stays opaque, with no desktop bleed.
		glassTint := gowt.RGBA{R: 232, G: 232, B: 206, A: 220} // sidebarBg, translucent: the glass stays glass
		look.SetGlass(w, true)
		look.SetGlassRadius(w, 0)
		look.SetGlassTint(w, glassTint)
		look.SetGlassTint(side, glassTint)
		w.SetBackground(sidebarBg)
		qa.HandleValue("inset", func(qa.Args) (any, error) { return contentTopInset(w), nil })
		// QA resize hook: the regression test for the row width drives the sidebar splitter
		// programmatically, no physical pointer.
		qa.Handle("sidebar-width", func(a qa.Args) error { look.SetSidebarWidth(w, a.Int("width")); return nil })
		render(feedPanel)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// render rebuilds the feed panel: messages before the split, the NEW divider, the rest.
func render(feedBox *gowt.Panel) {
	if feedBox == nil {
		return
	}
	feedBox.Clear()
	for i, m := range feed {
		if i == split {
			feedBox.Add(newDivider())
		}
		feedBox.Add(messageRow(m))
	}
	// Rows go into the scroll's content panel, not through ScrollView.Add, so the scrollable
	// range is recomputed here; without it the feed renders but never scrolls.
	if feedScroll != nil {
		feedScroll.Fit()
	}
}

// channelTitle is the header text for a channel row.
func channelTitle(c channel) string {
	if c.dm {
		return c.name
	}
	return "# " + c.name
}
