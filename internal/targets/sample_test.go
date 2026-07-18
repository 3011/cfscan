package targets

import (
	"net/netip"
	"testing"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestSampleDistributesAndStaysInsidePrefixes(t *testing.T) {
	prefixes := []model.Prefix{{CIDR: "104.16.0.0/24", IPVersion: 4, Active: true}, {CIDR: "172.64.0.0/24", IPVersion: 4, Active: true}}
	items, err := Sample(prefixes, 20, false, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 20 {
		t.Fatalf("got %d targets", len(items))
	}
	seen := map[string]bool{}
	parsed := []netip.Prefix{netip.MustParsePrefix(prefixes[0].CIDR), netip.MustParsePrefix(prefixes[1].CIDR)}
	for _, value := range items {
		if seen[value] {
			t.Fatalf("duplicate target %s", value)
		}
		seen[value] = true
		addr := netip.MustParseAddr(value)
		if !parsed[0].Contains(addr) && !parsed[1].Contains(addr) {
			t.Fatalf("target outside prefixes: %s", value)
		}
	}
}

func TestSampleRejectsExcessiveLimit(t *testing.T) {
	prefixes := []model.Prefix{{CIDR: "192.0.2.0/24", IPVersion: 4, Active: true}}
	if _, err := Sample(prefixes, MaxSampleTargets+1, false, "test"); err == nil {
		t.Fatal("expected excessive target limit to be rejected")
	}
}
