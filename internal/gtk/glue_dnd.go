//go:build linux

package gtk

// The content-provider helpers of ContentProviders.java (GTK 4 only) have no GIR counterpart. Linux runs
// GTK 3; like the other GTK 4 names these compile and panic when called.

func OSContent_providers_create_gtype(name string) int64 {
	panic("gowt: GTK 4 content providers are not ported")
}

func OSContent_providers_create_gvalue(gtype int64, sourceId int64) int64 {
	panic("gowt: GTK 4 content providers are not ported")
}
