// Hand-written stand-ins for the package dnd classes StyledText uses (tooling/j2go/stubs-dnd/org/eclipse/swt/dnd has the
// Java declarations): package dnd is not translated yet. Clipboard keeps its contents in the process.
package swt

const (
	DNDCLIPBOARD                  int32 = 1 << 0
	DNDSELECTION_CLIPBOARD        int32 = 1 << 1
	DNDERROR_CANNOT_SET_CLIPBOARD int32 = 2002
)

// The field keeps the instances at distinct addresses: Clipboard keys its contents by *Transfer.
type Transfer struct{ _ byte }

type TextTransfer struct{ Transfer }
type RTFTransfer struct{ Transfer }
type HTMLTransfer struct{ Transfer }

var (
	textTransferInstance = &TextTransfer{}
	rtfTransferInstance  = &RTFTransfer{}
	htmlTransferInstance = &HTMLTransfer{}
)

func TextTransferGetInstance() *TextTransfer { return textTransferInstance }
func RTFTransferGetInstance() *RTFTransfer   { return rtfTransferInstance }
func HTMLTransferGetInstance() *HTMLTransfer { return htmlTransferInstance }

type Clipboard struct{}

var clipboardContents = map[int32]map[*Transfer]any{}

func NewClipboard(display *Display) *Clipboard { return &Clipboard{} }

func (c *Clipboard) SetContents(data []any, dataTypes []*Transfer, clipboardType int32) {
	m := map[*Transfer]any{}
	for i, t := range dataTypes {
		m[t] = data[i]
	}
	clipboardContents[clipboardType] = m
}

func (c *Clipboard) GetContents(transfer *Transfer, clipboardType int32) any {
	return clipboardContents[clipboardType][transfer]
}

func (c *Clipboard) SetContentsDataDataTypes(data []any, dataTypes []*Transfer) {
	c.SetContents(data, dataTypes, DNDCLIPBOARD)
}

func (c *Clipboard) GetContentsTransfer(transfer *Transfer) any {
	return c.GetContents(transfer, DNDCLIPBOARD)
}

func (c *Clipboard) Dispose() {}

type StyledTextDropTargetEffect struct{}

func NewStyledTextDropTargetEffect(styledText *StyledText) *StyledTextDropTargetEffect {
	return &StyledTextDropTargetEffect{}
}
