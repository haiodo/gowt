package gowt

import "github.com/haiodo/gowt/internal/cocoa"

func systemDark() bool { return cocoa.EffectiveAppearanceDark() }
