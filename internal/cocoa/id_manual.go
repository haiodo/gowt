// Hand-written: org.eclipse.swt.internal.cocoa.id#objc_getClass()/toString() both call
// getClass().getName(), Java reflection with no Go equivalent (see tooling/j2go/manual.txt).
// this.impl always holds the pointer to the most-derived allocated type (set once, in the
// leaf New<Class>() factory - see tooling/j2go/README.md "impl cascade"), so its own runtime
// type name is exactly the ObjC class name SWT's alloc() pattern needs.
package cocoa

import (
	"reflect"
	"strconv"
)

// Exported aliases: Go visibility is the identifier's own case, and these two Java class names
// start lowercase - swt-package callers need "cocoa.Id"/"cocoa.ObjcSuper", not the bare originals.
type Id = id
type ObjcSuper = objc_super

func (this *id) className() string {
	return reflect.TypeOf(this.impl).Elem().Name()
}

func (this *id) ObjcGetClass() int64 {
	return OSObjc_getClass(this.className())
}

func (this *id) String() string {
	return this.className() + "{" + strconv.FormatInt(this.Id, 10) + "}"
}

// AsId upcasts any id-hierarchy pointer (NSObject, NSString, ...) to *id: the embedded field
// name is "id" (unexported, selectable only inside this package) but a promoted method isn't.
func (this *id) AsId() *id {
	return this
}

// Impl exposes the impl-cascade dynamic-type value to cross-package callers (swt's Control.java
// needs it for instanceof/downcast) - the field itself is unexported.
func (this *id) Impl() idImpl {
	return this.impl
}
