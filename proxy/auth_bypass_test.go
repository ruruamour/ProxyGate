package proxy

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"proxygate/config"
	"proxygate/storage"
)

type remoteAddrConn struct {
	net.Conn
	remote net.Addr
}

func (c *remoteAddrConn) RemoteAddr() net.Addr {
	return c.remote
}

func TestIsLoopbackRemoteAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{name: "ipv4 loopback", addr: "127.0.0.1:1234", want: true},
		{name: "ipv6 loopback", addr: "[::1]:1234", want: true},
		{name: "localhost", addr: "localhost:1234", want: true},
		{name: "public", addr: "198.51.100.10:1234", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLoopbackRemoteAddr(tc.addr); got != tc.want {
				t.Fatalf("isLoopbackRemoteAddr(%q) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

func TestServeHTTPAllowsLoopbackWithoutAuth(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	srv := &Server{
		storage:  store,
		cfg:      &config.Config{ProxyAuthEnabled: true, LocalAuthBypass: true},
		sessions: NewSessionManager(),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code == http.StatusProxyAuthRequired {
		t.Fatalf("ServeHTTP() returned %d, want non-auth failure for loopback client", rec.Code)
	}
}

func TestServeHTTPRejectsRemoteWithoutAuth(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	defer store.Close()

	srv := &Server{
		storage:  store,
		cfg:      &config.Config{ProxyAuthEnabled: true, LocalAuthBypass: true},
		sessions: NewSessionManager(),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "198.51.100.10:34567"
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestSOCKS5HandshakeAllowsLoopbackNoAuth(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	srv := &SOCKS5Server{
		cfg: &config.Config{
			ProxyAuthEnabled: true,
			LocalAuthBypass:  true,
		},
	}

	done := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		_, err := srv.socks5Handshake(&remoteAddrConn{
			Conn:   serverConn,
			remote: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 23456},
		})
		done <- err
	}()

	if _, err := clientConn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	reply := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, reply); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("socks5Handshake() error = %v", err)
	}
	if string(reply) != string([]byte{0x05, 0x00}) {
		t.Fatalf("reply = %v, want [5 0]", reply)
	}
}

func TestSOCKS5HandshakeRejectsRemoteNoAuthWhenAuthRequired(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	srv := &SOCKS5Server{
		cfg: &config.Config{
			ProxyAuthEnabled: true,
			LocalAuthBypass:  true,
		},
	}

	done := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		_, err := srv.socks5Handshake(&remoteAddrConn{
			Conn:   serverConn,
			remote: &net.TCPAddr{IP: net.ParseIP("198.51.100.10"), Port: 23456},
		})
		done <- err
	}()

	if _, err := clientConn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	reply := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, reply); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}

	if err := <-done; err == nil {
		t.Fatal("socks5Handshake() error = nil, want auth failure")
	}
	if string(reply) != string([]byte{0x05, 0xFF}) {
		t.Fatalf("reply = %v, want [5 255]", reply)
	}
}
