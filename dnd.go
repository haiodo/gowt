package gowt

import "github.com/haiodo/gowt/swt"

// Op is a drag and drop operation. In Drag.Ops and Drop.Ops the zero value means OpCopy.
type Op int

// The operations a drag may offer and a drop may accept; OpNone reports a drag nobody accepted.
const (
	OpNone Op = Op(swt.DNDDROP_NONE)
	OpCopy Op = Op(swt.DNDDROP_COPY)
	OpMove Op = Op(swt.DNDDROP_MOVE)
	OpLink Op = Op(swt.DNDDROP_LINK)
)

func (o Op) bits() int32 {
	if o == 0 {
		return swt.DNDDROP_COPY
	}
	return int32(o)
}

// Drag describes what a widget offers when the user drags from it. Every non-nil getter adds
// its format; getters run when a target asks for the data. Custom formats need swt.DragSource
// through Unwrap.
type Drag struct {
	Ops     Op              // allowed operations, zero means OpCopy
	Text    func() string   // plain text
	Files   func() []string // file paths
	Image   func() *Image   // not disposed by the facade
	OnStart func() bool     // false cancels this drag; may be nil
	OnDone  func(op Op)     // after the drop; OpNone if nobody took it; may be nil
}

// Drop describes what a widget accepts. Every non-nil callback adds its format. Custom
// formats need swt.DropTarget through Unwrap.
type Drop struct {
	Ops   Op                          // accepted operations, zero means OpCopy
	Text  func(text string, op Op)    // plain text dropped
	Files func(paths []string, op Op) // files dropped
	Image func(img *Image, op Op)     // image dropped; the App owns img (Dispose may be called earlier)
	Over  func(x, y int) bool         // x, y in widget coordinates; false rejects the drop at that point; may be nil
}

// DragHandle is the drag source created by DragFrom.
type DragHandle struct{ s *swt.DragSource }

// Unwrap returns the underlying swt.DragSource for the full API.
func (h *DragHandle) Unwrap() *swt.DragSource { return h.s }

// DropHandle is the drop target created by DropOn.
type DropHandle struct{ t *swt.DropTarget }

// Unwrap returns the underlying swt.DropTarget for the full API.
func (h *DropHandle) Unwrap() *swt.DropTarget { return h.t }

// DragFrom makes w a drag source. A control has at most one; a second call panics with an
// SWT error that Run returns. The source is disposed with the widget.
func DragFrom(w Widget, d Drag) *DragHandle {
	var ts []*swt.Transfer
	if d.Text != nil {
		ts = append(ts, swt.TextTransferGetInstance().AsTransfer())
	}
	if d.Files != nil {
		ts = append(ts, swt.FileTransferGetInstance().AsTransfer())
	}
	if d.Image != nil {
		ts = append(ts, swt.ImageTransferGetInstance().AsTransfer())
	}
	s := swt.NewDragSource(w.control(), d.Ops.bits())
	s.SetTransfer(ts)
	s.AddDragListener(&dragListener{d: d})
	return &DragHandle{s}
}

type dragListener struct{ d Drag }

func (l *dragListener) DragStart(e *swt.DragSourceEvent) {
	if l.d.OnStart != nil && !l.d.OnStart() {
		e.Doit = false
	}
}

func (l *dragListener) DragSetData(e *swt.DragSourceEvent) {
	switch {
	case l.d.Text != nil && swt.TextTransferGetInstance().IsSupportedType(e.DataType):
		e.Data = l.d.Text()
	case l.d.Files != nil && swt.FileTransferGetInstance().IsSupportedType(e.DataType):
		e.Data = l.d.Files()
	case l.d.Image != nil && swt.ImageTransferGetInstance().IsSupportedType(e.DataType):
		e.Data = l.d.Image().i.GetImageData()
	}
}

func (l *dragListener) DragFinished(e *swt.DragSourceEvent) {
	if l.d.OnDone != nil {
		l.d.OnDone(Op(e.Detail))
	}
}

// DropOn makes w a drop target. A control has at most one; a second call panics with an SWT
// error that Run returns. The target is disposed with the widget.
func DropOn(w Widget, d Drop) *DropHandle {
	ctl := w.control()
	var ts []*swt.Transfer
	if d.Text != nil {
		ts = append(ts, swt.TextTransferGetInstance().AsTransfer())
	}
	if d.Files != nil {
		ts = append(ts, swt.FileTransferGetInstance().AsTransfer())
	}
	if d.Image != nil {
		ts = append(ts, swt.ImageTransferGetInstance().AsTransfer())
	}
	t := swt.NewDropTarget(ctl, d.Ops.bits())
	t.SetTransfer(ts)
	t.AddDropListener(&dropListener{DropTargetAdapter: swt.NewDropTargetAdapter(), d: d, ctl: ctl})
	return &DropHandle{t}
}

type dropListener struct {
	*swt.DropTargetAdapter
	d   Drop
	ctl *swt.Control
}

func (l *dropListener) DragOver(e *swt.DropTargetEvent) {
	if l.d.Over == nil {
		return
	}
	p := l.ctl.ToControl(e.X, e.Y)
	if !l.d.Over(int(p.X), int(p.Y)) {
		e.Detail = swt.DNDDROP_NONE
	}
}

func (l *dropListener) Drop(e *swt.DropTargetEvent) {
	op := Op(e.Detail)
	switch data := e.Data.(type) {
	case string:
		if l.d.Text != nil {
			l.d.Text(data, op)
		}
	case []string:
		if l.d.Files != nil {
			l.d.Files(data, op)
		}
	case *swt.ImageData:
		if l.d.Image != nil {
			if a, ok := apps.Load(l.ctl.GetDisplay()); ok {
				app := a.(*App)
				l.d.Image(app.track(swt.NewImageDeviceData(app.display, data)), op)
			}
		}
	}
}
