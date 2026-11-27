package main

import "github.com/haiodo/gowt"

func categories(app *gowt.App) []category {
	return []category{
		{name: "Server", status: "Running", icon: symbol(app, pathMonitor, 16, fgDim), page: serverPage(app)},
		{name: "General", icon: symbol(app, pathSettings, 16, fgDim), page: generalPage(app)},
		{name: "Remote Hosts", icon: symbol(app, pathGlobe, 16, purple), page: remoteHostsPage(app)},
		{name: "Interface", icon: symbol(app, pathLayout, 16, fgDim), page: interfacePage(app)},
		{name: "Themes", icon: symbol(app, pathPalette, 16, purple), page: themesPage()},
		{name: "Terminal", icon: symbol(app, pathTerminal, 16, green), page: terminalPage()},
		{name: "Shortcuts", icon: symbol(app, pathCommand, 16, fgDim), page: shortcutsPage()},
		{name: "Integrations", icon: symbol(app, pathBlocks, 16, purple), page: integrationsPage()},
		{name: "Notifications", icon: symbol(app, pathBell, 16, red), page: notificationsPage()},
		{name: "Developer", icon: symbol(app, pathWrench, 16, fgDim), page: developerPage()},
	}
}

func generalPage(app *gowt.App) gowt.Option {
	update := gowt.HBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Gap(gowt.Spacing.MD),
		appTile(app, tileBlue, glyphDiamond, gowt.Scaled(40), false),
		gowt.VBox(gowt.Gap(1), gowt.Place(gowt.AlignFill, gowt.AlignCenter),
			label("Downloading Rex 0.1.0..."),
			label("82%", faint(), gowt.Typeface(gowt.Type.Caption))),
		gowt.Spacer(),
		progressBar(app, 82))
	channel := setting("Update Channel", "Finish the current update before changing channels.",
		gowt.HBox(gowt.Gap(gowt.Spacing.SM),
			gowt.Combo([]string{"Preview", "Stable", "Beta"}, gowt.ReadOnly(), gowt.Selected(0),
				gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
			infoButton(app)))
	return page(
		card(update, channel),
		heading("Client Info"),
		card(
			gowt.ID("client-info-card"),
			gowt.VBox(gowt.Gap(1), gowt.GrowX(), gowt.Background(bgCard),
				label("Label", faint(), gowt.Typeface(gowt.Type.Caption)),
				gowt.Text(gowt.Value("MacBook Pro (haido)"), gowt.GrowX())),
			setting("Remote Control", "Let Rex and other clients perform actions in this app.",
				gowt.Switch("", gowt.Checked(false)))),
		heading("App Icon"),
		card(
			gowt.HBox(gowt.GrowX(), gowt.Background(bgCard),
				label("Dynamic", faint(), gowt.Typeface(gowt.Type.Caption)),
				gowt.Spacer(),
				label("Alternates", faint(), gowt.Typeface(gowt.Type.Caption))),
			gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Background(bgCard),
				appTile(app, tilePurple, glyphDiamond, gowt.Scaled(44), true),
				appTile(app, tileBlue, glyphSpark, gowt.Scaled(44), false),
				appTile(app, tileGreen, glyphGrid, gowt.Scaled(44), false),
				appTile(app, tileGray, glyphBars, gowt.Scaled(44), false)),
			setting("Show app icon", "",
				gowt.Combo([]string{"Dock and menu bar", "Dock only", "Menu bar only", "None"},
					gowt.ReadOnly(), gowt.Selected(0)))),
		heading("Launch"))
}

func hostRow(app *gowt.App, name, address string) gowt.Option {
	return gowt.HBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Gap(gowt.Spacing.MD),
		gowt.Label("", gowt.Size(gowt.Scaled(30), gowt.Scaled(30)),
			gowt.Place(gowt.AlignStart, gowt.AlignCenter),
			gowt.Init(func(l *gowt.LabelView) {
				themedSymbol(app, pathMonitor, 18, fgDim, l.SetImage)
			})),
		gowt.VBox(gowt.Gap(1), gowt.Place(gowt.AlignStart, gowt.AlignCenter),
			label(name),
			label(address, faint(), gowt.Typeface(gowt.Type.Caption))),
		gowt.Spacer(),
		gowt.Button("Connect", gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
		infoButton(app))
}

func remoteHostsPage(app *gowt.App) gowt.Option {
	return page(
		card(
			hostRow(app, "m1max", "ssh://haido@192.168.1.177"),
			hostRow(app, "haido@10.0.2.2", "ssh://haido@10.0.2.2")),
		gowt.HBox(gowt.GrowX(), gowt.Spacer(), gowt.Button("Add Remote Host...")),
		heading("SSH"),
		card(
			setting("SSH agent", "Where ssh finds keys for ssh:// hosts.",
				gowt.Combo([]string{"System", "1Password", "Custom"}, gowt.ReadOnly(), gowt.Selected(0)))),
		gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
			gowt.Label("", gowt.Place(gowt.AlignStart, gowt.AlignStart),
				gowt.Init(func(l *gowt.LabelView) {
					themedIcon(app, "info", 14, l.SetImage)
				})),
			label("The macOS agent has no keys loaded. ssh still uses the key files in your ssh configuration.",
				faint(), gowt.Wrap(), gowt.GrowX())))
}

