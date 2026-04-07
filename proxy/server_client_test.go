package proxy

import (
	"testing"

	"proxygate/config"
	"proxygate/storage"
)

func TestBuildClientCacheEviction(t *testing.T) {
	srv := &Server{
		cfg: &config.Config{ValidateTimeout: 10},
	}
	proxyAddr := &storage.Proxy{Address: "127.0.0.1:8080", Protocol: "http"}

	first, err := srv.buildClient(proxyAddr)
	if err != nil {
		t.Fatalf("buildClient() error = %v", err)
	}
	second, err := srv.buildClient(proxyAddr)
	if err != nil {
		t.Fatalf("buildClient() second call error = %v", err)
	}
	if first != second {
		t.Fatal("buildClient() did not reuse cached client")
	}

	srv.evictClient(proxyAddr)

	third, err := srv.buildClient(proxyAddr)
	if err != nil {
		t.Fatalf("buildClient() after eviction error = %v", err)
	}
	if third == first {
		t.Fatal("buildClient() reused evicted client")
	}
}
