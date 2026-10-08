//go:build !darwin

package gowt

func classicLook() {}

func (w *Window) setFullSizeContent(bool) {}

func (p *panel) setGlass(bool) {}

func glassButton() Option { return Option{} }
