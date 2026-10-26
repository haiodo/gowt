// Package dbus is a small session bus client written from the D-Bus specification: just enough
// to own a well-known name and to call or serve org.freedesktop.Application.Open and Activate.
// Little-endian only.
package dbus

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	typeCall   = 1
	typeReturn = 2
	typeError  = 3

	flagNoReply = 1

	fieldPath, fieldIface, fieldMember, fieldErrName   = 1, 2, 3, 4
	fieldReplySerial, fieldDest, fieldSender, fieldSig = 5, 6, 7, 8

	appIface  = "org.freedesktop.Application"
	busName   = "org.freedesktop.DBus"
	maxMsg    = 1 << 24
	callLimit = 3 * time.Second
)

// Conn is a session bus connection. One goroutine at a time: setup calls first, then Serve.
type Conn struct {
	c      net.Conn
	r      *bufio.Reader
	serial uint32
	name   string
}

type message struct {
	typ, flags                                 byte
	serial, replySerial                        uint32
	path, iface, member, dest, sender, errName string
	sig                                        string
	body                                       []byte
}

// PathOf returns the object path of an application id: dots become slashes, dashes underscores.
func PathOf(id string) string {
	return "/" + strings.NewReplacer(".", "/", "-", "_").Replace(id)
}

// Connect opens the session bus, authenticates with EXTERNAL and says Hello.
// A missing bus is an error.
func Connect() (*Conn, error) {
	nc, err := dialSession()
	if err != nil {
		return nil, err
	}
	nc.SetDeadline(time.Now().Add(callLimit))
	c := &Conn{c: nc, r: bufio.NewReader(nc)}
	if err := c.auth(); err != nil {
		nc.Close()
		return nil, err
	}
	if _, err := c.call(busName, "/org/freedesktop/DBus", busName, "Hello", "", nil); err != nil {
		nc.Close()
		return nil, err
	}
	return c, nil
}

func dialSession() (net.Conn, error) {
	addr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if addr == "" {
		addr = "unix:path=/run/user/" + strconv.Itoa(os.Getuid()) + "/bus"
	}
	var last error = errors.New("dbus: no unix transport in " + addr)
	for _, a := range strings.Split(addr, ";") {
		rest, ok := strings.CutPrefix(a, "unix:")
		if !ok {
			continue
		}
		for _, kv := range strings.Split(rest, ",") {
			k, v, _ := strings.Cut(kv, "=")
			v, _ = url.PathUnescape(v)
			var path string
			switch k {
			case "path":
				path = v
			case "abstract":
				path = "@" + v
			default:
				continue
			}
			nc, err := net.DialTimeout("unix", path, callLimit)
			if err == nil {
				return nc, nil
			}
			last = err
		}
	}
	return nil, last
}

func (c *Conn) auth() error {
	uid := hex.EncodeToString([]byte(strconv.Itoa(os.Getuid())))
	if _, err := io.WriteString(c.c, "\x00AUTH EXTERNAL "+uid+"\r\n"); err != nil {
		return err
	}
	line, err := c.r.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "OK") {
		return fmt.Errorf("dbus: auth refused: %q", strings.TrimSpace(line))
	}
	_, err = io.WriteString(c.c, "BEGIN\r\n")
	return err
}

// Close closes the connection, which releases the names it owns.
func (c *Conn) Close() error { return c.c.Close() }

// RequestName asks for name without queueing; true means this connection owns it now.
func (c *Conn) RequestName(name string) (bool, error) {
	var b enc
	b.str(name)
	b.u32(4) // DBUS_NAME_FLAG_DO_NOT_QUEUE
	rep, err := c.call(busName, "/org/freedesktop/DBus", busName, "RequestName", "su", b.b)
	if err != nil {
		return false, err
	}
	d := dec{b: rep.body}
	switch v := d.u32(); v {
	case 1, 4: // primary owner, already owner
		c.name = name
		return true, nil
	case 2, 3: // in queue, exists
		return false, nil
	default:
		return false, fmt.Errorf("dbus: RequestName reply %d", v)
	}
}

