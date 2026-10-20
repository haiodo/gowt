package gowt

// postWheelDown is not written for Windows: the test VM was unreachable when OnScroll was added.
func postWheelDown(*ScrollView) bool { return false }
