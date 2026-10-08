package gowt

// SetRoundedCorners forces rounded (true) or square (false) window corners on Windows 11; other
// systems and older Windows builds ignore it. Windows 11 rounds top-level windows by default.
func (w *Window) SetRoundedCorners(on bool) { w.setRoundedCorners(on) }
