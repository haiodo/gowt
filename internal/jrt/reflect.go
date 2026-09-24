// Package jrt: java.lang.reflect subset ControlExample's Set/Get dialog needs (Tab.java's
// getValue/setValue/getReturnType/parameterInfo) - see tooling/j2go/README.md "Round 10
// reflection". java.lang.Class itself stays reflect.Type (Manual.java); this file adds
// getMethod/invoke/getReturnType/isArray/getComponentType/getName on top of it.
//
// The obvious implementation - resolve the Java method name to a Go name, then call
// reflect.Type.MethodByName/Method(i) - is deliberately not used: the Go linker sees any static
// call to those two APIs and, unable to tell which name will be looked up at runtime, keeps every
// exported method of every reflect-reachable type in the binary (defeats dead-code elimination
// program-wide). Instead j2go generates package swt/swtreflect, whose init() registers each public
// method of the translated widgets-package classes as a closure naming its concrete types
// statically. Only programs importing swtreflect keep those methods (README "Round 11").
// Inherited (promoted) methods are found by walking the embedding chain at lookup time
// (parentOf) and dispatched through Narrow, both using only struct-field reflection (Field/
// Addr/Interface), never Method/MethodByName.
package jrt

import (
	"fmt"
	"reflect"
)

// MethodEntry is one registered method: paramTypes/returnType are its own Go types (for
// getMethod's overload matching and Method.getReturnType), call performs the actual dispatch.
type MethodEntry struct {
	paramTypes []reflect.Type
	returnType reflect.Type
	call       func(target any, args []any) any
}

var classMethods = map[reflect.Type]map[string][]*MethodEntry{}

// RegisterMethod is called from generated code - swt/swtreflect's init() (ReflectEmitter, README
// "Round 10 reflection").
func RegisterMethod(t reflect.Type, javaName string, paramTypes []reflect.Type, returnType reflect.Type, call func(target any, args []any) any) {
	m := classMethods[t]
	if m == nil {
		m = map[string][]*MethodEntry{}
		classMethods[t] = m
	}
	m[javaName] = append(m[javaName], &MethodEntry{paramTypes: paramTypes, returnType: returnType, call: call})
}

// Method is java.lang.reflect.Method, as returned by ClassGetMethod.
type Method struct {
	javaName string
	e        *MethodEntry
}

// GetReturnType is Method.getReturnType(); nil (a nil reflect.Type) for a void method.
func (mth *Method) GetReturnType() reflect.Type { return mth.e.returnType }

// Invoke is Method.invoke(target, args...). A failing call panics *InvocationTargetException with
// the real cause, matching the "exceptions are panics" contract the rest of jrt already uses
// (ParseInt, ResourceBundle, ...).
func (mth *Method) Invoke(target any, args ...any) (result any) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if e, ok := r.(error); ok {
			panic(&InvocationTargetException{Cause: e})
		}
		panic(&InvocationTargetException{Cause: fmt.Errorf("%v", r)})
	}()
	return mth.e.call(target, args)
}

// ClassGetMethod is java.lang.Class#getMethod(String, Class...): looks the Java method name up in
// t's own registered methods, then its ancestors' (Control, Widget, ...) by walking the same
// field-0 embedding chain Go itself promotes methods through (every translated class embeds its
// superclass at field 0 - see README "impl cascade"). Several registered entries can share
// javaName (Java overloads); the first whose paramTypes are all assignable from the requested ones
// wins - correct as long as at most one overload matches, same ceiling Java's own overload
// resolution would have on an ambiguous erasure.
func ClassGetMethod(t reflect.Type, name string, paramTypes []reflect.Type) *Method {
	for tt := t; tt != nil; tt = parentOf(tt) {
		for _, e := range classMethods[tt][name] {
			if paramsMatch(e.paramTypes, paramTypes) {
				return &Method{javaName: name, e: e}
			}
		}
	}
	panic(&NoSuchMethodException{Message: name})
}