// Open calls org.freedesktop.Application.Open on the owner of name, or Activate when uris is empty.
func (c *Conn) Open(name string, uris []string) error {
	var b enc
	member, sig := "Activate", "a{sv}"
	if len(uris) > 0 {
		member, sig = "Open", "asa{sv}"
		b.strs(uris)
	}
	b.u32(0) // empty a{sv}
	b.align(8)
	_, err := c.call(name, PathOf(name), appIface, member, sig, b.b)
	return err
}

// Serve answers the calls that reach the owned name until the connection fails. onOpen gets the URIs
// of Open, and nil for Activate.
func (c *Conn) Serve(onOpen func(uris []string)) error {
	c.c.SetDeadline(time.Time{})
	for {
		m, err := c.read()
		if err != nil {
			return err
		}
		if m.typ != typeCall {
			continue
		}
		var out enc
		sig := ""
		switch m.iface + "." + m.member {
		case appIface + ".Open":
			d := dec{b: m.body}
			onOpen(d.strs())
		case appIface + ".Activate":
			onOpen(nil)
		case appIface + ".ActivateAction":
		case "org.freedesktop.DBus.Peer.Ping":
		case "org.freedesktop.DBus.Peer.GetMachineId":
			out.str(machineID())
			sig = "s"
		case "org.freedesktop.DBus.Introspectable.Introspect":
			out.str(introspect)
			sig = "s"
		default:
			if m.flags&flagNoReply == 0 {
				var e enc
				e.str("no method " + m.iface + "." + m.member)
				err = c.send(&message{typ: typeError, flags: flagNoReply, errName: "org.freedesktop.DBus.Error.UnknownMethod", replySerial: m.serial, dest: m.sender, sig: "s", body: e.b})
			}
			if err != nil {
				return err
			}
			continue
		}
		if m.flags&flagNoReply != 0 {
			continue
		}
		if err := c.send(&message{typ: typeReturn, flags: flagNoReply, replySerial: m.serial, dest: m.sender, sig: sig, body: out.b}); err != nil {
			return err
		}
	}
}

const introspect = `<node><interface name="org.freedesktop.Application">` +
	`<method name="Activate"><arg type="a{sv}" direction="in"/></method>` +
	`<method name="Open"><arg type="as" direction="in"/><arg type="a{sv}" direction="in"/></method>` +
	`<method name="ActivateAction"><arg type="s" direction="in"/><arg type="av" direction="in"/><arg type="a{sv}" direction="in"/></method>` +
	`</interface></node>`

