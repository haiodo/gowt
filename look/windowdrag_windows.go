package look

import (
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

func beginMove(s *swt.Shell, _ *swt.Control, _ *swt.MouseEvent) {
	win32.OSReleaseCapture()
	win32.OSSendMessage(s.Handle, win32.OSWM_NCLBUTTONDOWN, int64(win32.OSHTCAPTION), nil)
}

func doubleClick(s *swt.Shell) { toggleMaximized(s) }
