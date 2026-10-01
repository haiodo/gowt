//go:build linux

package gtk

import (
	"sync"
	"unsafe"
)

// SwtFixed: a GtkContainer placing children at the x/y/size SWT sets, preferred size 0x0 (SWT lays
// out itself), and a GtkScrollable whose adjustments are only stored, since GtkScrolledWindow
// refuses a child that is not scrollable.

type fixedChild struct {
	w                   int64
	x, y, width, height int32
}

type fixedState struct {
	children   []*fixedChild
	hadj, vadj int64
	hpol, vpol int32
}

var (
	fixedMu     sync.Mutex
	fixedStates = map[int64]*fixedState{}
	fixedType   int64
)

func fixedOf(obj int64) *fixedState {
	fixedMu.Lock()
	defer fixedMu.Unlock()
	s := fixedStates[obj]
	if s == nil {
		s = &fixedState{}
		fixedStates[obj] = s
	}
	return s
}

func (s *fixedState) find(w int64) int {
	for i, c := range s.children {
		if c.w == w {
			return i
		}
	}
	return -1
}

var (
	gTypeRegisterStatic     = lz[func(uintptr, unsafe.Pointer, unsafe.Pointer, int32) uintptr]{sym: "g_type_register_static"}
	gTypeAddInterfaceStatic = lz[func(uintptr, uintptr, unsafe.Pointer)]{sym: "g_type_add_interface_static"}
	gtkContainerGetType     = lz[func() uintptr]{sym: "gtk_container_get_type"}
	gtkScrollableGetType    = lz[func() uintptr]{sym: "gtk_scrollable_get_type"}
	gtkWidgetGetType        = lz[func() uintptr]{sym: "gtk_widget_get_type"}
	gObjectOverrideProp     = lz[func(uintptr, uint32, unsafe.Pointer)]{sym: "g_object_class_override_property"}
	gObjectRef              = lz[func(uintptr) uintptr]{sym: "g_object_ref"}
	gObjectUnref            = lz[func(uintptr)]{sym: "g_object_unref"}
	gObjectWeakRef          = lz[func(uintptr, uintptr, uintptr)]{sym: "g_object_weak_ref"}
	gValueGetObject         = lz[func(uintptr) uintptr]{sym: "g_value_get_object"}
	gValueSetObject         = lz[func(uintptr, uintptr)]{sym: "g_value_set_object"}
	gValueGetEnum           = lz[func(uintptr) int32]{sym: "g_value_get_enum"}
	gValueSetEnum           = lz[func(uintptr, int32)]{sym: "g_value_set_enum"}

	gtkWidgetSetParent        = lz[func(uintptr, uintptr)]{sym: "gtk_widget_set_parent"}
	gtkWidgetUnparent         = lz[func(uintptr)]{sym: "gtk_widget_unparent"}
	gtkWidgetSizeAllocate     = lz[func(uintptr, unsafe.Pointer)]{sym: "gtk_widget_size_allocate"}
	gtkWidgetSetAllocation    = lz[func(uintptr, unsafe.Pointer)]{sym: "gtk_widget_set_allocation"}
	gtkWidgetGetPreferredSize = lz[func(uintptr, unsafe.Pointer, unsafe.Pointer)]{sym: "gtk_widget_get_preferred_size"}
	gtkWidgetGetVisible       = lz[func(uintptr) int32]{sym: "gtk_widget_get_visible"}
	gtkWidgetGetRealized      = lz[func(uintptr) int32]{sym: "gtk_widget_get_realized"}
	gtkWidgetGetHasWindow     = lz[func(uintptr) int32]{sym: "gtk_widget_get_has_window"}
	gtkWidgetGetWindow        = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_window"}
	gtkWidgetQueueResize      = lz[func(uintptr)]{sym: "gtk_widget_queue_resize"}
	gdkWindowMoveResize       = lz[func(uintptr, int32, int32, int32, int32)]{sym: "gdk_window_move_resize"}
	gdkWindowNew              = lz[func(uintptr, unsafe.Pointer, int32) uintptr]{sym: "gdk_window_new"}
	gdkWindowDestroy          = lz[func(uintptr)]{sym: "gdk_window_destroy"}
	gtkWidgetSetRealized      = lz[func(uintptr, int32)]{sym: "gtk_widget_set_realized"}
	gtkWidgetSetWindow        = lz[func(uintptr, uintptr)]{sym: "gtk_widget_set_window"}
	gtkWidgetRegisterWindow   = lz[func(uintptr, uintptr)]{sym: "gtk_widget_register_window"}
	gtkWidgetUnregisterWindow = lz[func(uintptr, uintptr)]{sym: "gtk_widget_unregister_window"}
	gtkWidgetGetParentWindow  = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_parent_window"}
	gtkWidgetGetVisual        = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_visual"}
	gtkWidgetGetEvents        = lz[func(uintptr) int32]{sym: "gtk_widget_get_events"}
	gtkWidgetGetAllocation    = lz[func(uintptr, unsafe.Pointer)]{sym: "gtk_widget_get_allocation"}
	gTypeClassPeekParent      = lz[func(uintptr) uintptr]{sym: "g_type_class_peek_parent"}
	gSignalNew                = lz[func(unsafe.Pointer, uintptr, int32, uint32, uintptr, uintptr, uintptr, uintptr, uint32) uint32]{sym: "g_signal_new"}
)