func paramsMatch(want, have []reflect.Type) bool {
	if len(want) != len(have) {
		return false
	}
	for i, w := range want {
		if have[i] == nil || !have[i].AssignableTo(w) {
			return false
		}
	}
	return true
}

// parentOf is the type embedded at field 0 of t.Elem() (t is always a pointer to a translated
// class's struct) - nil once there is no more embedding (Widget itself). Struct-field reflection
// only (Field, not Method) - safe, see the package doc.
func parentOf(t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct || t.Elem().NumField() == 0 {
		return nil
	}
	f := t.Elem().Field(0)
	if !f.Anonymous {
		return nil
	}
	return reflect.PointerTo(f.Type)
}

// Narrow gives a registered method's closure the *DeclaringClass pointer it needs to call itself,
// from whatever concrete leaf type ClassGetMethod was actually asked about (target may be *Button
// while the method is declared on *Control) - the address of the field-0 chain's matching link,
// exactly what Go's own method promotion uses. Struct-field reflection only, see the package doc.
func Narrow[T any](target any) T {
	want := reflect.TypeFor[T]()
	v := reflect.ValueOf(target)
	for v.Type() != want {
		if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct || v.Elem().NumField() == 0 {
			panic(&IllegalArgumentException{Message: "jrt: " + v.Type().String() + " does not embed " + want.String()})
		}
		v = v.Elem().Field(0).Addr()
	}
	return v.Interface().(T)
}

// ArgAs converts one Method.invoke argument (boxed any, nil for Java null) to the registered
// closure's own parameter type.
func ArgAs[T any](a any) T {
	if a == nil {
		var zero T
		return zero
	}
	return a.(T)
}

// NoSuchMethodException is java.lang.NoSuchMethodException.
type NoSuchMethodException struct{ Message string }

func (e *NoSuchMethodException) Error() string { return "NoSuchMethodException: " + e.Message }

// InvocationTargetException is java.lang.reflect.InvocationTargetException.
type InvocationTargetException struct{ Cause error }

func (e *InvocationTargetException) Error() string {
	return "java.lang.reflect.InvocationTargetException: " + e.Cause.Error()
}
func (e *InvocationTargetException) Unwrap() error { return e.Cause }

// graphicsTypes are this port's swt value-type names whose real Java package is
// org.eclipse.swt.graphics, for ClassName's pointer-to-struct case.
var graphicsTypes = map[string]bool{
	"Point": true, "Rectangle": true, "RGB": true, "RGBA": true,
	"Color": true, "Font": true, "FontData": true, "Image": true, "FontMetrics": true,
}

// ClassName is java.lang.Class#getName(): JVM primitive names and array-descriptor syntax
// exactly (what the Set/Get dialog branches on), plus a best-effort "org.eclipse.swt.widgets.
// <Name>" guess for a pointer-to-struct type outside graphicsTypes - wrong package for e.g. a
// layout type, harmless since every caller (Widget.getName(), every toString()) only keeps the
// tail after the last '.'. Upgrade path: a translator-emitted per-class Java-FQN registry.
func ClassName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int16:
		return "short"
	case reflect.Int32:
		return "int"
	case reflect.Int64:
		return "long"
	case reflect.Uint16:
		return "char"
	case reflect.String:
		return "java.lang.String"
	case reflect.Slice:
		return "[" + arrayDescriptor(t.Elem())
	case reflect.Ptr:
		name := t.Elem().Name()
		if graphicsTypes[name] {
			return "org.eclipse.swt.graphics." + name
		}
		return "org.eclipse.swt.widgets." + name
	default:
		return t.String()
	}
}

func arrayDescriptor(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "Z"
	case reflect.Int32:
		return "I"
	case reflect.Int64:
		return "J"
	case reflect.Uint16:
		return "C"
	case reflect.String:
		return "Ljava.lang.String;"
	case reflect.Ptr:
		return "L" + ClassName(t) + ";"
	default:
		return "L" + t.String() + ";"
	}
}
