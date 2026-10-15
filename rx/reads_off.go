//go:build !rxdebug

package rx

// checkReads also checks the thread in Get, the hottest call; build with -tags rxdebug to enable.
const checkReads = false
