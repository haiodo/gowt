package icons

import (
	"fmt"
	"image"
	"sort"
	"sync"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons/lucide"
	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

// iconEntry maps one gowt icon name to each platform's own set. The names are the freedesktop
// canon; sf and fluent are SF Symbols names and Segoe Fluent Icons / MDL2 code points, looked
// up at run time on their own OS only (never embedded). lucide is the embedded fallback.
type iconEntry struct {
	fd     string
	sf     string
	fluent rune
	lucide string
}

var iconDict = map[string]iconEntry{
	"add":      {"list-add", "plus", 0xE710, "plus"},
	"remove":   {"list-remove", "minus", 0xE738, "minus"},
	"search":   {"edit-find", "magnifyingglass", 0xE721, "search"},
	"settings": {"preferences-system", "gearshape", 0xE713, "settings"},
	"close":    {"window-close", "xmark", 0xE711, "x"},
	"check":    {"object-select", "checkmark", 0xE73E, "check"},
	"edit":     {"document-edit", "pencil", 0xE70F, "pencil"},
	"delete":   {"edit-delete", "trash", 0xE74D, "trash"},
	"save":     {"document-save", "square.and.arrow.down", 0xE74E, "save"},
	"open":     {"document-open", "folder", 0xE8E5, "folder-open"},
	"refresh":  {"view-refresh", "arrow.clockwise", 0xE72C, "refresh-cw"},
	"back":     {"go-previous", "chevron.left", 0xE72B, "arrow-left"},
	"forward":  {"go-next", "chevron.right", 0xE72A, "arrow-right"},
	"up":       {"go-up", "chevron.up", 0xE74A, "arrow-up"},
	"down":     {"go-down", "chevron.down", 0xE74B, "arrow-down"},
	"home":     {"go-home", "house", 0xE80F, "house"},
	"info":     {"dialog-information", "info.circle", 0xE946, "info"},
	"warning":  {"dialog-warning", "exclamationmark.triangle", 0xE7BA, "triangle-alert"},
	"error":    {"dialog-error", "exclamationmark.circle", 0xEA39, "circle-alert"},
	"help":     {"help-about", "questionmark.circle", 0xE897, "circle-question-mark"},
	"copy":     {"edit-copy", "doc.on.doc", 0xE8C8, "copy"},
	"paste":    {"edit-paste", "doc.on.clipboard", 0xE77F, "clipboard"},
	"cut":      {"edit-cut", "scissors", 0xE8C6, "scissors"},
	"undo":     {"edit-undo", "arrow.uturn.backward", 0xE7A7, "undo-2"},
	"redo":     {"edit-redo", "arrow.uturn.forward", 0xE7A6, "redo-2"},
	"print":    {"document-print", "printer", 0xE749, "printer"},
	"folder":   {"folder", "folder", 0xE8B7, "folder"},
	"file":     {"text-x-generic", "doc", 0xE7C3, "file"},
	"star":     {"starred", "star", 0xE734, "star"},
	"heart":    {"emblem-favorite", "heart", 0xEB51, "heart"},
	"menu":     {"open-menu", "line.3.horizontal", 0xE700, "menu"},
	"filter":   {"view-filter", "line.3.horizontal.decrease", 0xE71C, "funnel"},
	"download": {"folder-download", "arrow.down.to.line", 0xE896, "download"},
	"upload":   {"folder-upload", "arrow.up.to.line", 0xE898, "upload"},
	"user":     {"avatar-default", "person", 0xE77B, "user"},
	"lock":     {"system-lock-screen", "lock", 0xE72E, "lock"},
	"play":     {"media-playback-start", "play.fill", 0xE768, "play"},
	"pause":    {"media-playback-pause", "pause.fill", 0xE769, "pause"},
	"stop":     {"media-playback-stop", "stop.fill", 0xE71A, "square"},
}

// Names lists the names Get accepts, sorted.
func Names() []string {
	names := make([]string, 0, len(iconDict))
	for n := range iconDict {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Option adjusts Get.
type Option func(*iconConfig)

type iconConfig struct {
	size  int
	color *gowt.RGB
}

// Size sets the icon's side in points (default 16).
func Size(points int) Option { return func(c *iconConfig) { c.size = points } }

// Color sets the tint; by default light gray in dark mode and near-black in light mode.
func Color(c gowt.RGB) Option { return func(ic *iconConfig) { ic.color = &c } }

// Get returns the named icon from the system set of the current OS (SF Symbols, Segoe Fluent
// Icons, freedesktop symbolic icons), or from the embedded Lucide set when the system has none.
// The result is drawn at every zoom and tinted for the theme at call time: after
// App.OnThemeChange call Get again (the old image keeps its colour). Images are cached per
// name, size and colour: the App owns them, the caller must not Dispose them. It panics on an
// unknown name; Names lists the valid ones.
func Get(a *gowt.App, name string, opts ...Option) *gowt.Image {
	e, ok := iconDict[name]
	if !ok {
		panic(fmt.Errorf("icons: unknown icon %q", name))
	}
	cfg := iconConfig{size: 16}
	for _, o := range opts {
		o(&cfg)
	}
	cfg.size = max(cfg.size, 1)
	c := gowt.RGB{R: 30, G: 30, B: 30}
	if a.Dark() {
		c = gowt.RGB{R: 235, G: 235, B: 235}
	}
	if cfg.color != nil {
		c = *cfg.color
	}
	key := iconKey{name, cfg.size, c}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if img, ok := cache[a][key]; ok {
		return img
	}
	src, _ := lucide.SVG(e.lucide)
	p := &iconProvider{a: a, e: e, size: cfg.size, c: c,
		fallback: svg.NewImageDataProvider(src, int32(cfg.size), int32(cfg.size), fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))}
	img := a.ImageFromProvider(p)
	if cache[a] == nil {
		cache[a] = map[iconKey]*gowt.Image{}
	}
	cache[a][key] = img
	return img
}

type iconKey struct {
	name string
	size int
	c    gowt.RGB
}

// The images belong to their App and are disposed when its Run returns, but the entries stay: one
// Run per process is the ceiling; the way out is an end-of-run hook on App.
var (
	cacheMu sync.Mutex
	cache   = map[*gowt.App]map[iconKey]*gowt.Image{}
)

type iconProvider struct {
	a        *gowt.App
	e        iconEntry
	size     int
	c        gowt.RGB
	fallback swt.ImageDataProvider
}

func (p *iconProvider) GetImageData(zoom int32) *swt.ImageData {
	px := p.size * int(zoom) / 100
	if d := systemIcon(p.a, p.e, px, p.c); d != nil {
		return d
	}
	return p.fallback.GetImageData(zoom)
}

// tintAlpha makes a px x px NRGBA image of colour c from an alpha mask given row by row.
func tintAlpha(px int, c gowt.RGB, alpha func(x, y int) uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, alpha(x, y)
		}
	}
	return img
}
