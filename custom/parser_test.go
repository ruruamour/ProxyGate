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
