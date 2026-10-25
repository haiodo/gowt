# Writing one layout for macOS, Windows and Linux

The same numbers do not mean the same size on the three systems. The system font is 13 px on macOS, 12 px (Segoe UI 9 pt) on Windows and 14.7 px (11 pt) on GNOME, and a font size in points becomes pixels at 1 pt = 1 px, 4/3 px and 4/3 px. A layout tuned on a Mac then has text 13% too big for its paddings on Linux, and too small on Windows. Sizes from tokens fix that: you name a role (`Type.Title`, `Spacing.RowHeight`) and gowt gives the size for the OS.

## The tokens

```go
func(app *gowt.App) {
	app.SetDensity(gowt.DensityNative) // first, before any widget and before tuning a token
	w := app.Window("Settings",
		gowt.Label("Appearance", gowt.Typeface(gowt.Type.Title)),
		gowt.Label("Applies to new windows", gowt.Typeface(gowt.Type.Caption)),
		gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.Button("Cancel"), gowt.Button("Save")))
	w.Show()
}
```

`gowt.Type` is seven `Font` values, sized in logical pixels (`Font.Px`, a float): `Caption`, `Body`, `BodyStrong`, `Subtitle`, `Title`, `LargeTitle`, `Mono`. Use them with `Typeface` or `SetFont`. After `SetDensity`, a widget without a `Typeface` has the body font; without it, the system font.

`gowt.Spacing` has the gaps `XS SM MD LG XL`, `PageMargin`, and the sizes `RowHeight` (list, table, tree), `ControlHeight` (button, text field, combo), `SidebarRow`, `SidebarWidth`, `InspectorWidth`. `look.SetSidebar` and `look.SetInspector` scale their minimum widths, the toolbar padding and the toolbar title with the density; pass `gowt.Spacing.SidebarWidth` as the width. For a size that has no token, `gowt.Scaled(240)` multiplies by the density and rounds to 4 px.

Tune a token after `SetDensity`, once at startup: `gowt.Spacing.RowHeight = 40`. `SetDensity` replaces `Type` and `Spacing` as a whole, and widgets already built keep their sizes.

## Native or Same

| | `DensityNative` | `DensitySame` (= `DensityFactor(1)`) | no `SetDensity` (this release) |
|---|---|---|---|
| Body text | the OS: 13 px macOS, 14 px Windows 11 (Segoe UI Variable Text), the GTK UI font on Linux | 13 px everywhere, the system family | the system font; `Type` follows `DensityNative` |
| Spacing | macOS value times `body / 13`, rounded to 4 px | the macOS value | the values of earlier releases |
| Result | each OS looks like its own apps; the window is bigger on Linux and Windows | the layout is the same size on all three | unchanged |

`DensityFactor(1.2)` is `DensitySame` with a 15.6 px body and 1.2 times the spacing.

Pick `DensityNative` unless you compare screenshots across the three systems or ship a design with a fixed grid; then pick `DensitySame`.

Native density is opt-in in this release. Without `SetDensity` nothing changes in an existing app, and it becomes the default in a later minor release, which the CHANGELOG announces.

## Resolved values

Values under `DensityNative`, as printed on Windows 11 arm64 and on Ubuntu 26.04 (GNOME, Adwaita Sans 11, Ubuntu Sans Mono 11). The Linux body follows `gtk-font-name`, so the column moves with the user's font.

| Token | macOS | Windows 11 | Linux GNOME 11 pt |
|---|---|---|---|
| Caption | 11 | 12 | 12.0 (0.82 of the body) |
| Body, BodyStrong | 13 | 14 | 14.7 |
| Subtitle | 15 | 20 | 17.6 (1.2) |
| Title | 22 | 28 (Segoe UI Variable Display) | 20.5 (1.4) |
| LargeTitle | 26 | 40 (Segoe UI Variable Display) | 26.4 (1.8) |
| Mono, at the body size | Menlo | Cascadia Mono, else Consolas | `monospace-font-name` of GSettings |
| Gap SM / MD / LG | 8 / 12 / 16 | 8 / 12 / 16 | 8 / 12 / 20 |
| PageMargin | 16 | 16 | 20 |
| ControlHeight | 28 | 32 | 32 |
| RowHeight | 32 | 36 | 36 |
| SidebarRow | 28 | 32 | 32 |
| SidebarWidth | 220 | 236 | 248 |

Title, Subtitle, BodyStrong and LargeTitle are Semibold. The values are the base ones times the factor and rounded to 4 px, so a step can jump: at a factor of 1.13, LG 16 becomes 20.

