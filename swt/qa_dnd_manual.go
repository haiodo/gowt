package swt

import (
	"errors"
	"slices"

	"github.com/haiodo/gowt/internal/jrt"
)

// QADragResult: Started is false when a listener vetoed the drag, Entered when the target took no
// offered type; Operation is DNDDROP_NONE when nobody accepted.
type QADragResult struct {
	Started, Entered bool
	Operation        int32
}

// QADrag runs one drag between two controls of this app without an OS drag session, in the order and
// with the types and operations the native code of this OS uses. DragOver runs once, not per mouse move.
// Points are in the controls' own coordinates.
func QADrag(src *Control, sx, sy int32, dst *Control, dx, dy int32) (QADragResult, error) {
	var r QADragResult
	ds, _ := src.impl.getDataKey_(DNDDRAG_SOURCE_KEY).(*DragSource)
	if ds == nil || ds.IsDisposed() {
		return r, errors.New("swt: the control has no drag source (gowt.DragFrom)")
	}
	dt, _ := dst.impl.getDataKey_(DNDDROP_TARGET_KEY).(*DropTarget)
	if dt == nil || dt.IsDisposed() {
		return r, errors.New("swt: the control has no drop target (gowt.DropOn)")
	}
	now := int32(jrt.CurrentTimeMillis())

	// The source's own DragDetect listener would start the OS session; the others see the event.
	det := NewEvent()
	det.Type, det.Display, det.Widget = DragDetect, src.GetDisplay(), upcastControlToWidget(src)
	det.X, det.Y, det.Button, det.Time = sx, sy, 1, now
	for _, l := range src.GetListeners(DragDetect) {
		if l != ds.controlListener {
			l.HandleEvent(det)
		}
	}
	if !det.Doit {
		return r, nil
	}

	start := newDNDEvent()
	start.Widget, start.X, start.Y, start.Time, start.Doit = upcastDragSourceToWidget(ds), sx, sy, now, true
	ds.NotifyListeners(DNDDragStart, upcastDNDEventToEvent(start))
	if !start.Doit || !ds.CanBeginDrag() {
		return r, nil
	}
	r.Started = true

	r.Entered, r.Operation = qaDropOn(ds, dt, dst, dx, dy)

	end := newDNDEvent()
	end.Widget, end.Time = upcastDragSourceToWidget(ds), int32(jrt.CurrentTimeMillis())
	end.Doit, end.Detail = r.Operation != DNDDROP_NONE, r.Operation
	ds.NotifyListeners(DNDDragEnd, upcastDNDEventToEvent(end))
	return r, nil
}

