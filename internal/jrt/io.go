package jrt

import (
	"bytes"
	"io"
	"os"
)

// InputStream mirrors the java.io.InputStream methods ImageLoader.java's translated code uses:
// Read/ReadRange return -1 at end of stream, matching Java's read()/read(byte[],off,len). []int8
// is this codebase's Go spelling of byte[] (see GoTypes). None of the ported code catches a
// per-call I/O error, so there is no error return - a real failure panics with an IOException.
type InputStream interface {
	Read() int32
	ReadRange(b []int8, off, length int32) int32
	Close()
}

// OutputStream mirrors write(int)/write(byte[],off,len)/flush()/close().
type OutputStream interface {
	Write(b int32)
	WriteRange(b []int8, off, length int32)
	Flush()
	Close()
}

// readerInputStream keeps a scratch []byte buffer, reused across calls, to bridge Go's io.Reader
// (which wants []byte) and the Java-shaped ReadRange (which wants []int8).
type readerInputStream struct {
	r   io.Reader
	buf []byte
}

// NewInputStream wraps a Go io.Reader - a file, an embedded resource, a byte slice - as the
// InputStream ImageLoader.java's translated code and the image codec wrapper read from.
func NewInputStream(r io.Reader) InputStream { return &readerInputStream{r: r} }

func (s *readerInputStream) Read() int32 {
	var b [1]byte
	n, _ := s.r.Read(b[:])
	if n == 0 {
		return -1
	}
	return int32(b[0])
}

func (s *readerInputStream) ReadRange(b []int8, off, length int32) int32 {
	if length == 0 {
		return 0
	}
	if int32(cap(s.buf)) < length {
		s.buf = make([]byte, length)
	}
	n, _ := s.r.Read(s.buf[:length])
	if n == 0 {
		return -1
	}
	for i := 0; i < n; i++ {
		b[off+int32(i)] = int8(s.buf[i])
	}
	return int32(n)
}

func (s *readerInputStream) Close() {
	if c, ok := s.r.(io.Closer); ok {
		c.Close()
	}
}

// asWriter adapts a Java-shaped OutputStream to a real io.Writer: image/png, image/jpeg and
// x/image/bmp all encode to a plain io.Writer, which an OutputStream isn't.
type asWriter struct {
	s   OutputStream
	buf []int8
}

func AsWriter(s OutputStream) io.Writer { return &asWriter{s: s} }

func (a *asWriter) Write(p []byte) (int, error) {
	if int32(cap(a.buf)) < int32(len(p)) {
		a.buf = make([]int8, len(p))
	}
	buf := a.buf[:len(p)]
	for i, b := range p {
		buf[i] = int8(b)
	}
	a.s.WriteRange(buf, 0, int32(len(p)))
	return len(p), nil
}

// writerOutputStream is readerInputStream's mirror: a scratch []byte buffer bridging io.Writer
// and the Java-shaped WriteRange.
type writerOutputStream struct {
	w   io.Writer
	buf []byte
}

// NewOutputStream wraps a Go io.Writer as the OutputStream LEDataOutputStream/the format encoders
// write to.
func NewOutputStream(w io.Writer) OutputStream { return &writerOutputStream{w: w} }

func (s *writerOutputStream) Write(b int32) { s.w.Write([]byte{byte(b)}) }

func (s *writerOutputStream) WriteRange(b []int8, off, length int32) {
	if int32(cap(s.buf)) < length {
		s.buf = make([]byte, length)
	}
	buf := s.buf[:length]
	for i := int32(0); i < length; i++ {
		buf[i] = byte(b[off+i])
	}
	s.w.Write(buf)
}

func (s *writerOutputStream) Flush() {
	if f, ok := s.w.(interface{ Flush() error }); ok {
		f.Flush()
	}
}

func (s *writerOutputStream) Close() {
	if c, ok := s.w.(io.Closer); ok {
		c.Close()
	}
}

// IOException is java.io.IOException: a pointer so the panic value satisfies Go's error
// interface (see ControlFlowEmitter's concrete-catch type assertion).
type IOException struct{}

func NewIOException() *IOException { return &IOException{} }

func (e *IOException) Error() string { return "IOException" }

// ReadAllBytes is java.io.InputStream.readAllBytes(): read until EOF.
func ReadAllBytes(s InputStream) []int8 {
	var out []int8
	buf := make([]int8, 4096)
	for {
		n := s.ReadRange(buf, 0, int32(len(buf)))
		if n <= 0 {
			break
		}
		out = append(out, buf[:n]...)
	}
	return out
}

// NewFileInputStream/NewFileOutputStream back `new FileInputStream(filename)`/`new
// FileOutputStream(filename)` (ImageLoader.load/save(String)) - "FileInputStream"/
// "FileOutputStream" name no real Go type (see Manual), only these two constructors exist.
func NewFileInputStream(filename string) InputStream {
	f, err := os.Open(filename)
	if err != nil {
		panic(NewIOException())
	}
	return NewInputStream(f)
}

func NewFileOutputStream(filename string) OutputStream {
	f, err := os.Create(filename)
	if err != nil {
		panic(NewIOException())
	}
	return NewOutputStream(f)
}

// ByteArrayInputStream is java.io.ByteArrayInputStream.
type ByteArrayInputStream struct{ readerInputStream }

func NewByteArrayInputStream(buf []int8) *ByteArrayInputStream {
	b := make([]byte, len(buf))
	for i, v := range buf {
		b[i] = byte(v)
	}
	return &ByteArrayInputStream{readerInputStream{r: bytes.NewReader(b)}}
}

// ByteArrayOutputStream is java.io.ByteArrayOutputStream.
type ByteArrayOutputStream struct {
	writerOutputStream
	buf *bytes.Buffer
}

func NewByteArrayOutputStream() *ByteArrayOutputStream {
	b := &bytes.Buffer{}
	return &ByteArrayOutputStream{writerOutputStream{w: b}, b}
}

func (s *ByteArrayOutputStream) ToByteArray() []int8 {
	out := make([]int8, s.buf.Len())
	for i, v := range s.buf.Bytes() {
		out[i] = int8(v)
	}
	return out
}

func (s *ByteArrayOutputStream) Size() int32 { return int32(s.buf.Len()) }

// ToString is toString() and toString(Charset): the bytes as UTF-8, the only charset used.
func (s *ByteArrayOutputStream) ToString(charset ...any) string { return s.buf.String() }
