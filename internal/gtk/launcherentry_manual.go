//go:build linux

package gtk

import "unsafe"

var (
	gBusGetSync          = lz[func(int32, uintptr, unsafe.Pointer) uintptr]{sym: "g_bus_get_sync"}
	gDBusEmitSignal      = lz[func(uintptr, uintptr, unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, uintptr, unsafe.Pointer) int32]{sym: "g_dbus_connection_emit_signal"}
	gVariantNewInt64     = lz[func(int64) uintptr]{sym: "g_variant_new_int64"}
	gVariantNewVariant   = lz[func(uintptr) uintptr]{sym: "g_variant_new_variant"}
	gVariantNewDictEntry = lz[func(uintptr, uintptr) uintptr]{sym: "g_variant_new_dict_entry"}
	gVariantNewArray     = lz[func(uintptr, unsafe.Pointer, uintptr) uintptr]{sym: "g_variant_new_array"}
)

// Looked up once: a desktop without a session bus costs one failed lookup, not one per update.
var launcherBus struct {
	conn  uintptr
	tried bool
}

// LauncherEntryUpdate emits com.canonical.Unity.LauncherEntry.Update (count, count-visible) for
// appURI "application://<desktop file id>" on the session bus; without one it does nothing.
func LauncherEntryUpdate(appURI string, count int64, visible bool) {
	if !launcherBus.tried {
		launcherBus.tried = true
		var gerr uintptr
		launcherBus.conn = gBusGetSync.get()(OSG_BUS_TYPE_SESSION, 0, unsafe.Pointer(&gerr))
		if gerr != 0 {
			OSG_error_free(int64(gerr))
		}
	}
	if launcherBus.conn == 0 {
		return
	}
	entry := func(key string, value uintptr) uintptr {
		return gVariantNewDictEntry.get()(uintptr(OSG_variant_new_string(cs(key))), gVariantNewVariant.get()(value))
	}
	props := []uintptr{
		entry("count", gVariantNewInt64.get()(count)),
		entry("count-visible", uintptr(OSG_variant_new_boolean(visible))),
	}
	// (sa{sv}); every constructor takes the floating references of its children, emit_signal the tuple's.
	params := OSG_variant_new_tuple([]int64{
		OSG_variant_new_string(cs(appURI)),
		int64(gVariantNewArray.get()(0, sp(props), uintptr(len(props)))),
	}, 2)
	var gerr uintptr
	gDBusEmitSignal.get()(launcherBus.conn, 0, sp(cs("/com/canonical/unity/launcherentry/gowt")),
		sp(cs("com.canonical.Unity.LauncherEntry")), sp(cs("Update")), uintptr(params), unsafe.Pointer(&gerr))
	if gerr != 0 {
		OSG_error_free(int64(gerr))
	}
}
