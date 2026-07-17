package targets

import (
	"net/netip"
	"testing"

	"github.com/3011/cfscan/internal/model"
)

func TestOnePerPrefixReservesAddressesForSpecificPrefixes(t *testing.T) {
	prefixes := []model.Prefix{
		{CIDR: "104.16.0.0/24", IPVersion: 4, Active: true},
		{CIDR: "104.16.0.0/31", IPVersion: 4, Active: true},
		{CIDR: "104.16.0.0/32", IPVersion: 4, Active: true},
	}
	items, err := OnePerPrefix(prefixes, false, "one-per-prefix-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != len(prefixes) {
		t.Fatalf("got %d targets, want %d", len(items), len(prefixes))
	}
	seen := map[string]bool{}
	ordered := []netip.Prefix{
		netip.MustParsePrefix("104.16.0.0/32"),
		netip.MustParsePrefix("104.16.0.0/31"),
		netip.MustParsePrefix("104.16.0.0/24"),
	}
	for index, value := range items {
		if seen[value] {
			t.Fatalf("duplicate target %s", value)
		}
		seen[value] = true
		if !ordered[index].Contains(netip.MustParseAddr(value)) {
			t.Fatalf("target %s is outside expected prefix %s", value, ordered[index])
		}
	}
	if items[0] != "104.16.0.0" {
		t.Fatalf("most specific /32 was not reserved: %v", items)
	}
}

func TestOnePerPrefixHonorsIPv6Flag(t *testing.T) {
	prefixes := []model.Prefix{
		{CIDR: "104.16.0.0/24", IPVersion: 4, Active: true},
		{CIDR: "2606:4700::/48", IPVersion: 6, Active: true},
	}
	v4Only, err := OnePerPrefix(prefixes, false, "v4")
	if err != nil {
		t.Fatal(err)
	}
	if len(v4Only) != 1 || !netip.MustParseAddr(v4Only[0]).Is4() {
		t.Fatalf("unexpected IPv4-only targets: %v", v4Only)
	}
	all, err := OnePerPrefix(prefixes, true, "dual-stack")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d dual-stack targets", len(all))
	}
}
