package swt

import (
	"reflect"

	"github.com/haiodo/gowt/internal/jrt"
)

// GetTypedListeners is Widget.getTypedListeners: the listeners added with addTypedListener
// (and the DND ones) whose class is listenerType. Generic in Java, so written by hand.
func (this *Widget) GetTypedListeners(eventType int32, listenerType reflect.Type) *jrt.List {
	var out []any
	for _, l := range this.GetListeners(eventType) {
		if t, ok := typedListenerImplAsTypedListener(l); ok && t.eventListener != nil && reflect.TypeOf(t.eventListener).AssignableTo(listenerType) {
			out = append(out, t.eventListener)
		}
	}
	return jrt.ListOf(out...)
}
