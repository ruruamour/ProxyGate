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

func TestBuildOutboundSupportsAuthenticatedDirectProxies(t *testing.T) {
	httpNode := ParsedNode{
		Type:   "http",
		Server: "proxy.example.com",
		Port:   8443,
		Raw: map[string]interface{}{
			"username": "alice",
			"password": "wonderland",
			"tls":      true,
		},
	}
	httpOutbound := buildOutbound(httpNode, "http-node")
	if httpOutbound == nil {
		t.Fatal("http outbound = nil")
	}
	if got := httpOutbound["type"]; got != "http" {
		t.Fatalf("http outbound type = %v, want http", got)
	}
	if got := httpOutbound["username"]; got != "alice" {
		t.Fatalf("http outbound username = %v, want alice", got)
	}
	if got := httpOutbound["password"]; got != "wonderland" {
		t.Fatalf("http outbound password = %v, want wonderland", got)
	}
	if _, ok := httpOutbound["tls"]; !ok {
		t.Fatal("http outbound tls missing")
	}

	socksNode := ParsedNode{
		Type:   "socks5",
		Server: "socks.example.com",
		Port:   1080,
		Raw: map[string]interface{}{
			"username": "bob",
			"password": "secret",
		},
	}
	socksOutbound := buildOutbound(socksNode, "socks-node")
	if socksOutbound == nil {
		t.Fatal("socks outbound = nil")
	}
	if got := socksOutbound["type"]; got != "socks" {
		t.Fatalf("socks outbound type = %v, want socks", got)
	}
	if got := socksOutbound["username"]; got != "bob" {
		t.Fatalf("socks outbound username = %v, want bob", got)
	}
	if got := socksOutbound["password"]; got != "secret" {
		t.Fatalf("socks outbound password = %v, want secret", got)
	}
}

func TestFallbackNodesForCheckErrorSkipsQUICNodes(t *testing.T) {
	nodes := []ParsedNode{
		{Name: "vless-node", Type: "vless"},
		{Name: "hy2-node", Type: "hysteria2"},
		{Name: "tuic-node", Type: "tuic"},
		{Name: "trojan-node", Type: "trojan"},
	}

	filtered, skipped, retried := fallbackNodesForCheckError(nodes, &singBoxConfigCheckError{
		Output: "FATAL initialize outbound[1]: QUIC is not included in this build, rebuild with -tags with_quic",
	})
	if !retried {
		t.Fatal("expected QUIC fallback to trigger")
	}
	if len(filtered) != 2 {
		t.Fatalf("filtered node count = %d, want 2", len(filtered))
	}
	if filtered[0].Type != "vless" || filtered[1].Type != "trojan" {
		t.Fatalf("unexpected filtered nodes: %+v", filtered)
	}
	if len(skipped) != 2 {
		t.Fatalf("skipped node count = %d, want 2", len(skipped))
	}
	if skipped[0].Type != "hysteria2" || skipped[1].Type != "tuic" {
		t.Fatalf("unexpected skipped nodes: %+v", skipped)
	}
}

func TestFallbackNodesForCheckErrorKeepsNodesForOtherFailures(t *testing.T) {
	nodes := []ParsedNode{
		{Name: "vmess-node", Type: "vmess"},
		{Name: "tuic-node", Type: "tuic"},
	}

	filtered, skipped, retried := fallbackNodesForCheckError(nodes, &singBoxConfigCheckError{
		Output: "FATAL parse config: invalid tls setting",
	})
	if retried {
		t.Fatal("unexpected fallback for non-QUIC failure")
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped node count = %d, want 0", len(skipped))
	}
	if len(filtered) != len(nodes) {
		t.Fatalf("filtered node count = %d, want %d", len(filtered), len(nodes))
	}
}
