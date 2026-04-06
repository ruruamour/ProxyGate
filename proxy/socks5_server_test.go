package proxy

import (
	"bytes"
	"io"
	"testing"
)

func TestReadSOCKS5ConnectReplyConsumesVariableLengthReply(t *testing.T) {
	tests := []struct {
		name  string
		reply []byte
	}{
		{
			name:  "ipv4",
			reply: []byte{0x05, 0x00, 0x00, 0x01, 1, 2, 3, 4, 0x1F, 0x90},
		},
		{
			name:  "domain",
			reply: append([]byte{0x05, 0x00, 0x00, 0x03, 11}, append([]byte("example.com"), 0x1F, 0x90)...),
		},
		{
			name:  "ipv6",
			reply: append([]byte{0x05, 0x00, 0x00, 0x04}, append(bytes.Repeat([]byte{0x20}, 16), 0x1F, 0x90)...),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(append(tc.reply, []byte("tail")...))
			if err := readSOCKS5ConnectReply(buf); err != nil {
				t.Fatalf("readSOCKS5ConnectReply() error = %v", err)
			}

			rest, err := io.ReadAll(buf)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if string(rest) != "tail" {
				t.Fatalf("remaining bytes = %q, want %q", string(rest), "tail")
			}
		})
	}
}

func TestReadSOCKS5ConnectReplyRejectsFailureReply(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x05, 0x05, 0x00, 0x01, 1, 2, 3, 4, 0, 80})
	if err := readSOCKS5ConnectReply(buf); err == nil {
		t.Fatal("readSOCKS5ConnectReply() error = nil, want failure")
	}
}
