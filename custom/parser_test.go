package custom

import "testing"

func TestNodeKeyStableAndUnique(t *testing.T) {
	nodeA := ParsedNode{
		Type:   "vmess",
		Server: "example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"uuid":    "uuid-a",
			"network": "ws",
		},
	}
	nodeASame := ParsedNode{
		Type:   "vmess",
		Server: "example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"network": "ws",
			"uuid":    "uuid-a",
		},
	}
	nodeB := ParsedNode{
		Type:   "vmess",
		Server: "example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"uuid":    "uuid-b",
			"network": "ws",
		},
	}

	if nodeA.NodeKey() != nodeASame.NodeKey() {
		t.Fatalf("same config should keep same node key")
	}
	if nodeA.NodeKey() == nodeB.NodeKey() {
		t.Fatalf("different credentials should not share node key")
	}
}

func TestParsePlainSupportsAuthenticatedDirectProxyURLs(t *testing.T) {
	nodes, err := parsePlain([]byte("socks5://demo:secret@example.com:1080\nhttps://user:pass@proxy.example.com:8443"))
	if err != nil {
		t.Fatalf("parsePlain() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(nodes))
	}

	if nodes[0].Type != "socks5" || nodes[0].Server != "example.com" || nodes[0].Port != 1080 {
		t.Fatalf("unexpected first node = %+v", nodes[0])
	}
	if got := rawString(nodes[0].Raw, "username"); got != "demo" {
		t.Fatalf("first node username = %q, want demo", got)
	}
	if got := rawString(nodes[0].Raw, "password"); got != "secret" {
		t.Fatalf("first node password = %q, want secret", got)
	}
	if nodes[0].IsDirect() {
		t.Fatal("authenticated socks5 node should go through sing-box, got direct")
	}

	if nodes[1].Type != "http" || nodes[1].Server != "proxy.example.com" || nodes[1].Port != 8443 {
		t.Fatalf("unexpected second node = %+v", nodes[1])
	}
	if got := rawString(nodes[1].Raw, "username"); got != "user" {
		t.Fatalf("second node username = %q, want user", got)
	}
	if got := rawString(nodes[1].Raw, "password"); got != "pass" {
		t.Fatalf("second node password = %q, want pass", got)
	}
	if !nodes[1].Raw["tls"].(bool) {
		t.Fatal("https proxy URL should keep tls=true in raw config")
	}
	if nodes[1].IsDirect() {
		t.Fatal("authenticated https proxy node should go through sing-box, got direct")
	}
}
