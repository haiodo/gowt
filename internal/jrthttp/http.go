package jrthttp

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
)

// The com.sun.net.httpserver subset EchoHttpServer (tests/swttests) uses, over net/http.

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
func (e *HttpExchange) GetRequestURI() *jrt.URI          { return jrt.NewURI(e.r.URL.RequestURI()) }
func (e *HttpExchange) GetResponseHeaders() *HttpHeaders { return &HttpHeaders{e.w.Header()} }
func (e *HttpExchange) GetRequestBody() jrt.InputStream  { return jrt.NewInputStream(e.r.Body) }

func (e *HttpExchange) SendResponseHeaders(code int32, length int64) {
	if length >= 0 {
		e.w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	}
	e.w.WriteHeader(int(code))
	e.sent = true
}

type responseBody struct{ e *HttpExchange }

func (b responseBody) Write(p []byte) (int, error) { return b.e.w.Write(p) }
func (b responseBody) Close() error                { b.e.Close(); return nil }
func (e *HttpExchange) GetResponseBody() jrt.OutputStream {
	return jrt.NewOutputStream(responseBody{e})
}

// Close ends the exchange: the handler's return does that anyway. Closing before any header was sent
// drops the connection, as the JDK server does; a writer that cannot hijack gets an empty 200 instead.
func (e *HttpExchange) Close() {
	if e.sent {
		return
	}
	e.sent = true
	if conn, _, err := http.NewResponseController(e.w).Hijack(); err == nil {
		_ = conn.Close()
		return
	}
	e.w.WriteHeader(http.StatusOK)
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
		panic(jrt.NewIOException())
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
