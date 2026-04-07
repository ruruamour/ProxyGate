package pool

import (
	"testing"

	"proxygate/config"
)

func TestNeedsFetchSkipsFreeFetchInCustomOnlyMode(t *testing.T) {
	mgr := &Manager{
		cfg: &config.Config{CustomProxyMode: "custom_only"},
	}

	need, mode, protocol := mgr.NeedsFetch(&PoolStatus{
		HTTP:        0,
		SOCKS5:      0,
		HTTPSlots:   30,
		SOCKS5Slots: 70,
		State:       "emergency",
	})
	if need {
		t.Fatalf("NeedsFetch() = true, want false in custom_only mode (mode=%q protocol=%q)", mode, protocol)
	}
}
