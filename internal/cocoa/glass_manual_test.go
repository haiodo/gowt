//go:build darwin

package cocoa

import "testing"

func TestRequireCompatibleLook(t *testing.T) {
	t.Logf("glass available: %v", GlassAvailable())
	onMain(func() {
		RequireCompatibleLook()
		if !CompatibleLookKey() {
			t.Error("UIDesignRequiresCompatibility not readable from the main bundle after RequireCompatibleLook")
		}
	})
}