// Constants from GDK's window creation API (GdkWindowType, GdkWindowWindowClass, GdkWindowAttributesType).
const (
	gdkWindowChild  = 2
	gdkInputOutput  = 0
	gdkExposureMask = 1 << 1
	gdkWAX          = 1 << 2
	gdkWAY          = 1 << 3
	gdkWAVisual     = 1 << 5
)

var fixedParentClass int64

const (
	fixedHAdj = 1 + iota
	fixedVAdj
	fixedHPolicy
	fixedVPolicy
)

func fixedClassInit(klass, _ uintptr) {
	fixedParentClass = int64(gTypeClassPeekParent.get()(klass))
	c := (*GtkContainerClass)(at(int64(klass)))
	c.Realize = NewCallbackAny(fixedRealize)
	c.Set_property = NewCallbackAny(fixedSetProperty)
	c.Get_property = NewCallbackAny(fixedGetProperty)
	c.Size_allocate = NewCallbackAny(fixedSizeAllocate)
	c.Get_preferred_width = NewCallbackAny(fixedPreferred)
	c.Get_preferred_height = NewCallbackAny(fixedPreferred)
	c.Add = NewCallbackAny(fixedAddVfunc)
	c.Remove = NewCallbackAny(fixedRemoveVfunc)
	c.Forall = NewCallbackAny(fixedForall)
	c.Child_type = NewCallbackAny(func(uintptr) uintptr { return gtkWidgetGetType.get()() })
	for i, n := range []string{"hadjustment", "vadjustment", "hscroll-policy", "vscroll-policy"} {
		gObjectOverrideProp.get()(klass, uint32(fixedHAdj+i), sp(cs(n)))
	}
}

// fixedRealize gives the widget its own GdkWindow when it has_window (GtkWidget's default realize
// only handles windowless widgets); the default unrealize destroys that window again.
func fixedRealize(widget uintptr) {
	if gtkWidgetGetHasWindow.get()(widget) == 0 {
		cCall((*GtkContainerClass)(at(fixedParentClass)).Realize, widget)
		return
	}
	gtkWidgetSetRealized.get()(widget, 1)
	var a GdkRectangle
	gtkWidgetGetAllocation.get()(widget, unsafe.Pointer(&a))
	attr := GdkWindowAttr{
		Window_type: gdkWindowChild, Wclass: gdkInputOutput,
		X: a.X, Y: a.Y, Width: a.Width, Height: a.Height,
		Visual:     int64(gtkWidgetGetVisual.get()(widget)),
		Event_mask: gtkWidgetGetEvents.get()(widget) | gdkExposureMask,
	}
	win := gdkWindowNew.get()(gtkWidgetGetParentWindow.get()(widget), unsafe.Pointer(&attr), gdkWAX|gdkWAY|gdkWAVisual)
	gtkWidgetSetWindow.get()(widget, win)
	gtkWidgetRegisterWindow.get()(widget, win)
}

