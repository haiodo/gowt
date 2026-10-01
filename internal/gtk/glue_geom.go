//go:build linux

package gtk

// GeometryInterface is what Shell needs of a window geometry: the GTK 3 GdkGeometry hints plus the
// state SWT tracks next to them.
type GeometryInterface interface {
	GetMinWidth() int32
	GetMinHeight() int32
	GetMaxWidth() int32
	GetMaxHeight() int32
	GetResize() bool
	GetRequestedWidth() int32
	GetRequestedHeight() int32
	GetMinSizeRequested() bool
	SetMinWidth(int32)
	SetMinHeight(int32)
	SetMaxWidth(int32)
	SetMaxHeight(int32)
	SetResize(bool)
	SetRequestedWidth(int32)
	SetRequestedHeight(int32)
	SetMinSizeRequested(bool)
}

// GdkGeometry starts with the C struct of that name (Gdk-3.0: eight gint hints, two gdouble
// aspect ratios, the gravity), so a pointer to it goes straight to gtk_window_set_geometry_hints.
type GdkGeometry struct {
	Min_width, Min_height, Max_width, Max_height   int32
	Base_width, Base_height, Width_inc, Height_inc int32
	Min_aspect, Max_aspect                         float64
	Win_gravity                                    int32
	_                                              int32
	resize                                         bool
	minSizeRequested                               bool
	requestedWidth, requestedHeight                int32
}

func NewGdkGeometry() *GdkGeometry { return &GdkGeometry{} }

func (g *GdkGeometry) GetMinWidth() int32         { return g.Min_width }
func (g *GdkGeometry) GetMinHeight() int32        { return g.Min_height }
func (g *GdkGeometry) GetMaxWidth() int32         { return g.Max_width }
func (g *GdkGeometry) GetMaxHeight() int32        { return g.Max_height }
func (g *GdkGeometry) GetResize() bool            { return g.resize }
func (g *GdkGeometry) GetRequestedWidth() int32   { return g.requestedWidth }
func (g *GdkGeometry) GetRequestedHeight() int32  { return g.requestedHeight }
func (g *GdkGeometry) GetMinSizeRequested() bool  { return g.minSizeRequested }
func (g *GdkGeometry) SetMinWidth(v int32)        { g.Min_width = v }
func (g *GdkGeometry) SetMinHeight(v int32)       { g.Min_height = v }
func (g *GdkGeometry) SetMaxWidth(v int32)        { g.Max_width = v }
func (g *GdkGeometry) SetMaxHeight(v int32)       { g.Max_height = v }
func (g *GdkGeometry) SetResize(v bool)           { g.resize = v }
func (g *GdkGeometry) SetRequestedWidth(v int32)  { g.requestedWidth = v }
func (g *GdkGeometry) SetRequestedHeight(v int32) { g.requestedHeight = v }
func (g *GdkGeometry) SetMinSizeRequested(v bool) { g.minSizeRequested = v }
