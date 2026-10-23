//go:build !gowtqa

package gowt

func qaStart(*App) {}

func qaQuiet() bool { return false }
