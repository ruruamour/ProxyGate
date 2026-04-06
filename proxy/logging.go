package proxy

import "sync/atomic"

const successLogSampleEvery = 128

var (
	httpSuccessLogSeq      atomic.Uint64
	tunnelEstablishedSeq   atomic.Uint64
	socks5EstablishedSeq   atomic.Uint64
	successLogSampleModulo = uint64(successLogSampleEvery)
)

func sampledSuccessLog(counter *atomic.Uint64) (uint64, bool) {
	seq := counter.Add(1)
	if successLogSampleModulo == 0 {
		return seq, false
	}
	return seq, seq%successLogSampleModulo == 0
}
