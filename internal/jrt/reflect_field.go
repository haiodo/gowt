package jrt

import (
	"reflect"
	"unsafe"
)

// Field is java.lang.reflect.Field of a translated class: a struct field found by its Java name.
// Struct-field reflection only, so it does not defeat dead-code elimination (see reflect.go).
type Field struct {
	owner reflect.Type // *Class
	index int
}

// NoSuchFieldException is java.lang.NoSuchFieldException.
type NoSuchFieldException struct{ Message string }

func (e *NoSuchFieldException) Error() string { return "NoSuchFieldException: " + e.Message }

// ClassGetDeclaredField is Class#getDeclaredField for a pointer-to-struct type; the Go field keeps the Java name.
func ClassGetDeclaredField(t reflect.Type, name string) *Field {
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		if f, ok := t.Elem().FieldByName(name); ok && len(f.Index) == 1 {
			return &Field{owner: t, index: f.Index[0]}
		}
	}
	panic(&NoSuchFieldException{Message: name})
}

// Get is Field#get; unexported fields are read through their address (setAccessible(true) is implied).
func (f *Field) Get(obj any) any {
	v := upcast(reflect.ValueOf(obj), f.owner).Elem().Field(f.index)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface()
}
