package swt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
)

// The virtual clipboard of a QA run: while qaClipboardPath names a file, a Clipboard reads and writes
// that JSON file, never the system clipboard. port.sh makes the Clipboard methods call these.
// The X11 selection is not kept: it reads empty and writes to it are dropped.

// qaClipFormats is the file; qa.Clipboard is the same struct for tests, keep the tags equal.
type qaClipFormats struct {
	Text   string            `json:"text,omitempty"`
	RTF    string            `json:"rtf,omitempty"`
	HTML   string            `json:"html,omitempty"`
	URL    string            `json:"url,omitempty"`
	Image  []byte            `json:"image,omitempty"` // PNG
	Files  []string          `json:"files,omitempty"`
	Custom map[string][]byte `json:"custom,omitempty"` // by the first type name of the Transfer
}

func qaClipLoad() qaClipFormats {
	var f qaClipFormats
	if b, err := os.ReadFile(qaClipboardPath()); err == nil {
		json.Unmarshal(b, &f)
	}
	return f
}

func qaClipSave(f qaClipFormats) {
	p := qaClipboardPath()
	b, err := json.Marshal(f)
	if err == nil {
		tmp := p + ".tmp"
		if err = os.WriteFile(tmp, b, 0o600); err == nil {
			err = os.Rename(tmp, p)
		}
	}
	if err != nil {
		DNDError(DNDERROR_CANNOT_SET_CLIPBOARD)
	}
}

// qaClipKey names the format of a Transfer: "text", "rtf", "html", "url", "image", "files", or
// "custom:" and its first type name.
func qaClipKey(t *Transfer) string {
	switch t {
	case TextTransferGetInstance().AsTransfer():
		return "text"
	case RTFTransferGetInstance().AsTransfer():
		return "rtf"
	case HTMLTransferGetInstance().AsTransfer():
		return "html"
	case URLTransferGetInstance().AsTransfer():
		return "url"
	case ImageTransferGetInstance().AsTransfer():
		return "image"
	case FileTransferGetInstance().AsTransfer():
		return "files"
	}
	if names := t.GetTypeNames(); len(names) > 0 {
		return "custom:" + names[0]
	}
	return "custom:"
}

var qaClipSeq atomic.Int64

func qaClipTemp() string {
	return qaClipboardPath() + ".img" + strconv.FormatInt(qaClipSeq.Add(1), 10) + ".png"
}

func qaImagePNG(d *ImageData) []byte {
	tmp := qaClipTemp()
	defer os.Remove(tmp)
	l := NewImageLoader()
	l.Data = []*ImageData{d}
	l.SaveFilenameFormat(tmp, IMAGE_PNG)
	b, _ := os.ReadFile(tmp)
	return b
}

func qaPNGImage(b []byte) *ImageData {
	tmp := qaClipTemp()
	defer os.Remove(tmp)
	if os.WriteFile(tmp, b, 0o600) != nil {
		return nil
	}
	if imgs := NewImageLoader().LoadFilename(tmp); len(imgs) > 0 {
		return imgs[0]
	}
	return nil
}

func (this *Clipboard) qaGet(transferLike TransferLike, clipboards int32) any {
	this.CheckWidget()
	if transferLike == nil {
		DNDError(ERROR_NULL_ARGUMENT)
	}
	if clipboards&DNDCLIPBOARD == 0 {
		return nil
	}
	f, key := qaClipLoad(), qaClipKey(transferLike.AsTransfer())
	switch key {
	case "text":
		if f.Text != "" {
			return f.Text
		}
	case "rtf":
		if f.RTF != "" {
			return f.RTF
		}
	case "html":
		if f.HTML != "" {
			return f.HTML
		}
	case "url":
		if f.URL != "" {
			return f.URL
		}
	case "image":
		if f.Image != nil {
			if d := qaPNGImage(f.Image); d != nil {
				return d
			}
		}
	case "files":
		if len(f.Files) > 0 {
			return f.Files
		}
	default:
		if b := f.Custom[key[len("custom:"):]]; b != nil {
			out := make([]int8, len(b))
			for i, c := range b {
				out[i] = int8(c)
			}
			return out
		}
	}
	return nil
}

func (this *Clipboard) qaSet(data []any, dataTypes []*Transfer, clipboards int32) {
	this.CheckWidget()
	if data == nil || dataTypes == nil || len(data) != len(dataTypes) || len(data) == 0 {
		DNDError(ERROR_INVALID_ARGUMENT)
	}
	var f qaClipFormats
	for i, d := range data {
		if d == nil || dataTypes[i] == nil {
			DNDError(ERROR_INVALID_ARGUMENT)
		}
		key := qaClipKey(dataTypes[i])
		ok := false
		switch key {
		case "text":
			f.Text, ok = d.(string)
		case "rtf":
			f.RTF, ok = d.(string)
		case "html":
			f.HTML, ok = d.(string)
		case "url":
			f.URL, ok = d.(string)
		case "image":
			var img *ImageData
			if img, ok = d.(*ImageData); ok {
				f.Image = qaImagePNG(img)
			}
		case "files":
			f.Files, ok = d.([]string)
		default:
			var b []int8
			if b, ok = d.([]int8); ok {
				if f.Custom == nil {
					f.Custom = map[string][]byte{}
				}
				bs := make([]byte, len(b))
				for j, c := range b {
					bs[j] = byte(c)
				}
				f.Custom[key[len("custom:"):]] = bs
			}
		}
		if !ok {
			DNDError(ERROR_INVALID_ARGUMENT)
		}
	}
	if clipboards&DNDCLIPBOARD != 0 {
		qaClipSave(f)
	}
}

func (this *Clipboard) qaClear(clipboards int32) {
	this.CheckWidget()
	if clipboards&DNDCLIPBOARD != 0 {
		os.Remove(qaClipboardPath())
	}
}

// qaClipTransfers are the Transfers of the formats the file holds, and the names of the custom ones.
func qaClipTransfers() (ts []*Transfer, custom []string) {
	f := qaClipLoad()
	for _, p := range []struct {
		on bool
		t  *Transfer
	}{
		{f.Text != "", TextTransferGetInstance().AsTransfer()},
		{f.RTF != "", RTFTransferGetInstance().AsTransfer()},
		{f.HTML != "", HTMLTransferGetInstance().AsTransfer()},
		{f.URL != "", URLTransferGetInstance().AsTransfer()},
		{f.Image != nil, ImageTransferGetInstance().AsTransfer()},
		{len(f.Files) > 0, FileTransferGetInstance().AsTransfer()},
	} {
		if p.on {
			ts = append(ts, p.t)
		}
	}
	for name := range f.Custom {
		custom = append(custom, name)
	}
	return ts, custom
}

func (this *Clipboard) qaTypes(clipboards int32) []*TransferData {
	this.CheckWidget()
	out := make([]*TransferData, 0)
	if clipboards&DNDCLIPBOARD != 0 {
		ts, _ := qaClipTransfers()
		for _, t := range ts {
			out = append(out, t.GetSupportedTypes()...)
		}
	}
	return out
}

func (this *Clipboard) qaTypeNames() []string {
	this.CheckWidget()
	out := make([]string, 0)
	ts, custom := qaClipTransfers()
	for _, t := range ts {
		out = append(out, t.GetTypeNames()...)
	}
	return append(out, custom...)
}

// qaClipboardDir is where the virtual clipboard keeps its file.
func qaClipboardDir(data string) string {
	if data == "" {
		data = os.TempDir()
	}
	return filepath.Join(data, "clipboard.json")
}
