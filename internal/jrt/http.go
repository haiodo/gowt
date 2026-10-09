package jrt

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// The java.net and com.sun.net.httpserver subset EchoHttpServer (tests/swttests) uses, over net/http.

type InetAddress struct{ host string }

func InetAddressGetLoopbackAddress() *InetAddress { return &InetAddress{"127.0.0.1"} }

type InetSocketAddress struct {
	addr *InetAddress
	port int32
}

func NewInetSocketAddress(addr *InetAddress, port int32) *InetSocketAddress {
	return &InetSocketAddress{addr, port}
}

func (a *InetSocketAddress) GetPort() int32 { return a.port }

// Charset stands for java.nio.charset.Charset: everything is UTF-8 here.
type Charset struct{}

var StandardCharsetsUTF_8 = &Charset{}

func URLEncoderEncode(s string, _ *Charset) string { return url.QueryEscape(s) }

func URLDecoderDecode(s string, _ *Charset) string {
	d, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return d
}

// GetQuery is URI.getQuery(): the raw-decoded query, null (the empty string here) without one.
func (u *URI) GetQuery() string {
	p, err := url.Parse(u.s)
	if err != nil {
		return ""
	}
	q, err := url.PathUnescape(p.RawQuery)
	if err != nil {
		return p.RawQuery
	}
	return q
}

type HttpHeaders struct{ h http.Header }

func (h *HttpHeaders) Set(k, v string) { h.h.Set(k, v) }

// HttpExchange is one request being answered; SendResponseHeaders(code, length) with length -1 means no body.
type HttpExchange struct {
	w    http.ResponseWriter
	r    *http.Request
	sent bool
	done chan struct{}
}

func (e *HttpExchange) GetRequestMethod() string         { return e.r.Method }
func (e *HttpExchange) GetRequestURI() *URI              { return &URI{e.r.URL.RequestURI()} }
func (e *HttpExchange) GetResponseHeaders() *HttpHeaders { return &HttpHeaders{e.w.Header()} }
func (e *HttpExchange) GetRequestBody() InputStream      { return NewInputStream(e.r.Body) }

func (e *HttpExchange) SendResponseHeaders(code int32, length int64) {
	if length >= 0 {
		e.w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	}
	e.w.WriteHeader(int(code))
	e.sent = true
}

type responseBody struct{ e *HttpExchange }

func (b responseBody) Write(x int32)                     { _, _ = b.e.w.Write([]byte{byte(x)}) }
func (b responseBody) WriteRange(p []int8, off, n int32) { _, _ = b.e.w.Write(toBytes(p[off : off+n])) }
func (b responseBody) Flush()                            {}
func (b responseBody) Close()                            { b.e.Close() }
func (e *HttpExchange) GetResponseBody() OutputStream    { return responseBody{e} }

// Close ends the exchange: the handler's return does that anyway, but an early close sends the headers.
func (e *HttpExchange) Close() {
	if !e.sent {
		e.w.WriteHeader(http.StatusOK)
		e.sent = true
	}
}

// HttpServer is com.sun.net.httpserver.HttpServer on a loopback listener.
type HttpServer struct {
	mu   sync.Mutex
	ln   net.Listener
	srv  *http.Server
	mux  *http.ServeMux
	port int32
}

func HttpServerCreate(addr *InetSocketAddress, _ int32) *HttpServer {
	ln, err := net.Listen("tcp", net.JoinHostPort(addr.addr.host, strconv.Itoa(int(addr.port))))
	if err != nil {
		panic(NewIOException())
	}
	mux := http.NewServeMux()
	return &HttpServer{ln: ln, mux: mux, srv: &http.Server{Handler: mux}, port: int32(ln.Addr().(*net.TCPAddr).Port)}
}

func (s *HttpServer) CreateContext(path string, h func(*HttpExchange)) {
	s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// A handler that panics (a Java exception) ends its connection, as the JDK server does.
			if recover() != nil {
				panic(http.ErrAbortHandler)
			}
		}()
		e := &HttpExchange{w: w, r: r}
		h(e)
		e.Close()
	})
}

func (s *HttpServer) Start() { go func() { _ = s.srv.Serve(s.ln) }() }

func (s *HttpServer) Stop(delay int32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(delay)*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	_ = s.srv.Close()
}

func (s *HttpServer) GetAddress() *InetSocketAddress {
	return &InetSocketAddress{&InetAddress{"127.0.0.1"}, s.port}
}

// GetBytes is String.getBytes(): the UTF-8 bytes as Java's signed bytes.
func GetBytes(s string) []int8 {
	b := []byte(s)
	out := make([]int8, len(b))
	for i, v := range b {
		out[i] = int8(v)
	}
	return out
}

// WriteBytes is OutputStream.write(byte[]).
func WriteBytes(o OutputStream, b []int8) { o.WriteRange(b, 0, int32(len(b))) }

// StringFromBytes is new String(byte[], UTF_8).
func StringFromBytes(b []int8) string { return string(toBytes(b)) }