func fixedSetProperty(obj uintptr, id uint32, value, _ uintptr) {
	s := fixedOf(int64(obj))
	switch id {
	case fixedHAdj, fixedVAdj:
		p := &s.hadj
		if id == fixedVAdj {
			p = &s.vadj
		}
		n := int64(gValueGetObject.get()(value))
		if n != 0 {
			gObjectRef.get()(uintptr(n))
		}
		if *p != 0 {
			gObjectUnref.get()(uintptr(*p))
		}
		*p = n
	case fixedHPolicy:
		s.hpol = gValueGetEnum.get()(value)
	case fixedVPolicy:
		s.vpol = gValueGetEnum.get()(value)
	}
}

func fixedGetProperty(obj uintptr, id uint32, value, _ uintptr) {
	s := fixedOf(int64(obj))
	switch id {
	case fixedHAdj:
		gValueSetObject.get()(value, uintptr(s.hadj))
	case fixedVAdj:
		gValueSetObject.get()(value, uintptr(s.vadj))
	case fixedHPolicy:
		gValueSetEnum.get()(value, s.hpol)
	case fixedVPolicy:
		gValueSetEnum.get()(value, s.vpol)
	}
}

func fixedPreferred(_ uintptr, min, nat uintptr) {
	*(*int32)(at(int64(min))) = 0
	*(*int32)(at(int64(nat))) = 0
}

func fixedSizeAllocate(widget uintptr, alloc uintptr) {
	a := (*GdkRectangle)(at(int64(alloc)))
	gtkWidgetSetAllocation.get()(widget, at(int64(alloc)))
	hasWindow := gtkWidgetGetHasWindow.get()(widget) != 0
	if hasWindow && gtkWidgetGetRealized.get()(widget) != 0 {
		gdkWindowMoveResize.get()(gtkWidgetGetWindow.get()(widget), a.X, a.Y, a.Width, a.Height)
	}
	var ox, oy int32
	if !hasWindow {
		ox, oy = a.X, a.Y
	}
	s := fixedOf(int64(widget))
	for _, c := range append([]*fixedChild(nil), s.children...) {
		if gtkWidgetGetVisible.get()(uintptr(c.w)) == 0 {
			continue
		}
		var req [4]int32 // minimum and natural GtkRequisition; computing them first keeps GTK from warning
		gtkWidgetGetPreferredSize.get()(uintptr(c.w), unsafe.Pointer(&req[0]), unsafe.Pointer(&req[2]))
		r := GdkRectangle{X: ox + c.x, Y: oy + c.y, Width: c.width, Height: c.height}
		if r.Width <= 0 || r.Height <= 0 {
			r.Width, r.Height = req[2], req[3]
		}
		gtkWidgetSizeAllocate.get()(uintptr(c.w), unsafe.Pointer(&r))
	}
}

func fixedAddVfunc(container, w uintptr) { OSSwt_fixed_add(int64(container), int64(w)) }

func fixedRemoveVfunc(container, w uintptr) { OSSwt_fixed_remove(int64(container), int64(w)) }

func fixedForall(container uintptr, _ int32, cb, data uintptr) {
	for _, c := range append([]*fixedChild(nil), fixedOf(int64(container)).children...) {
		cCall(int64(cb), uintptr(c.w), data)
	}
}

func fixedWeakGone(_, obj uintptr) {
	fixedMu.Lock()
	delete(fixedStates, int64(obj))
	fixedMu.Unlock()
}

var fixedInit sync.Once