## Idioms from examples/apps

- `Scaled` rounds to 4 px even at factor 1, so `Scaled(26)` is 28 on macOS. Write multiples of 4 (`Scaled(28)`) when the macOS size must stay as designed.
- Table and Tree column widths, avatar and badge sizes, and fixed field widths go through `Scaled`. Let a column follow its content with `Fit()` instead of guessing a width.
- A token in another weight is a copy: `f := gowt.Type.Caption; f.Weight = gowt.Semibold`.
- Icon sizes (`icons.Size(18)`) are not scaled by `Scaled` in these apps; they stay 18.
- A `Canvas` callback has `g.Size()`, the drawable size in points, so a chart fills its cell; take the margins from `g.TextSize` of the label font (`Type.Caption`), not from numbers.

## Controls that differ by design

- Primary and destructive buttons: `gowt.Primary()` and `gowt.Destructive()` take the system's own look, see [widgets.md](widgets.md#primary-and-destructive-buttons). Do not colour a button by hand with `Foreground`.
- Form labels: `gowt.Fields` puts the labels trailing on macOS and leading on Windows and GNOME, as each OS does, see [widgets.md](widgets.md#forms). A hand-made Grid row aligns them one way on all three.
- Line numbers: `gowt.LineNumbers()` on a StyledText; the gutter is made of SWT bullets drawn with the lines, so it scrolls with the text and follows its font and the theme.

## Forcing dark or light

`app.SetAppearance(gowt.AppearanceDark | AppearanceLight | AppearanceSystem)` makes `app.Dark()` and the `Adaptive` colours follow the choice and runs `OnThemeChange`. Use it for screenshots and for an app with its own scheme setting; `kit.Appearance` in `examples/apps` reads `GOWT_APPEARANCE=dark|light`.

| | What switches | Live | Verified |
|---|---|---|---|
| macOS | everything, through `NSApp.appearance` (controls, title bars, sheets, menus) | yes | not run on a Mac yet |
| Linux GTK | `gtk-application-prefer-dark-theme`, and the portal no longer overrides it; needs a theme with a dark variant (Adwaita, Yaru) | yes | Docker stand (Adwaita), Ubuntu 26.04 GNOME |
| Windows | title bars; scroll bars, tables, trees and buttons in the dark controls theme (the undocumented uxtheme path of `look.SetDarkContent`); the controls gowt paints itself read the dark flag of the same switch (the Fluent palette) | title bars yes; content only for windows created after the call, so call it first | Windows 11 arm64 |

On Windows dark content is partial (wave 3 of the look plan): the window background and the StyledText stay light next to the dark scroll bars and title bar, as the editor app shows on Windows 11; other controls were not checked one by one. Do not use `AppearanceDark` for a Windows screenshot of a dark design yet. `AppearanceSystem` after a force on Windows keeps the content theme as it is; restart to get it back.

## What to avoid

- A fixed pixel width for text: `Size(120, 0)` for a label or a button that holds a caption. At density 1.13 or with a long translation the caption is cut off. Let the layout take the natural width, or write `Size(gowt.Scaled(120), 0)`.
- An absolute font size: `Font{Size: 12}` is points, so it is 12 px on macOS and 16 px elsewhere, and it ignores the density. Use a `Type` token, or `Font{Scale: 1.2}` for a relative size.
- Hard-coded gaps and heights: `Gap(10)`, `Padding(14)`, a row height of 24. Take `Spacing.*`.
- A token read before `Run`: `gowt.Type` is empty until `Run` fills it. Build the options inside the setup function.
- Fixed pixel math that assumes the font height, for a canvas for example: measure with `GC.TextSize`.

## Before and after

Before: fits on a Mac, cut off on Linux, small on Windows.

```go
gowt.Label("Appearance", gowt.Typeface(gowt.Font{Size: 18, Weight: gowt.Semibold})),
gowt.HBox(gowt.Gap(10), gowt.Padding(14),
	gowt.Label("Name"), gowt.Text(gowt.Size(160, 0))),
```

After: the title is the OS's title size, the gaps follow the density, the text field takes the width of 160 points scaled.

```go
app.SetDensity(gowt.DensityNative)
...
gowt.Label("Appearance", gowt.Typeface(gowt.Type.Title)),
gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.Padding(gowt.Spacing.LG),
	gowt.Label("Name"), gowt.Text(gowt.Size(gowt.Scaled(160), 0))),
```

To check a screen, run it with `DensityNative` and with `DensitySame` on each OS and compare the screenshots (see [screenshots.md](screenshots.md)).
