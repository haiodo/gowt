//go:build !(amd64 || arm64)

package jrt

func goid() uint64 { return goidSlow() }
