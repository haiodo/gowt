package swt

// Compatibility.pow2
func CompatibilityPow2(n int32) int32 {
	if n >= 1 && n <= 30 {
		return 2 << (n - 1)
	}
	if n != 0 {
		Error(ERROR_INVALID_RANGE)
	}
	return 1
}