func terminalPage() gowt.Option {
	previewText := faint()
	return page(
		heading("Typography"),
		card(
			gowt.HBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Gap(gowt.Spacing.MD),
				label("Terminal font", gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
				gowt.Spacer(),
				label("Menlo", faint(), gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
				gowt.Button("Choose...")),
			gowt.Grid(4, gowt.GrowX(), gowt.Background(bgCard), gowt.Gap(gowt.Spacing.SM),
				labeled("Font style", gowt.Combo([]string{"Regular", "Medium", "Bold"},
					gowt.ReadOnly(), gowt.Selected(0))),
				labeled("Font size", gowt.HBox(gowt.Gap(2),
					gowt.Spinner(8, 72, gowt.Value(15)), label("pt"))),
				labeled("Weight", gowt.Text(gowt.Value("400"), gowt.ReadOnly(),
					gowt.Size(gowt.Scaled(64), 0))),
				labeled("Line height", gowt.HBox(gowt.Gap(2),
					gowt.Spinner(0, 200, gowt.Value(0)), label("%"))))),
		gowt.VBox(gowt.GrowX(),
			gowt.Background(gowt.Adaptive{
				Light: gowt.RGB{R: 245, G: 245, B: 247},
				Dark:  gowt.RGB{R: 28, G: 28, B: 30},
			}),
			gowt.Padding(gowt.Scaled(12)), gowt.Gap(2),
			label("~/rex main $ swift build && swift test",
				gowt.Typeface(gowt.Type.Mono), previewText),
			label("main.swift:4: warning: 'run() -> Bool' is deprecated",
				gowt.Typeface(gowt.Type.Mono), previewText),
			label("Build complete! (1.24 sec)",
				gowt.Typeface(gowt.Type.Mono), gowt.Foreground(fg))),
		gowt.Disclosure("Advanced font settings", gowt.Collapsed(), gowt.GrowX(),
			card(
				setting("Ligatures", "Use the terminal font's programming ligatures.",
					gowt.Switch("", gowt.Checked(true))))),
		heading("Colors"),
		card(
			setting("Improve text contrast", "Adjust unreadable text against its background.",
				gowt.Switch("", gowt.Checked(true))),
			setting("Force theme palette for text", "Map text colors to the nearest theme color.",
				gowt.Switch("")),
			setting("Use Display P3 color space", "Read theme and program colors as wide-gamut Display P3.",
				gowt.Switch(""))),
		heading("Cursor"))
}

func serverPage(app *gowt.App) gowt.Option {
	status := gowt.HBox(gowt.Gap(gowt.Spacing.SM),
		gowt.Label("", gowt.Place(gowt.AlignStart, gowt.AlignCenter),
			gowt.Init(func(l *gowt.LabelView) {
				themedSymbol(app, pathDot, 10, green, l.SetImage)
			})),
		label("Running", gowt.Place(gowt.AlignStart, gowt.AlignCenter)))
	return page(
		card(
			setting("Status", "", status),
			setting("Port", "", gowt.Text(gowt.Value("7000"), gowt.Size(gowt.Scaled(80), 0))),
			setting("Launch at login", "", gowt.Switch("", gowt.Checked(false)))),
		gowt.HBox(gowt.GrowX(), gowt.Spacer(), gowt.Button("Restart"), gowt.Button("Stop")))
}

func interfacePage(app *gowt.App) gowt.Option {
	return page(
		card(
			setting("Appearance", "",
				gowt.Segmented([]gowt.Segment{{Text: "System"}, {Text: "Light"}, {Text: "Dark"}},
					gowt.Selected(2), gowt.ID("appearance"),
					gowt.OnSelect(func(i int) {
						setAppearance(app, []gowt.Appearance{gowt.AppearanceSystem, gowt.AppearanceLight, gowt.AppearanceDark}[i])
					}))),
			setting("Sidebar icon size", "", gowt.Spinner(12, 32, gowt.Value(18))),
			setting("Show toolbar", "", gowt.Switch("", gowt.Checked(true))),
			setting("Reduce motion", "", gowt.Switch(""))))
}

func themesPage() gowt.Option {
	return page(
		card(
			setting("Theme", "",
				gowt.Segmented([]gowt.Segment{{Text: "Rexie"}, {Text: "Graphite"}, {Text: "Paper"}},
					gowt.Selected(0))),
			setting("Match system appearance", "", gowt.Switch("", gowt.Checked(true))),
			setting("Transparency", "", gowt.Switch("", gowt.Checked(true)))))
}

func shortcutsPage() gowt.Option {
	key := func(text string) gowt.Option {
		return label(text, faint(), gowt.Typeface(gowt.Type.Caption),
			gowt.Place(gowt.AlignEnd, gowt.AlignCenter))
	}
	return page(
		card(
			setting("New session", "Open another terminal session.", key("Cmd N")),
			setting("Clear screen", "Clear the current terminal buffer.", key("Cmd K")),
			setting("Split pane", "Split the active terminal pane.", key("Cmd D")),
			setting("Open settings", "Open this Preferences window.", key("Cmd ,"))))
}

func integrationsPage() gowt.Option {
	return page(
		card(
			setting("GitHub", "Issues and pull requests in the sidebar.", gowt.Button("Connect")),
			setting("Slack", "Build and session notifications.", gowt.Button("Connect")),
			setting("Linear", "Issue tracking for the current project.", gowt.Button("Connect"))))
}

func notificationsPage() gowt.Option {
	return page(
		card(
			setting("Session finished", "Notify when a long session ends.",
				gowt.Switch("", gowt.Checked(true))),
			setting("Build failed", "Notify when a build fails.",
				gowt.Switch("", gowt.Checked(true))),
			setting("Play a sound", "Play a sound with notifications.", gowt.Switch(""))))
}

func developerPage() gowt.Option {
	return page(
		card(
			setting("Remote control", "Let other clients control this app.",
				gowt.Switch("", gowt.Checked(true))),
			setting("Log level", "",
				gowt.Combo([]string{"Info", "Debug", "Trace"}, gowt.ReadOnly(), gowt.Selected(0))),
			gowt.HBox(gowt.GrowX(), gowt.Button("Open data folder"))))
}
