package custom

import (
	"path/filepath"
	"testing"
)

func TestGenerateConfigKeepsStablePorts(t *testing.T) {
	process := &SingBoxProcess{
		configFile: filepath.Join(t.TempDir(), "config.json"),
		basePort:   20000,
		portMap:    make(map[string]int),
	}

	nodeA := ParsedNode{
		Type:   "vmess",
		Server: "a.example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"uuid": "uuid-a",
		},
	}
	nodeB := ParsedNode{
		Type:   "vmess",
		Server: "b.example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"uuid": "uuid-b",
		},
	}
	nodeC := ParsedNode{
		Type:   "vmess",
		Server: "c.example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"uuid": "uuid-c",
		},
	}

	if err := process.generateConfig([]ParsedNode{nodeA, nodeB}); err != nil {
		t.Fatalf("generateConfig first: %v", err)
	}
	portA := process.portMap[nodeA.NodeKey()]
	portB := process.portMap[nodeB.NodeKey()]
	if portA == 0 || portB == 0 || portA == portB {
		t.Fatalf("unexpected initial ports: A=%d B=%d", portA, portB)
	}

	if err := process.generateConfig([]ParsedNode{nodeB, nodeA, nodeC}); err != nil {
		t.Fatalf("generateConfig second: %v", err)
	}

	if got := process.portMap[nodeA.NodeKey()]; got != portA {
		t.Fatalf("nodeA port changed: got %d want %d", got, portA)
	}
	if got := process.portMap[nodeB.NodeKey()]; got != portB {
		t.Fatalf("nodeB port changed: got %d want %d", got, portB)
	}

	portC := process.portMap[nodeC.NodeKey()]
	if portC == 0 || portC == portA || portC == portB {
		t.Fatalf("unexpected nodeC port: %d", portC)
	}
}

func TestBuildOutboundDoesNotMutateNodeKeyForAnytls(t *testing.T) {
	node := ParsedNode{
		Type:   "anytls",
		Server: "tls.example.com",
		Port:   443,
		Raw: map[string]interface{}{
			"password": "secret",
		},
	}

	before := node.NodeKey()
	outbound := buildOutbound(node, "node-0")
	if outbound == nil {
		t.Fatalf("buildOutbound returned nil")
	}
	if after := node.NodeKey(); after != before {
		t.Fatalf("node key mutated after buildOutbound: before=%s after=%s", before, after)
	}
}
