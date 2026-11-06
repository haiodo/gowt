// Command clipboardpeer is the cross-application clipboard owner for the SWT
// clipboard tests: a separate process with its own Display and Clipboard,
// driven by the test process over JSON lines on stdin/stdout. It replaces the
// Swing/RMI peer of the upstream SWT tests; see tests/swttests/clipboard_manual.go.
//
// One request per line, one reply per line. Bytes (image, my_type_name) travel
// base64 encoded. clipboard is 1 (CLIPBOARD, the default) or 2 (SELECTION,
// GTK only).
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

// myTypeName is the custom transfer type of the ByteArrayTransfer tests; both
// processes must register the same name for the type ids to match.
const myTypeName = "my_type_name"

type myTypeTransfer struct{ *swt.ByteArrayTransfer }

var myTypeID = swt.TransferRegisterType(myTypeName)

func (m *myTypeTransfer) GetTypeIds_() []int32    { return []int32{myTypeID} }
func (m *myTypeTransfer) GetTypeNames_() []string { return []string{myTypeName} }

// The Swing peer asks GTK for UTF8_STRING only. Requesting every TextTransfer
// alias would repeat the expected warning in the disposed-display test.
type utf8TextTransfer struct{ *swt.TextTransfer }

func (t *utf8TextTransfer) GetTypeIds_() []int32 {
	return []int32{swt.TransferRegisterType("UTF8_STRING")}
}
func (t *utf8TextTransfer) GetTypeNames_() []string { return []string{"UTF8_STRING"} }

type request struct {
	Cmd       string   `json:"cmd"`
	Value     string   `json:"value,omitempty"`
	Files     []string `json:"files,omitempty"`
	Clipboard int32    `json:"clipboard,omitempty"`
}

type reply struct {
	OK    bool     `json:"ok"`
	Value string   `json:"value,omitempty"`
	Files []string `json:"files,omitempty"`
	Error string   `json:"error,omitempty"`
}

type peer struct {
	display *swt.Display
	clip    *swt.Clipboard
	text    *swt.Transfer
	myType  *myTypeTransfer
}

func main() {
	p := &peer{}
	p.display = swt.NewDisplay()
	p.clip = swt.NewClipboard(p.display)
	p.text = swt.TextTransferGetInstance().AsTransfer()
	if runtime.GOOS == "linux" {
		text := &utf8TextTransfer{swt.TextTransferGetInstance()}
		text.SetImpl_(text)
		p.text = text.AsTransfer()
	}
	p.myType = &myTypeTransfer{swt.NewByteArrayTransfer()}
	p.myType.SetImpl_(p.myType)

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 16<<20)
	out := bufio.NewWriter(os.Stdout)
	stopping := false
	go func() {
		for in.Scan() {
			line := append([]byte(nil), in.Bytes()...)
			p.display.AsyncExec(jrt.NewRunnable(func() {
				var req request
				if err := json.Unmarshal(line, &req); err != nil {
					write(out, reply{Error: err.Error()})
					out.Flush()
					return
				}
				if req.Cmd == "quit" {
					write(out, reply{OK: true})
					out.Flush()
					stopping = true
					return
				}
				write(out, p.handle(&req))
				out.Flush()
			}))
		}
		p.display.AsyncExec(jrt.NewRunnable(func() {
			stopping = true
		}))
	}()
	for !stopping {
		if !p.display.ReadAndDispatch() {
			p.display.Sleep()
		}
	}
	p.clip.Dispose()
	p.display.Dispose()
}

func write(out *bufio.Writer, r reply) {
	b, err := json.Marshal(r)
	if err != nil {
		b, _ = json.Marshal(reply{Error: err.Error()})
	}
	out.Write(b)
	out.WriteByte('\n')
}

func clipboardID(req *request) int32 {
	if req.Clipboard == 0 {
		return swt.DNDCLIPBOARD
	}
	return req.Clipboard
}

func (p *peer) handle(req *request) (res reply) {
	defer func() {
		if r := recover(); r != nil {
			res = reply{Error: fmt.Sprint(r)}
		}
	}()
	switch req.Cmd {
	case "set_text":
		p.clip.SetContentsDataDataTypesClipboards([]any{req.Value},
			[]*swt.Transfer{p.text}, clipboardID(req))
	case "get_text":
		if s, ok := p.clip.GetContentsTransferClipboards(p.text, clipboardID(req)).(string); ok {
			res.Value = s
		}
	case "set_rtf":
		p.clip.SetContents([]any{req.Value}, []*swt.Transfer{swt.RTFTransferGetInstance().AsTransfer()})
	case "get_rtf":
		if s, ok := p.clip.GetContents(swt.RTFTransferGetInstance().AsTransfer()).(string); ok {
			res.Value = s
		}
	case "set_html":
		p.clip.SetContents([]any{req.Value}, []*swt.Transfer{swt.HTMLTransferGetInstance().AsTransfer()})
	case "get_html":
		if s, ok := p.clip.GetContents(swt.HTMLTransferGetInstance().AsTransfer()).(string); ok {
			res.Value = s
		}
	case "set_url":
		p.clip.SetContents([]any{req.Value}, []*swt.Transfer{swt.URLTransferGetInstance().AsTransfer()})
	case "get_url":
		if s, ok := p.clip.GetContents(swt.URLTransferGetInstance().AsTransfer()).(string); ok {
			res.Value = s
		}
	case "set_files":
		p.clip.SetContents([]any{req.Files}, []*swt.Transfer{swt.FileTransferGetInstance().AsTransfer()})
	case "get_files":
		if f, ok := p.clip.GetContents(swt.FileTransferGetInstance().AsTransfer()).([]string); ok {
			res.Files = f
		}
	case "set_image":
		b, err := base64.StdEncoding.DecodeString(req.Value)
		if err != nil {
			return reply{Error: err.Error()}
		}
		data := swt.NewImageDataStream(jrt.NewByteArrayInputStream(int8s(b)))
		p.clip.SetContents([]any{data}, []*swt.Transfer{swt.ImageTransferGetInstance().AsTransfer()})
	case "get_image":
		if d, ok := p.clip.GetContents(swt.ImageTransferGetInstance().AsTransfer()).(*swt.ImageData); ok && d != nil {
			out := jrt.NewByteArrayOutputStream()
			loader := swt.NewImageLoader()
			loader.Data = []*swt.ImageData{d}
			loader.Save(out, swt.IMAGE_PNG)
			res.Value = base64.StdEncoding.EncodeToString(uint8s(out.ToByteArray()))
		}
	case "set_mine":
		b, err := base64.StdEncoding.DecodeString(req.Value)
		if err != nil {
			return reply{Error: err.Error()}
		}
		p.clip.SetContents([]any{int8s(b)}, []*swt.Transfer{p.myType.AsTransfer()})
	case "get_mine":
		if b, ok := p.clip.GetContents(p.myType.AsTransfer()).([]int8); ok {
			res.Value = base64.StdEncoding.EncodeToString(uint8s(b))
		}
	case "clear":
		p.clip.ClearContentsClipboards(clipboardID(req))
	case "ping":
	default:
		return reply{Error: "unknown command " + req.Cmd}
	}
	res.OK = true
	return res
}

func int8s(b []byte) []int8 {
	s := make([]int8, len(b))
	for i, v := range b {
		s[i] = int8(v)
	}
	return s
}

func uint8s(s []int8) []byte {
	b := make([]byte, len(s))
	for i, v := range s {
		b[i] = byte(v)
	}
	return b
}
