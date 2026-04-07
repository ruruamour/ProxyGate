package main

import (
	"testing"

	"proxygate/pool"
	"proxygate/storage"
)

func TestCandidateBudget(t *testing.T) {
	if got := candidateBudget("refill", 1); got != refillCandidateMin {
		t.Fatalf("candidateBudget(refill, 1) = %d, want %d", got, refillCandidateMin)
	}
	if got := candidateBudget("emergency", 1); got != emergencyCandidateMin {
		t.Fatalf("candidateBudget(emergency, 1) = %d, want %d", got, emergencyCandidateMin)
	}
	if got := candidateBudget("refill", 100); got != refillCandidateMax {
		t.Fatalf("candidateBudget(refill, 100) = %d, want %d", got, refillCandidateMax)
	}
	if got := candidateBudget("emergency", 100); got != emergencyCandidateMax {
		t.Fatalf("candidateBudget(emergency, 100) = %d, want %d", got, emergencyCandidateMax)
	}
}

func TestLimitCandidatesForProtocol(t *testing.T) {
	status := &pool.PoolStatus{
		HTTP:        8,
		SOCKS5:      70,
		HTTPSlots:   30,
		SOCKS5Slots: 70,
	}

	candidates := make([]storage.Proxy, 3000)
	for i := range candidates {
		candidates[i] = storage.Proxy{Address: "127.0.0.1:8080", Protocol: "http"}
	}

	limited := limitCandidatesForProtocol("refill", status, "http", candidates)
	if len(limited) != 1408 {
		t.Fatalf("len(limitCandidatesForProtocol()) = %d, want %d", len(limited), 1408)
	}

	if got := limitCandidatesForProtocol("refill", status, "socks5", candidates); got != nil {
		t.Fatalf("limitCandidatesForProtocol() for non-short protocol = %v, want nil", got)
	}
}
