package swt

import "testing"

func TestCompatibilityGetMessage(t *testing.T) {
	for _, c := range []struct {
		got, want string
	}{
		{GetMessage("SWT_Yes"), "Yes"},
		{GetMessage("SWT_No"), "No"},
		{GetMessage("SWT_Missing_Key"), "SWT_Missing_Key"},
		{GetMessageKeyArgs("SWT_Page_Mnemonic", []any{uint16('3')}), "Alt+3"},
		{GetMessageKeyArgs("SWT_Download_Status", []any{12, 40}), "Download: 12 KB of 40 KB"},
		{GetMessageKeyArgs("SWT_IssuedTo", []any{"me"}), "Issued to:\tme"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
