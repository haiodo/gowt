// Hand-written: ACC role to NSAccessibility role name, checked without a Display or a window.
package swt

import "testing"

func TestAccessibleRoleToOs(t *testing.T) {
	acc := &Accessible{}
	cases := map[int32]string{
		ACCROLE_PUSHBUTTON:  "AXButton",
		ACCROLE_CHECKBUTTON: "AXCheckBox",
		ACCROLE_RADIOBUTTON: "AXRadioButton",
		ACCROLE_LABEL:       "AXStaticText",
		ACCROLE_COMBOBOX:    "AXComboBox",
		ACCROLE_SCROLLBAR:   "AXScrollBar",
		ACCROLE_MENUITEM:    "AXMenuItem",
	}
	for role, want := range cases {
		if got := acc.RoleToOs(role); got != want {
			t.Errorf("RoleToOs(%#x) = %q, want %q", role, got, want)
		}
	}
}
