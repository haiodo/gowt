package look

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func beginMove(s *swt.Shell, _ *swt.Control, _ *swt.MouseEvent) { cocoa.BeginWindowDrag(s.View) }

func doubleClick(s *swt.Shell) { cocoa.TitleBarDoubleClick(s.View) }
