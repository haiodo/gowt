package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// defaultCSS is at GTK's application priority: above the theme, below an app's USER provider and SWT's
// per-widget ones. It uses theme named colours only, so Adwaita, Yaru, light and dark follow.
const defaultCSS = `
treeview.view header button {
	background-image: none;
	background-color: transparent;
	border-style: solid;
	border-width: 0 0 1px 0;
	border-color: @borders;
	border-radius: 0;
	box-shadow: none;
	padding: 6px 8px;
}
treeview.view header button:hover { background-color: alpha(@theme_fg_color, 0.06); }
treeview.view header button:active { background-color: alpha(@theme_fg_color, 0.12); }
treeview.view { -GtkTreeView-vertical-separator: 6; }
frame.gowt-group > label { font-weight: 600; padding: 0 2px 6px 2px; }
frame.gowt-group > border {
	background-color: alpha(@theme_fg_color, 0.05);
	border: 1px solid alpha(@theme_fg_color, 0.12);
	border-radius: 10px;
}
`

// sidebarCSS must outrank the portal accent rule (USER), or the accent colours the sidebar selection.
// look.SetSidebar puts gowt-sidebar on the sidebar panel.
const sidebarCSS = `
.gowt-sidebar treeview.view:selected,
.gowt-sidebar treeview.view:selected:focus,
.gowt-sidebar treeview.view:selected:backdrop {
	background-color: alpha(@theme_fg_color, 0.12);
	color: @theme_fg_color;
	border-radius: 6px;
}
.gowt-sidebar treeview.view,
.gowt-sidebar treeview.view:backdrop {
	background-color: transparent;
	-GtkTreeView-horizontal-separator: 20;
}
`

func installStyle() {
	gtk.AddDefaultCSS(defaultCSS)
	gtk.AddOverrideCSS(sidebarCSS)
}

// groupCard puts the Group title above a card instead of in the frame line.
func groupCard(g *swt.Group) {
	gtk.AddStyleClass(g.Handle, "gowt-group")
	gtk.SetFrameLabelAlign(g.Handle, 0, 1)
}

// scrollOverlay makes the scroll bars of c overlay bars that show on overflow, as on Multiline text.
func scrollOverlay(c interface{ SetScrollbarsMode(int32) }) {
	c.SetScrollbarsMode(swt.SCROLLBAR_OVERLAY)
}
