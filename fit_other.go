//go:build !linux

package gowt

// packColumn is the toolkit's own autosize: header and content.
func packColumn(c column) { c.Pack() }