func machineID() string {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// call sends a method call and waits for its reply; an error reply becomes an error.
func (c *Conn) call(dest, path, iface, member, sig string, body []byte) (*message, error) {
	m := &message{typ: typeCall, dest: dest, path: path, iface: iface, member: member, sig: sig, body: body}
	if err := c.send(m); err != nil {
		return nil, err
	}
	for {
		r, err := c.read()
		if err != nil {
			return nil, err
		}
		if r.replySerial != m.serial || (r.typ != typeReturn && r.typ != typeError) {
			continue
		}
		if r.typ == typeError {
			text := ""
			if r.sig == "s" {
				d := dec{b: r.body}
				text = d.str()
			}
			return nil, fmt.Errorf("dbus: %s: %s", r.errName, text)
		}
		return r, nil
	}
}

func (c *Conn) send(m *message) error {
	c.serial++
	m.serial = c.serial
	var e enc
	e.b = append(e.b, 'l', m.typ, m.flags, 1)
	e.u32(uint32(len(m.body)))
	e.u32(m.serial)
	lenAt := len(e.b)
	e.u32(0)
	e.align(8)
	start := len(e.b)
	field := func(code byte, sig string, put func()) {
		e.align(8)
		e.b = append(e.b, code)
		e.sigStr(sig)
		put()
	}
	for _, f := range []struct {
		code byte
		sig  string
		s    string
	}{{fieldPath, "o", m.path}, {fieldIface, "s", m.iface}, {fieldMember, "s", m.member}, {fieldErrName, "s", m.errName}, {fieldDest, "s", m.dest}} {
		if f.s != "" {
			field(f.code, f.sig, func() { e.str(f.s) })
		}
	}
	if m.typ == typeReturn || m.typ == typeError {
		field(fieldReplySerial, "u", func() { e.u32(m.replySerial) })
	}
	if m.sig != "" {
		field(fieldSig, "g", func() { e.sigStr(m.sig) })
	}
	binary.LittleEndian.PutUint32(e.b[lenAt:], uint32(len(e.b)-start))
	e.align(8)
	e.b = append(e.b, m.body...)
	c.c.SetWriteDeadline(time.Now().Add(callLimit))
	_, err := c.c.Write(e.b)
	return err
}

func (c *Conn) read() (*message, error) {
	var fixed [16]byte
	if _, err := io.ReadFull(c.r, fixed[:]); err != nil {
		return nil, err
	}
	if fixed[0] != 'l' {
		return nil, errors.New("dbus: big-endian message")
	}
	bodyLen := binary.LittleEndian.Uint32(fixed[4:])
	fieldsLen := binary.LittleEndian.Uint32(fixed[12:])
	if bodyLen > maxMsg || fieldsLen > maxMsg {
		return nil, errors.New("dbus: message too large")
	}
	hdr := make([]byte, (16+int(fieldsLen)+7)&^7+int(bodyLen))
	copy(hdr, fixed[:])
	if _, err := io.ReadFull(c.r, hdr[16:]); err != nil {
		return nil, err
	}
	m := &message{typ: fixed[1], flags: fixed[2], serial: binary.LittleEndian.Uint32(fixed[8:])}
	d := dec{b: hdr[:16+fieldsLen], off: 16}
	for d.off < len(d.b) {
		d.align(8)
		if d.off >= len(d.b) {
			break
		}
		code := d.byte()
		sig := d.sigStr()
		switch sig {
		case "s", "o":
			s := d.str()
			switch code {
			case fieldPath:
				m.path = s
			case fieldIface:
				m.iface = s
			case fieldMember:
				m.member = s
			case fieldErrName:
				m.errName = s
			case fieldDest:
				m.dest = s
			case fieldSender:
				m.sender = s
			}
		case "u":
			m.replySerial = d.u32()
		case "g":
			m.sig = d.sigStr()
		default:
			return nil, errors.New("dbus: unknown header field type " + sig)
		}
	}
	m.body = hdr[(16+int(fieldsLen)+7)&^7:]
	return m, nil
}

type enc struct{ b []byte }

func (e *enc) align(n int) {
	for len(e.b)%n != 0 {
		e.b = append(e.b, 0)
	}
}

func (e *enc) u32(v uint32) {
	e.align(4)
	e.b = binary.LittleEndian.AppendUint32(e.b, v)
}

func (e *enc) str(s string) {
	e.u32(uint32(len(s)))
	e.b = append(append(e.b, s...), 0)
}

func (e *enc) sigStr(s string) { e.b = append(append(e.b, byte(len(s))), append([]byte(s), 0)...) }

func (e *enc) strs(ss []string) {
	e.u32(0)
	lenAt, start := len(e.b)-4, len(e.b)
	for _, s := range ss {
		e.str(s)
	}
	binary.LittleEndian.PutUint32(e.b[lenAt:], uint32(len(e.b)-start))
}

// dec reads with the alignment of d.b[0]; out-of-range reads give zero values and stop at the end.
type dec struct {
	b   []byte
	off int
}

func (d *dec) align(n int) { d.off = (d.off + n - 1) &^ (n - 1) }

func (d *dec) byte() byte {
	if d.off >= len(d.b) {
		return 0
	}
	d.off++
	return d.b[d.off-1]
}

func (d *dec) u32() uint32 {
	d.align(4)
	if d.off+4 > len(d.b) {
		d.off = len(d.b)
		return 0
	}
	d.off += 4
	return binary.LittleEndian.Uint32(d.b[d.off-4:])
}

func (d *dec) str() string {
	n := int(d.u32())
	if n < 0 || d.off+n+1 > len(d.b) {
		d.off = len(d.b)
		return ""
	}
	s := string(d.b[d.off : d.off+n])
	d.off += n + 1
	return s
}

func (d *dec) sigStr() string {
	n := int(d.byte())
	if d.off+n+1 > len(d.b) {
		d.off = len(d.b)
		return ""
	}
	s := string(d.b[d.off : d.off+n])
	d.off += n + 1
	return s
}

func (d *dec) strs() []string {
	n := int(d.u32())
	end := d.off + n
	if end > len(d.b) {
		end = len(d.b)
	}
	var out []string
	for d.off < end {
		out = append(out, d.str())
	}
	return out
}
