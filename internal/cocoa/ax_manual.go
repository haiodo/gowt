//go:build darwin

// Hand-written: read AppKit accessibility properties and perform accessibility actions on the
// same NSControl cell or NSView that VoiceOver queries.
package cocoa

func axHandle(view *NSView) uintptr {
	v := uintptr(view.Id)
	if respondsTo(v, "cell") {
		if cell := msg(v, "cell"); cell != 0 {
			return cell
		}
	}
	return v
}

// AXStringAttribute returns the string value of an NSAccessibility attribute ("" when nil).
func AXStringAttribute(view *NSView, attr string) string {
	return goString(msg(axHandle(view), axProperty(attr)))
}

// AXBoolAttribute returns the bool value of an NSAccessibility attribute.
func AXBoolAttribute(view *NSView, attr string) bool {
	if attr == "AXEnabled" || attr == "AXFocused" {
		return msg(msg(axHandle(view), "accessibilityAttributeValue:", nsString(attr)), "boolValue") != 0
	}
	return msg(msg(axHandle(view), axProperty(attr)), "boolValue") != 0
}

// AXIntAttribute returns the integer value of an NSAccessibility attribute.
func AXIntAttribute(view *NSView, attr string) int {
	return int(msg(msg(axHandle(view), axProperty(attr)), "integerValue"))
}

func axProperty(attr string) string {
	switch attr {
	case "AXRole":
		return "accessibilityRole"
	case "AXTitle":
		return "accessibilityTitle"
	case "AXValue":
		return "accessibilityValue"
	}
	return "accessibility" + attr[2:]
}

// AXPerformAction performs an NSAccessibility action such as NSAccessibilityPressAction.
func AXPerformAction(view *NSView, action string) {
	if action == "AXPress" {
		msg(axHandle(view), "accessibilityPerformPress")
		return
	}
	msg(axHandle(view), "accessibilityPerformAction:", nsString(action))
}

// AXActionNames lists the NSAccessibility actions the accessible cell or view answers.
func AXActionNames(view *NSView) []string {
	a := msg(axHandle(view), "accessibilityActionNames")
	n := int(msg(a, "count"))
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, goString(msg(a, "objectAtIndex:", uintptr(i))))
	}
	return out
}
