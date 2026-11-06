//go:build linux

package gtk

import "unsafe"

var (
	gtkHasClass          = lz[func(uintptr, unsafe.Pointer) int32]{sym: "gtk_style_context_has_class"}
	gtkGetAccessible     = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_accessible"}
	atkSetName           = lz[func(uintptr, unsafe.Pointer)]{sym: "atk_object_set_name"}
	atkGetName           = lz[func(uintptr) unsafe.Pointer]{sym: "atk_object_get_name"}
	atkGetRole           = lz[func(uintptr) int32]{sym: "atk_object_get_role"}
	atkRoleName          = lz[func(int32) unsafe.Pointer]{sym: "atk_role_get_name"}
	atkTextGetText       = lz[func(uintptr, int32, int32) uintptr]{sym: "atk_text_get_text"}
	atkValueGetCurrent   = lz[func(uintptr, uintptr)]{sym: "atk_value_get_current_value"}
	gValueGetDouble      = lz[func(uintptr) float64]{sym: "g_value_get_double"}
	atkObjectRefStateSet = lz[func(uintptr) uintptr]{sym: "atk_object_ref_state_set"}
	atkStateTypeForName  = lz[func(unsafe.Pointer) int32]{sym: "atk_state_type_for_name"}
	atkStateSetContains  = lz[func(uintptr, int32) int32]{sym: "atk_state_set_contains_state"}
	atkActionGetNActions = lz[func(uintptr) int32]{sym: "atk_action_get_n_actions"}
	atkActionGetName     = lz[func(uintptr, int32) uintptr]{sym: "atk_action_get_name"}
	atkActionDoAction    = lz[func(uintptr, int32) int32]{sym: "atk_action_do_action"}
	gtkButtonImage       = lz[func(uintptr) uintptr]{sym: "gtk_button_get_image"}
)

// SetButtonIconName shows the named theme icon in the image of a GTK 3 button that has one, such as
// SWT's arrow button. GTK 4 keeps the image.
func SetButtonIconName(button int64, name string) {
	if GTKGTK4 {
		return
	}
	if img := gtkButtonImage.get()(uintptr(button)); img != 0 {
		GTK3Gtk_image_set_from_icon_name(int64(img), cs(name), GTKGTK_ICON_SIZE_MENU)
	}
}

// AddStyleClass adds a CSS class such as "flat" or "circular" to the widget's style context.
func AddStyleClass(widget int64, class string) {
	GTKGtk_style_context_add_class(GTKGtk_widget_get_style_context(widget), cs(class))
}

// HasStyleClass reports whether the widget's style context has the CSS class.
func HasStyleClass(widget int64, class string) bool {
	return gtkHasClass.get()(uintptr(GTKGtk_widget_get_style_context(widget)), sp(cs(class))) != 0
}

// SetAccessibleName sets the name a screen reader announces for the widget (ATK).
func SetAccessibleName(widget int64, name string) {
	if a := gtkGetAccessible.get()(uintptr(widget)); a != 0 {
		atkSetName.get()(a, sp(cs(name)))
	}
}

// AccessibleName returns the ATK name a screen reader announces for the widget ("" when none).
func AccessibleName(widget int64) string {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return ""
	}
	return ConverterCCharPtrToJavaString(int64(uintptr(atkGetName.get()(a))), false)
}

// AccessibleRoleName returns the ATK role name of the widget, such as "push button" or "check box".
func AccessibleRoleName(widget int64) string {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return ""
	}
	return ConverterCCharPtrToJavaString(int64(uintptr(atkRoleName.get()(atkGetRole.get()(a)))), false)
}

// AccessibleText returns the text exposed through the widget's AtkText interface.
func AccessibleText(widget int64) string {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return ""
	}
	return ConverterCCharPtrToJavaString(int64(uintptr(atkTextGetText.get()(a, 0, -1))), true)
}

// AccessibleNumber returns the current numeric value exposed through the widget's AtkValue interface.
func AccessibleNumber(widget int64) (float64, bool) {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return 0, false
	}
	var value GValue
	atkValueGetCurrent.get()(a, uintptr(unsafe.Pointer(&value)))
	if value.G_type == 0 {
		return 0, false
	}
	defer gValueUnset.get()(unsafe.Pointer(&value))
	if value.G_type != 15<<2 { // G_TYPE_DOUBLE
		return 0, false
	}
	return gValueGetDouble.get()(uintptr(unsafe.Pointer(&value))), true
}

// AccessibleState reports whether the ATK state set contains the named state, such as "focused".
func AccessibleState(widget int64, name string) bool {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return false
	}
	states := atkObjectRefStateSet.get()(a)
	if states == 0 {
		return false
	}
	defer OSG_object_unref(int64(states))
	state := atkStateTypeForName.get()(sp(cs(name)))
	return atkStateSetContains.get()(states, state) != 0
}

// AccessibleActionNames lists the actions exposed through the widget's AtkAction interface.
func AccessibleActionNames(widget int64) []string {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return nil
	}
	n := int(atkActionGetNActions.get()(a))
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, ConverterCCharPtrToJavaString(int64(uintptr(atkActionGetName.get()(a, int32(i)))), false))
	}
	return out
}

// AccessiblePerformAction performs the named ATK action.
func AccessiblePerformAction(widget int64, name string) bool {
	a := gtkGetAccessible.get()(uintptr(widget))
	if a == 0 {
		return false
	}
	for i, action := range AccessibleActionNames(widget) {
		if action == name {
			return atkActionDoAction.get()(a, int32(i)) != 0
		}
	}
	return false
}
