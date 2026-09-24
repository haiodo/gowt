package jrt

import (
	"reflect"
	"testing"
)

// DemoBase/DemoWidget stand in for a translated class hierarchy (see ClassEmitter's reflect
// registry) without needing a real swt.Display: DemoWidget embeds DemoBase at field 0, exactly
// like every translated widget embeds its superclass.
type DemoBase struct{ tip string }

func (b *DemoBase) GetTip() string  { return b.tip }
func (b *DemoBase) SetTip(v string) { b.tip = v }

type DemoWidget struct {
	DemoBase
	text string
}

func (w *DemoWidget) GetText() string         { return w.text }
func (w *DemoWidget) SetText(v string)        { w.text = v }
func (w *DemoWidget) SetValueInt(v int32)     { w.text = "int:" + string(rune('0'+v)) }
func (w *DemoWidget) SetValueString(v string) { w.text = "string:" + v }

func init() {
	RegisterMethod(reflect.TypeFor[*DemoBase](), "getTip", nil, reflect.TypeFor[string](),
		func(target any, args []any) any { return Narrow[*DemoBase](target).GetTip() })
	RegisterMethod(reflect.TypeFor[*DemoBase](), "setTip", []reflect.Type{reflect.TypeFor[string]()}, nil,
		func(target any, args []any) any { Narrow[*DemoBase](target).SetTip(ArgAs[string](args[0])); return nil })
	RegisterMethod(reflect.TypeFor[*DemoWidget](), "getText", nil, reflect.TypeFor[string](),
		func(target any, args []any) any { return Narrow[*DemoWidget](target).GetText() })
	RegisterMethod(reflect.TypeFor[*DemoWidget](), "setText", []reflect.Type{reflect.TypeFor[string]()}, nil,
		func(target any, args []any) any {
			Narrow[*DemoWidget](target).SetText(ArgAs[string](args[0]))
			return nil
		})
	// Two overloads of the same Java name "setValue" - exercises ClassGetMethod's paramTypes
	// disambiguation (same shape as Text.setSelection(int)/setSelection(Point) in the real port).
	RegisterMethod(reflect.TypeFor[*DemoWidget](), "setValue", []reflect.Type{reflect.TypeFor[int32]()}, nil,
		func(target any, args []any) any {
			Narrow[*DemoWidget](target).SetValueInt(ArgAs[int32](args[0]))
			return nil
		})
	RegisterMethod(reflect.TypeFor[*DemoWidget](), "setValue", []reflect.Type{reflect.TypeFor[string]()}, nil,
		func(target any, args []any) any {
			Narrow[*DemoWidget](target).SetValueString(ArgAs[string](args[0]))
			return nil
		})
}

func TestClassGetMethodInvoke(t *testing.T) {
	w := &DemoWidget{}
	rt := reflect.TypeOf(w)

	setText := ClassGetMethod(rt, "setText", []reflect.Type{reflect.TypeFor[string]()})
	setText.Invoke(w, "hello")
	getText := ClassGetMethod(rt, "getText", nil)
	if got := getText.Invoke(w); got != "hello" {
		t.Errorf("getText = %v, want hello", got)
	}
	if got := getText.GetReturnType(); got != reflect.TypeFor[string]() {
		t.Errorf("GetReturnType = %v, want string", got)
	}

	// Inherited (promoted) method, only registered on the embedded DemoBase.
	setTip := ClassGetMethod(rt, "setTip", []reflect.Type{reflect.TypeFor[string]()})
	setTip.Invoke(w, "tooltip")
	if w.tip != "tooltip" {
		t.Errorf("tip = %q, want tooltip", w.tip)
	}

	// Overload disambiguation by paramTypes.
	ClassGetMethod(rt, "setValue", []reflect.Type{reflect.TypeFor[int32]()}).Invoke(w, int32(5))
	if w.text != "int:5" {
		t.Errorf("text after setValue(int) = %q", w.text)
	}
	ClassGetMethod(rt, "setValue", []reflect.Type{reflect.TypeFor[string]()}).Invoke(w, "direct")
	if w.text != "string:direct" {
		t.Errorf("text after setValue(string) = %q", w.text)
	}
}

func TestClassGetMethodNotFound(t *testing.T) {
	defer func() {
		if _, ok := recover().(*NoSuchMethodException); !ok {
			t.Fatal("expected NoSuchMethodException")
		}
	}()
	ClassGetMethod(reflect.TypeOf(&DemoWidget{}), "doesNotExist", nil)
}

func TestClassName(t *testing.T) {
	cases := []struct {
		t    reflect.Type
		want string
	}{
		{reflect.TypeFor[int32](), "int"},
		{reflect.TypeFor[bool](), "boolean"},
		{reflect.TypeFor[uint16](), "char"},
		{reflect.TypeFor[string](), "java.lang.String"},
		{reflect.TypeFor[[]int32](), "[I"},
		{reflect.TypeFor[[]uint16](), "[C"},
		{reflect.TypeFor[[]string](), "[Ljava.lang.String;"},
	}
	for _, c := range cases {
		if got := ClassName(c.t); got != c.want {
			t.Errorf("ClassName(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}
