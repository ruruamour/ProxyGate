package proxy

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestTunnelLooksHealthy(t *testing.T) {
	t.Run("healthy when upstream returns data", func(t *testing.T) {
		if !tunnelLooksHealthy(tunnelOutcome{duration: 200 * time.Millisecond, upstreamBytes: 1}) {
			t.Fatal("expected tunnel with upstream bytes to be healthy")
		}
	})

	t.Run("healthy when tunnel stays open long enough", func(t *testing.T) {
		if !tunnelLooksHealthy(tunnelOutcome{duration: 3 * time.Second}) {
			t.Fatal("expected long-lived tunnel to be healthy")
		}
	})

	t.Run("not healthy on early close without upstream data", func(t *testing.T) {
		if tunnelLooksHealthy(tunnelOutcome{duration: 500 * time.Millisecond}) {
			t.Fatal("expected short tunnel without upstream data to avoid success accounting")
		}
	})
}

func TestRelayTunnelPreservesHalfClose(t *testing.T) {
	downstreamClient, downstreamRelay := tcpPair(t)
	upstreamPeer, upstreamRelay := tcpPair(t)

	outcomeCh := make(chan tunnelOutcome, 1)
	go func() {
		outcomeCh <- relayTunnel(downstreamRelay, upstreamRelay)
	}()

	if _, err := downstreamClient.Write([]byte("ping")); err != nil {
		t.Fatalf("write downstream request: %v", err)
	}
	if err := downstreamClient.CloseWrite(); err != nil {
		t.Fatalf("close downstream write: %v", err)
	}

	request, err := io.ReadAll(upstreamPeer)
	if err != nil {
		t.Fatalf("read upstream request: %v", err)
	}
	if string(request) != "ping" {
		t.Fatalf("upstream request = %q, want %q", string(request), "ping")
	}

	if _, err := upstreamPeer.Write([]byte("pong")); err != nil {
		t.Fatalf("write upstream response: %v", err)
	}
	if err := upstreamPeer.CloseWrite(); err != nil {
		t.Fatalf("close upstream write: %v", err)
	}

	response, err := io.ReadAll(downstreamClient)
	if err != nil {
		t.Fatalf("read downstream response: %v", err)
	}
	if string(response) != "pong" {
		t.Fatalf("downstream response = %q, want %q", string(response), "pong")
	}

	outcome := <-outcomeCh
	if outcome.clientBytes != 4 {
		t.Fatalf("clientBytes = %d, want 4", outcome.clientBytes)
	}
	if outcome.upstreamBytes != 4 {
		t.Fatalf("upstreamBytes = %d, want 4", outcome.upstreamBytes)
	}
}

func tcpPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	accepted := make(chan *net.TCPConn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			errCh <- acceptErr
			return
		}
		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			_ = conn.Close()
			errCh <- io.ErrUnexpectedEOF
			return
		}
		accepted <- tcpConn
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	clientTCP, ok := clientConn.(*net.TCPConn)
	if !ok {
		_ = clientConn.Close()
		t.Fatal("client connection is not *net.TCPConn")
	}

	select {
	case acceptErr := <-errCh:
		_ = clientTCP.Close()
		t.Fatalf("accept: %v", acceptErr)
	case serverTCP := <-accepted:
		t.Cleanup(func() { _ = clientTCP.Close() })
		t.Cleanup(func() { _ = serverTCP.Close() })
		return clientTCP, serverTCP
	}

	return nil, nil
}
