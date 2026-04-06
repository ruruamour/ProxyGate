package validator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

func newProbeTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     false,
		MaxConnsPerHost:       1,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: time.Second,
	}
}

func newHTTPClient(address string, timeout time.Duration) (*http.Client, func(), error) {
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", address))
	if err != nil {
		return nil, nil, err
	}

	transport := newProbeTransport(timeout)
	transport.Proxy = http.ProxyURL(proxyURL)

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, transport.CloseIdleConnections, nil
}

func newSOCKS5Client(address string, timeout time.Duration) (*http.Client, func(), error) {
	dialer, err := proxy.SOCKS5("tcp", address, nil, proxy.Direct)
	if err != nil {
		return nil, nil, err
	}

	transport := newProbeTransport(timeout)
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, transport.CloseIdleConnections, nil
}
