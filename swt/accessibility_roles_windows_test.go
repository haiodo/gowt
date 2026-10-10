// Hand-written: ACC role to MSAA role and back, needs no Display or window.
package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/win32"
)

func TestAccessibleRoleToOs(t *testing.T) {
	acc := &Accessible{}
	cases := map[int32]int32{
		ACCROLE_PUSHBUTTON:  win32.COMROLE_SYSTEM_PUSHBUTTON,
		ACCROLE_CHECKBUTTON: win32.COMROLE_SYSTEM_CHECKBUTTON,
		ACCROLE_RADIOBUTTON: win32.COMROLE_SYSTEM_RADIOBUTTON,
		ACCROLE_LABEL:       win32.COMROLE_SYSTEM_STATICTEXT,
		ACCROLE_COMBOBOX:    win32.COMROLE_SYSTEM_COMBOBOX,
		ACCROLE_TEXT:        win32.COMROLE_SYSTEM_TEXT,
	}
	for role, want := range cases {
		if got := acc.RoleToOs(role); got != want {
			t.Errorf("RoleToOs(%#x) = %#x, want %#x", role, got, want)
		}
		if got := acc.OsToRole(want); got != role {
			t.Errorf("OsToRole(%#x) = %#x, want %#x", want, got, role)
		}
	}
}
