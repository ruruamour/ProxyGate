package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

func dialSOCKS5Connect(proxyAddress, target string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	proxyConn, err := dialer.Dial("tcp", proxyAddress)
	if err != nil {
		return nil, err
	}
	if err := proxyConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}

	if _, err := proxyConn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}

	handshake := make([]byte, 2)
	if _, err := io.ReadFull(proxyConn, handshake); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}
	if handshake[0] != 0x05 || handshake[1] != 0x00 {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("socks5 handshake failed")
	}

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		_ = proxyConn.Close()
		return nil, err
	}

	req := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append(req, 0x01)
			req = append(req, ip4...)
		} else {
			req = append(req, 0x04)
			req = append(req, ip...)
		}
	} else {
		req = append(req, 0x03)
		req = append(req, byte(len(host)))
		req = append(req, host...)
	}

	var portNum uint16
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, portNum)
	req = append(req, portBytes...)

	if _, err := proxyConn.Write(req); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}
	if err := readSOCKS5ConnectReply(proxyConn); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}
	if err := proxyConn.SetDeadline(time.Time{}); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}

	return proxyConn, nil
}
