package gowt

import (
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// Option describes a widget at construction: a style, common, placement or box option, Ref,
// Shortcut, or a child. Every widget constructor (Label, Button, VBox, ...) returns an Option that
// creates the widget when its parent is created or when it is passed to Add. Only this package
// makes Options. An option the widget cannot use goes to App.OnError and the widget is created
// without it; the "Options:" line of a constructor lists the style options it reads.
type Option interface{ option() }

// style sets and clears style bits before the native widget is created; id names it for the check.
type style struct {
	id    styleID
	set   int32
	clear int32
}

// styleID is one bit per style option; a constructor passes the set it reads to finish.
type styleID uint32

const (
	sCheck styleID = 1 << iota
	sRadio
	sMultiline
	sPassword
	sReadOnly
	sBorder
	sCellSelect
	sNoBorder
	sSearch
	sMultiSelect
	sSimple
	sVertical
	sSmooth
	sIndeterminate
	sFlat
	sWrap
	sBottom
	sClosable
	sDropDown
	sRight
	sCenter
	sAsTime
	sAsCalendar
	sScrollbars
	sFit
)

var styleNames = [...]string{"Check", "Radio", "Multiline", "Password", "ReadOnly", "Border", "CellSelect",
	"NoBorder", "Search", "MultiSelect", "Simple", "Vertical", "Smooth", "Indeterminate", "Flat", "Wrap",
	"Bottom", "Closable", "DropDown", "Right", "Center", "AsTime", "AsCalendar", "Scrollbars", "Fit"}

func (s styleID) String() string {
	for i, n := range styleNames {
		if s == 1<<i {
			return n
		}
	}
	return fmt.Sprintf("style %#x", uint32(s))
}

// apply runs on the control right after it is created.
type apply func(*swt.Control)

// place adds to the layout data of the widget in its parent.
type place func(*placement)

// menuOpt adjusts a menu item (Shortcut).
type menuOpt func(*swt.MenuItem)

// boxOpt sets the gap or the padding of a container's own layout.
type boxOpt struct {
	padding bool
	n       int
}

// hook runs on the created widget (Ref, Init, events, initial values, bindings); it returns the
// error to report when w cannot take it.
type hook func(w Widget) error

// lateHook runs after the hooks: Rows and Nodes fill rows after Init has made the columns.
type lateHook func(w Widget) error

// node is a widget description: build creates the widget in h, applies its options and children,
// and returns it, or nil after reporting that h cannot hold it. Nothing of it stays after build.
type node struct {
	name  string
	build func(h host) Widget
}

func (style) option()    {}
func (apply) option()    {}
func (place) option()    {}
func (menuOpt) option()  {}
func (boxOpt) option()   {}
func (hook) option()     {}
func (lateHook) option() {}
func (*node) option()    {}

// host is a container that children are created in.
type host interface {
	Widget
	into() *swt.Composite // the native parent of a new child
	added(w Widget)       // runs after each child is created
}

// Ref stores the widget in *p once it and its children are created, for the code after the Add
// that creates it, which calls its methods. A Ref to a type other than the widget's is reported to App.OnError.
func Ref[T Widget](p *T) Option {
	return hook(func(w Widget) error {
		v, ok := w.(T)
		if !ok {
			return fmt.Errorf("gowt: Ref to a %T given to %s", v, nameOf(w.control()))
		}
		*p = v
		return nil
	})
}

// Init runs f on the widget once it and its children are created: a function that returns a
// description sets up its widget with it. A widget of another type than f takes goes to App.OnError.
func Init[T Widget](f func(T)) Option {
	return hook(func(w Widget) error {
		v, ok := w.(T)
		if !ok {
			return fmt.Errorf("gowt: Init for a %T given to %s", v, nameOf(w.control()))
		}
		f(v)
		return nil
	})
}

// Gap sets the space in points between the children of a container, rows and columns of a Grid alike.
// The default is Spacing.SM; Gap(0) puts the children edge to edge.
func Gap(points int) Option { return boxOpt{n: points} }

// Padding sets the space in points between the edge of a container and its children. The default is
// Spacing.PageMargin for a Window and a Tab, Spacing.SM for a Group and 0 elsewhere.
func Padding(points int) Option { return boxOpt{padding: true, n: points} }

// spacingOf returns the Gap and Padding given in opts, Spacing.SM and pad when absent.
func spacingOf(opts []Option, pad int) (gap, padding int) {
	gap, padding = Spacing.SM, pad
	for _, o := range opts {
		if b, ok := o.(boxOpt); ok {
			if b.padding {
				padding = b.n
			} else {
				gap = b.n
			}
		}
	}
	return gap, padding
}

func resolve(base int32, opts []Option) int32 {
	for _, o := range opts {
		if s, ok := o.(style); ok {
			base |= s.set
		}
	}
	for _, o := range opts {
		if s, ok := o.(style); ok {
			base &^= s.clear
		}
	}
	return base
}

// finish applies opts to w, a widget just created from the style options that allow lists, then
// creates the children when w is a container, then runs the hooks. Options w cannot use go to App.OnError.
func finish(w Widget, allow styleID, opts []Option) {
	c := w.control()
	h, isHost := w.(host)
	if a := appOf(c); a != nil && a.body != nil && c.GetName() != "Composite" {
		// The widget is new and not laid out yet, so the size change needs no relayout.
		setFont(c, *a.body, func(_ *swt.Control, set func()) { set() })
	}
	var pl *placement
	for _, o := range opts {
		switch o := o.(type) {
		case style:
			if o.id&allow == 0 {
				reportOn(c, fmt.Errorf("gowt: %v does not apply to %s", o.id, nameOf(c)))
			}
		case apply:
			o(c)
		case place:
			if pl == nil {
				pl = &placement{}
			}
			o(pl)
		case menuOpt:
			reportOn(c, fmt.Errorf("gowt: Shortcut applies to Menu.Item, not to %s", nameOf(c)))
		case boxOpt:
			if !isHost {
				reportOn(c, fmt.Errorf("gowt: Gap and Padding apply to containers, not to %s", nameOf(c)))
			}
		case *node:
			if !isHost {
				reportOn(c, fmt.Errorf("gowt: %s takes no children, %s is dropped", nameOf(c), o.name))
			}
		case failedOpt:
			reportOn(c, fmt.Errorf("gowt: Failed applies to Boundary, not to %s", nameOf(c)))
		}
	}
	if pl != nil {
		// A mismatch leaves c at its natural size: layout data of the wrong kind fails on every layout pass.
		d, err := pl.data(nameOf(c), layoutName(c.GetParent()))
		if err != nil {
			reportOn(c, err)
		} else {
			c.SetLayoutData(d)
		}
	}
	if isHost {
		addNodes(h, opts)
	}
	// Hooks come after the children, so that Selected finds the tabs and Init the panes of a Split.
	for _, o := range opts {
		if f, ok := o.(hook); ok {
			if err := f(w); err != nil {
				reportOn(c, err)
			}
		}
	}
	for _, o := range opts {
		if f, ok := o.(lateHook); ok {
			if err := f(w); err != nil {
				reportOn(c, err)
			}
		}
	}
	fluentBorder(c)
}

// addNodes creates the children among opts in h, in order; other options are skipped.
func addNodes(h host, opts []Option) {
	only := ""
	if o, ok := h.(interface{ only() string }); ok {
		only = o.only()
	}
	for _, o := range opts {
		n, ok := o.(*node)
		if !ok {
			continue
		}
		if only != "" && n.name != only {
			reportOn(h.control(), fmt.Errorf("gowt: %s takes %s children, not %s", nameOf(h.control()), only, n.name))
			continue
		}
		if w := n.build(h); w != nil {
			h.added(w)
		}
	}
}

// add is the Add of every container: children only, anything else is reported.
func add(h host, children []Option) {
	for _, o := range children {
		if _, ok := o.(*node); !ok {
			reportOn(h.control(), fmt.Errorf("gowt: Add on %s takes widgets; give options where the widget is created", nameOf(h.control())))
		}
	}
	addIn(h, children)
}

// leaf returns the node of a widget without children: make creates it in parent.
func leaf(name string, allow styleID, opts []Option, make func(parent *swt.Composite) Widget) Option {
	return &node{name, func(h host) Widget {
		w := make(h.into())
		finish(w, allow, opts)
		return w
	}}
}

// nameOf names c as the facade does, for messages.
func nameOf(c *swt.Control) string {
	switch n := c.GetName(); n {
	case "Composite":
		return "Panel"
	case "Shell":
		return "Window"
	case "TabFolder":
		return "Tabs"
	case "CTabFolder":
		return "CTabs"
	case "SashForm":
		return "Split"
	case "ScrolledComposite":
		return "Scroll"
	case "ProgressBar":
		return "Progress"
	default:
		return n
	}
}

// itemOpts checks the options of an item (Tab, ToolBar.Item, Menu.Item, Column): style options in
// allow, and Shortcut where shortcut is set, apply; anything else is reported against owner.
func itemOpts(owner displayed, item string, allow styleID, shortcut bool, opts []Option) {
	for _, o := range opts {
		switch o := o.(type) {
		case style:
			if o.id&allow != 0 {
				continue
			}
			reportOn(owner, fmt.Errorf("gowt: %v does not apply to %s", o.id, item))
		case menuOpt:
			if !shortcut {
				reportOn(owner, fmt.Errorf("gowt: Shortcut applies to Menu.Item, not to %s", item))
			}
		default:
			reportOn(owner, fmt.Errorf("gowt: %s takes only its style options", item))
		}
	}
}
