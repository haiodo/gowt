package swt

import "github.com/haiodo/gowt/internal/jrt"

// The GTK 4 half of Eclipse SWT Drag and Drop/gtk (ContentProviders, ClipboardProxyGTK4) is not ported: Linux runs
// GTK 3, and every branch that reaches these is guarded by GTK.GTK4. Like the other GTK 4 names they compile and
// panic when called.

const gtk4DndUnsupported = "gowt: GTK 4 drag and drop is not ported"

type ContentProviders struct{}

// ContentProviders.CLIPBOARD_DATA, a nested Java enum.
type ContentProviders_CLIPBOARD_DATA int32

const (
	ContentProviders_CLIPBOARD_DATACLIPBOARD ContentProviders_CLIPBOARD_DATA = iota
	ContentProviders_CLIPBOARD_DATAPRIMARYCLIPBOARD
	ContentProviders_CLIPBOARD_DATADRAG
)

func ContentProvidersGetInstance() *ContentProviders { panic(gtk4DndUnsupported) }

func (c *ContentProviders) GetGType(transfer *Transfer) int64 { panic(gtk4DndUnsupported) }

func (c *ContentProviders) GetObject(gvalue int64) any { panic(gtk4DndUnsupported) }

func (c *ContentProviders) RegisterType(formatName string) int32 { panic(gtk4DndUnsupported) }

func (c *ContentProviders) CreateContentProviders(data []any, transfers []*Transfer, clipboard ContentProviders_CLIPBOARD_DATA) int64 {
	panic(gtk4DndUnsupported)
}

type ClipboardProxyGTK4 struct{}

func ClipboardProxyGTK4_getInstance(display *Display) *ClipboardProxyGTK4 { panic(gtk4DndUnsupported) }

func (p *ClipboardProxyGTK4) Clear(owner *Clipboard, clipboards int32, store bool) {
	panic(gtk4DndUnsupported)
}

func (p *ClipboardProxyGTK4) GetData(owner *Clipboard, transfer *Transfer, clipboards int32) *jrt.CompletableFuture {
	panic(gtk4DndUnsupported)
}

func (p *ClipboardProxyGTK4) SetData(owner *Clipboard, data []any, dataTypes []*Transfer, clipboards int32) bool {
	panic(gtk4DndUnsupported)
}
