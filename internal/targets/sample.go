package targets

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/netip"
	"sort"

	"github.com/3011/cfscan/v2/internal/model"
)

const MaxSampleTargets = 10000

func Sample(prefixes []model.Prefix, limit int, includeIPv6 bool, seed string) ([]model.ScanTarget, error) {
	if limit <= 0 || limit > MaxSampleTargets {
		return nil, fmt.Errorf("target limit must be between 1 and %d", MaxSampleTargets)
	}
	usable, err := usablePrefixes(prefixes, includeIPv6)
	if err != nil {
		return nil, err
	}
	sort.Slice(usable, func(i, j int) bool { return usable[i].String() < usable[j].String() })

	result := make([]model.ScanTarget, 0, limit)
	seen := make(map[string]struct{}, limit)
	for round := 0; len(result) < limit && round < limit*4; round++ {
		for _, prefix := range usable {
			if len(result) >= limit {
				break
			}
			addr, err := addressFor(prefix, seed, round)
			if err != nil {
				return nil, err
			}
			value := addr.String()
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, model.ScanTarget{TargetIP: value, PrefixCIDR: prefix.String()})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("could not sample target addresses")
	}
	return result, nil
}

// OnePerPrefix returns exactly one unique address for every active prefix.
// More-specific prefixes are processed first so constrained prefixes reserve
// their addresses before broader, overlapping prefixes choose alternatives.
func OnePerPrefix(prefixes []model.Prefix, includeIPv6 bool, seed string) ([]model.ScanTarget, error) {
	usable, err := usablePrefixes(prefixes, includeIPv6)
	if err != nil {
		return nil, err
	}
	sort.Slice(usable, func(i, j int) bool {
		if usable[i].Bits() != usable[j].Bits() {
			return usable[i].Bits() > usable[j].Bits()
		}
		return usable[i].String() < usable[j].String()
	})

	result := make([]model.ScanTarget, 0, len(usable))
	seenAddresses := make(map[string]struct{}, len(usable))
	for _, prefix := range usable {
		items, err := FromPrefix(prefix.String(), 1, seed, seenAddresses)
		if err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("could not choose a unique address for prefix %s", prefix)
		}
		result = append(result, items[0])
	}
	return result, nil
}

// FromPrefix chooses up to count deterministic, unique addresses from one prefix.
// The caller may pass a shared seen map to avoid collisions across overlapping prefixes.
func FromPrefix(cidr string, count int, seed string, seen map[string]struct{}) ([]model.ScanTarget, error) {
	if count <= 0 {
		return []model.ScanTarget{}, nil
	}
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("parse prefix %q: %w", cidr, err)
	}
	prefix = prefix.Masked()
	if seen == nil {
		seen = make(map[string]struct{}, count)
	}
	items := make([]model.ScanTarget, 0, count)
	for round := 0; len(items) < count && round < 4096; round++ {
		addr, err := addressFor(prefix, seed, round)
		if err != nil {
			return nil, err
		}
		value := addr.String()
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, model.ScanTarget{TargetIP: value, PrefixCIDR: prefix.String()})
	}
	return items, nil
}

func usablePrefixes(prefixes []model.Prefix, includeIPv6 bool) ([]netip.Prefix, error) {
	usable := make([]netip.Prefix, 0, len(prefixes))
	seenPrefixes := make(map[string]struct{}, len(prefixes))
	for _, item := range prefixes {
		if !item.Active || (!includeIPv6 && item.IPVersion == 6) {
			continue
		}
		prefix, err := netip.ParsePrefix(item.CIDR)
		if err != nil {
			return nil, fmt.Errorf("parse prefix %q: %w", item.CIDR, err)
		}
		prefix = prefix.Masked()
		key := prefix.String()
		if _, exists := seenPrefixes[key]; exists {
			continue
		}
		seenPrefixes[key] = struct{}{}
		usable = append(usable, prefix)
	}
	if len(usable) == 0 {
		return nil, fmt.Errorf("no active prefixes available")
	}
	return usable, nil
}

func addressFor(prefix netip.Prefix, seed string, round int) (netip.Addr, error) {
	bits := 128
	bytes := prefix.Addr().As16()
	if prefix.Addr().Is4() {
		bits = 32
		v4 := prefix.Addr().As4()
		bytes = [16]byte{}
		copy(bytes[12:], v4[:])
	}

	hostBits := bits - prefix.Bits()
	base := new(big.Int).SetBytes(bytes[:])
	if bits == 32 {
		base.SetBytes(bytes[12:])
	}
	count := new(big.Int).Lsh(big.NewInt(1), uint(hostBits))

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", prefix.String(), seed, round)))
	offset := new(big.Int).SetBytes(hash[:])
	offset.Mod(offset, count)
	if bits == 32 && hostBits > 1 {
		usable := new(big.Int).Sub(count, big.NewInt(2))
		offset.Mod(offset, usable)
		offset.Add(offset, big.NewInt(1))
	}
	value := new(big.Int).Add(base, offset)

	if bits == 32 {
		b := value.FillBytes(make([]byte, 4))
		return netip.AddrFrom4([4]byte{b[0], b[1], b[2], b[3]}), nil
	}
	b := value.FillBytes(make([]byte, 16))
	var out [16]byte
	copy(out[:], b)
	return netip.AddrFrom16(out), nil
}
