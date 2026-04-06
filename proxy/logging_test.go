package proxy

import (
	"sync/atomic"
	"testing"
)

func TestSampledSuccessLog(t *testing.T) {
	var counter atomic.Uint64

	for i := 1; i < successLogSampleEvery; i++ {
		if seq, ok := sampledSuccessLog(&counter); ok {
			t.Fatalf("sampledSuccessLog() at seq=%d unexpectedly sampled", seq)
		}
	}

	seq, ok := sampledSuccessLog(&counter)
	if !ok {
		t.Fatalf("sampledSuccessLog() at seq=%d = false, want true", seq)
	}
	if seq != successLogSampleEvery {
		t.Fatalf("sampledSuccessLog() seq = %d, want %d", seq, successLogSampleEvery)
	}

	for i := 1; i < successLogSampleEvery; i++ {
		if seq, ok := sampledSuccessLog(&counter); ok {
			t.Fatalf("second window sampled early at seq=%d", seq)
		}
	}

	seq, ok = sampledSuccessLog(&counter)
	if !ok {
		t.Fatalf("sampledSuccessLog() at second boundary seq=%d = false, want true", seq)
	}
	if seq != successLogSampleEvery*2 {
		t.Fatalf("sampledSuccessLog() seq = %d, want %d", seq, successLogSampleEvery*2)
	}
}