// qaDropOn is the target half of QADrag; the order and the rules for the type and the operation are
// those of DropTarget.DragEnter, DragOver and Drop in each OS file.
func qaDropOn(ds *DragSource, dt *DropTarget, dst *Control, dx, dy int32) (entered bool, op int32) {
	var offers []*TransferData
	for _, a := range ds.transferAgents {
		if a == nil {
			continue
		}
		for _, td := range a.impl.getSupportedTypes_() {
			if qaTargetTakes(dt, td) {
				offers = append(offers, td)
			}
		}
	}
	style := dt.impl.getStyle_()
	allowed := ds.impl.getStyle_() & style
	if len(offers) == 0 || allowed == DNDDROP_NONE {
		return false, DNDDROP_NONE
	}
	key := int32(DNDDROP_DEFAULT)
	if style&DNDDROP_DEFAULT == 0 {
		key = DNDDROP_NONE
		if allowed&DNDDROP_MOVE != 0 {
			key = DNDDROP_MOVE
		}
	}
	p := qaDisplayPoint(dst, dx, dy)
	newEvent := func() *DNDEvent {
		e := newDNDEvent()
		e.Widget, e.X, e.Y, e.Time = upcastDropTargetToWidget(dt), p.X, p.Y, int32(jrt.CurrentTimeMillis())
		e.Feedback, e.DataTypes, e.DataType = DNDFEEDBACK_SELECT, slices.Clone(offers), offers[0]
		e.Operations, e.Detail = allowed, key
		if dt.dropEffect != nil {
			e.Item = dt.dropEffect.GetItem(p.X, p.Y)
		}
		return e
	}
	// pick applies what the listeners left in e: DEFAULT becomes MOVE, the type must be one offered.
	pick := func(e *DNDEvent, exact bool) (*TransferData, int32) {
		if e.Detail == DNDDROP_DEFAULT {
			e.Detail = DNDDROP_NONE
			if allowed&DNDDROP_MOVE != 0 {
				e.Detail = DNDDROP_MOVE
			}
		}
		var sel *TransferData
		if e.DataType != nil {
			for _, o := range offers {
				if o.Type == e.DataType.Type {
					sel = o
					break
				}
			}
		}
		if sel != nil && (exact && allowed&e.Detail == e.Detail || !exact && allowed&e.Detail != 0) {
			return sel, e.Detail
		}
		return sel, DNDDROP_NONE
	}
	leave := func() {
		e := newDNDEvent()
		e.Widget, e.Time, e.Detail = upcastDropTargetToWidget(dt), int32(jrt.CurrentTimeMillis()), DNDDROP_NONE
		if dt.dropEffect != nil {
			e.Item = dt.dropEffect.GetItem(p.X, p.Y)
		}
		dt.NotifyListeners(DNDDragLeave, upcastDNDEventToEvent(e))
	}

	e := newEvent()
	dt.NotifyListeners(DNDDragEnter, upcastDNDEventToEvent(e))
	sel, op := pick(e, false)

	e = newEvent()
	e.DataType, e.Detail = sel, op
	dt.NotifyListeners(DNDDragOver, upcastDNDEventToEvent(e))
	sel, op = pick(e, true)

	leave()
	if op == DNDDROP_NONE {
		return true, DNDDROP_NONE
	}
	e = newEvent()
	e.DataType, e.Detail = sel, op
	dt.NotifyListeners(DNDDropAccept, upcastDNDEventToEvent(e))
	sel, op = pick(e, true)
	if sel == nil || op == DNDDROP_NONE {
		return true, DNDDROP_NONE
	}
	data := qaPull(ds, dt, sel)
	if jrt.IsNil(data) {
		op = DNDDROP_NONE
	}
	e.DataType, e.Detail, e.Data = sel, op, data
	dt.NotifyListeners(DNDDrop, upcastDNDEventToEvent(e))
	if allowed&e.Detail == e.Detail {
		return true, e.Detail
	}
	return true, DNDDROP_NONE
}

func qaTargetTakes(dt *DropTarget, td *TransferData) bool {
	for _, a := range dt.transferAgents {
		if a != nil && a.impl.isSupportedType_(td) {
			return true
		}
	}
	return false
}

// qaPull moves the data of type td from the source to the target: the source answers DragSetData
// and puts it in native form, the target reads it back. qaWire is the part that differs per OS.
func qaPull(ds *DragSource, dt *DropTarget, td *TransferData) any {
	done, ok := qaWire(ds, td)
	if !ok {
		return nil
	}
	defer done()
	for _, a := range dt.transferAgents {
		if a != nil && a.impl.isSupportedType_(td) {
			return a.impl.nativeToJava_(td)
		}
	}
	return nil
}

// qaSetData sends DragSetData and converts what the listeners gave to the native form in td.
func qaSetData(ds *DragSource, td *TransferData) bool {
	e := newDNDEvent()
	e.Widget, e.Time, e.DataType = upcastDragSourceToWidget(ds), int32(jrt.CurrentTimeMillis()), td
	ds.NotifyListeners(DNDDragSetData, upcastDNDEventToEvent(e))
	if !e.Doit {
		return false
	}
	for _, a := range ds.transferAgents {
		if a != nil && a.impl.isSupportedType_(td) {
			a.impl.javaToNative_(e.Data, td)
			return true
		}
	}
	return false
}

// qaDisplayPoint is the display point that ToControl maps back to dx,dy. With a zoom other than 100 the
// round trip ToDisplay then ToControl rounds twice and can land a point off; a neighbour is exact.
func qaDisplayPoint(c *Control, dx, dy int32) *Point {
	p := c.ToDisplay(dx, dy)
	for _, d := range [...][2]int32{{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, 1}, {-1, 1}, {1, -1}} {
		q := NewPoint(p.X+d[0], p.Y+d[1])
		if b := c.ToControl(q.X, q.Y); b.X == dx && b.Y == dy {
			return q
		}
	}
	return p
}
