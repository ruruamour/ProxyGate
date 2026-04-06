package proxy

import (
	"io"
	"net"
	"time"
)

const tunnelSuccessGrace = 2 * time.Second

type tunnelCopyResult struct {
	direction string
	bytes     int64
}

type tunnelOutcome struct {
	duration      time.Duration
	clientBytes   int64
	upstreamBytes int64
}

type closeReader interface {
	CloseRead() error
}

type closeWriter interface {
	CloseWrite() error
}

func closeConnRead(conn net.Conn) {
	if conn == nil {
		return
	}
	if closer, ok := conn.(closeReader); ok {
		_ = closer.CloseRead()
		return
	}
	_ = conn.Close()
}

func closeConnWrite(conn net.Conn) {
	if conn == nil {
		return
	}
	if closer, ok := conn.(closeWriter); ok {
		_ = closer.CloseWrite()
		return
	}
	_ = conn.Close()
}

func relayTunnel(clientConn, upstreamConn net.Conn) tunnelOutcome {
	started := time.Now()
	results := make(chan tunnelCopyResult, 2)

	copyFn := func(direction string, dst, src net.Conn) {
		n, _ := io.Copy(dst, src)
		closeConnWrite(dst)
		closeConnRead(src)
		results <- tunnelCopyResult{direction: direction, bytes: n}
	}

	go copyFn("client_to_upstream", upstreamConn, clientConn)
	go copyFn("upstream_to_client", clientConn, upstreamConn)

	first := <-results
	second := <-results
	_ = clientConn.Close()
	_ = upstreamConn.Close()

	outcome := tunnelOutcome{duration: time.Since(started)}
	for _, result := range []tunnelCopyResult{first, second} {
		if result.direction == "client_to_upstream" {
			outcome.clientBytes += result.bytes
		}
		if result.direction == "upstream_to_client" {
			outcome.upstreamBytes += result.bytes
		}
	}
	return outcome
}

func tunnelLooksHealthy(outcome tunnelOutcome) bool {
	if outcome.upstreamBytes > 0 {
		return true
	}
	return outcome.duration >= tunnelSuccessGrace
}
