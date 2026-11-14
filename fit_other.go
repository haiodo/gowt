//go:build !linux && !darwin

package gowt

// packColumn is the toolkit's own autosize: header and content.
func packColumn(c column) { c.Pack() }