func fixedRegister() {
	info := GTypeInfo{
		Class_size:    int16(unsafe.Sizeof(GtkContainerClass{})),
		Class_init:    NewCallbackAny(fixedClassInit),
		Instance_size: int16(unsafe.Sizeof(GtkContainer{})),
	}
	t := gTypeRegisterStatic.get()(gtkContainerGetType.get()(), sp(cs("SwtFixed")), unsafe.Pointer(&info), 0)
	// SWT connects to "dpi-changed" on every widget; GTK 3 has no such signal, so give GtkWidget a
	// do-nothing one (nothing emits it) instead of a critical warning per connect.
	gSignalNew.get()(sp(cs("dpi-changed")), gtkWidgetGetType.get()(), 1, 0, 0, 0, 0, 1<<2, 0)
	var iface GInterfaceInfo
	gTypeAddInterfaceStatic.get()(t, gtkScrollableGetType.get()(), unsafe.Pointer(&iface))
	fixedType = int64(t)
}

func OSSwt_fixed_get_type() int64 {
	fixedInit.Do(fixedRegister)
	return fixedType
}

// fixedParent returns the state of container, or nil when it is not a SwtFixed (SWT also passes
// the parent of arbitrary handles).
func fixedParent(container int64) *fixedState {
	if container == 0 || !isA(container, fixedType) {
		return nil
	}
	return fixedOf(container)
}

var fixedWeak int64

func OSSwt_fixed_add(container, widget int64) {
	s := fixedParent(container)
	if s == nil {
		return
	}
	if len(s.children) == 0 && fixedWeak == 0 {
		fixedWeak = NewCallbackAny(fixedWeakGone)
	}
	s.children = append(s.children, &fixedChild{w: widget})
	gtkWidgetSetParent.get()(uintptr(widget), uintptr(container))
	gtkWidgetQueueResize.get()(uintptr(container))
}

func OSSwt_fixed_remove(container, widget int64) {
	s := fixedParent(container)
	if s == nil {
		return
	}
	i := s.find(widget)
	if i < 0 {
		return
	}
	s.children = append(s.children[:i], s.children[i+1:]...)
	gtkWidgetUnparent.get()(uintptr(widget))
	gtkWidgetQueueResize.get()(uintptr(container))
}

func OSSwt_fixed_move(container, widget int64, x, y int32) {
	if s := fixedParent(container); s != nil {
		if i := s.find(widget); i >= 0 {
			s.children[i].x, s.children[i].y = x, y
			gtkWidgetQueueResize.get()(uintptr(container))
		}
	}
}

func OSSwt_fixed_resize(container, widget int64, width, height int32) {
	if s := fixedParent(container); s != nil {
		if i := s.find(widget); i >= 0 {
			s.children[i].width, s.children[i].height = width, height
			gtkWidgetQueueResize.get()(uintptr(container))
		}
	}
}

// OSSwt_fixed_restack puts widget just above or below sibling (zero: top or bottom). The list runs
// topmost first because getChildren reads it through forall; SWT restacks the GdkWindows itself.
func OSSwt_fixed_restack(container, widget, sibling int64, above bool) {
	s := fixedParent(container)
	if s == nil {
		return
	}
	i := s.find(widget)
	if i < 0 {
		return
	}
	c := s.children[i]
	s.children = append(s.children[:i], s.children[i+1:]...)
	pos := len(s.children)
	if above {
		pos = 0
	}
	if j := s.find(sibling); sibling != 0 && j >= 0 {
		pos = j
		if !above {
			pos = j + 1
		}
	}
	s.children = append(s.children[:pos], append([]*fixedChild{c}, s.children[pos:]...)...)
	gtkWidgetQueueResize.get()(uintptr(container))
}

// OSSwt_scaled_paintable_new belongs to the GTK 4 renderer; GTK 3 never reaches it.
func OSSwt_scaled_paintable_new(texture int64, width, height int32) int64 {
	panic("gowt/internal/gtk: scaled paintables need GTK 4")
}
