package league

import (
	"crypto/sha256"
	"encoding/binary"
	"net/netip"
	"sort"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
	"github.com/3011/cfscan/v2/internal/targets"
)

const (
	TierObservation = "observation"
	TierChallenger  = "challenger"
	TierChampion    = "champion"
)

type Thresholds struct {
	MaxLatencyMS  float64
	MaxPacketLoss float64
}

type PlanOptions struct {
	Now    time.Time
	Seed   string
	Budget int
	Force  bool
}

func Evaluate(entry model.PrefixLeagueEntry, thresholds Thresholds) model.PrefixLeagueEntry {
	if entry.Tier == "" {
		entry.Tier = TierObservation
	}
	if entry.LastResultAt == nil || (entry.LastEvaluatedAt != nil && !entry.LastResultAt.After(*entry.LastEvaluatedAt)) {
		return entry
	}

	good := entry.SampleCount >= 3 && entry.DistinctIPCount >= 3 &&
		entry.AvailabilityRate >= 90 && positiveWithin(entry.LatencyP95MS, thresholds.MaxLatencyMS) &&
		entry.PacketLossAvg <= thresholds.MaxPacketLoss
	championReady := entry.SampleCount >= 8 && entry.DistinctIPCount >= 6 &&
		entry.AvailabilityRate >= 95 && positiveWithin(entry.LatencyP95MS, thresholds.MaxLatencyMS) &&
		entry.PacketLossAvg <= thresholds.MaxPacketLoss

	lossBoundary := thresholds.MaxPacketLoss*1.5 + 5
	if lossBoundary > 100 {
		lossBoundary = 100
	}
	bad := entry.RecentSampleCount >= 3 && (entry.RecentAvailabilityRate < 80 ||
		(entry.RecentLatencyP95MS > 0 && entry.RecentLatencyP95MS > thresholds.MaxLatencyMS*1.5) ||
		entry.RecentPacketLossAvg > lossBoundary)
	severe := entry.RecentSampleCount >= 3 && entry.RecentAvailabilityRate == 0

	if bad {
		entry.BadStreak++
	} else {
		entry.BadStreak = 0
	}

	switch entry.Tier {
	case TierChampion:
		switch {
		case severe:
			entry.Tier = TierObservation
			entry.BadStreak = 0
		case entry.BadStreak >= 2:
			entry.Tier = TierChallenger
			entry.BadStreak = 0
		}
	case TierChallenger:
		switch {
		case severe:
			entry.Tier = TierObservation
			entry.BadStreak = 0
		case championReady:
			entry.Tier = TierChampion
			entry.BadStreak = 0
		case entry.BadStreak >= 2:
			entry.Tier = TierObservation
			entry.BadStreak = 0
		}
	default:
		entry.Tier = TierObservation
		if good {
			entry.Tier = TierChallenger
			entry.BadStreak = 0
		}
	}

	evaluatedAt := *entry.LastResultAt
	entry.LastEvaluatedAt = &evaluatedAt
	return entry
}

func Due(entry model.PrefixLeagueEntry, now time.Time, force bool) bool {
	if !entry.Active {
		return false
	}
	if force || entry.LastScheduledAt == nil {
		return true
	}
	return !entry.LastScheduledAt.Add(intervalForTier(entry.Tier)).After(now)
}

func PlanAgent(entries []model.PrefixLeagueEntry, candidates map[string][]model.LeagueCandidate, options PlanOptions) []model.ScanTarget {
	if options.Budget <= 0 || options.Budget > targets.MaxSampleTargets {
		options.Budget = targets.MaxSampleTargets
	}
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}

	due := make([]model.PrefixLeagueEntry, 0, len(entries))
	for _, entry := range entries {
		if Due(entry, options.Now, options.Force) {
			due = append(due, entry)
		}
	}
	sort.Slice(due, func(i, j int) bool {
		left, leftErr := netip.ParsePrefix(due[i].PrefixCIDR)
		right, rightErr := netip.ParsePrefix(due[j].PrefixCIDR)
		if leftErr == nil && rightErr == nil && left.Bits() != right.Bits() {
			return left.Bits() > right.Bits()
		}
		return stableScore(options.Seed, due[i].AgentID, due[i].PrefixCIDR) < stableScore(options.Seed, due[j].AgentID, due[j].PrefixCIDR)
	})

	type prefixQueue struct {
		prefix string
		items  []model.ScanTarget
	}
	queues := map[string][]prefixQueue{
		TierChampion:    {},
		TierChallenger:  {},
		TierObservation: {},
	}
	seen := make(map[string]struct{}, options.Budget)
	for _, entry := range due {
		tier := normalizedTier(entry.Tier)
		desired := samplesForTier(tier)
		items := make([]model.ScanTarget, 0, desired)
		prefix, err := netip.ParsePrefix(entry.PrefixCIDR)
		if err != nil {
			continue
		}
		prefix = prefix.Masked()
		anchorLimit := anchorsForTier(tier)
		for _, candidate := range candidates[entry.PrefixCIDR] {
			if len(items) >= anchorLimit {
				break
			}
			addr, err := netip.ParseAddr(candidate.TargetIP)
			if err != nil || !prefix.Contains(addr) {
				continue
			}
			if _, exists := seen[candidate.TargetIP]; exists {
				continue
			}
			seen[candidate.TargetIP] = struct{}{}
			items = append(items, model.ScanTarget{TargetIP: candidate.TargetIP, PrefixCIDR: prefix.String()})
		}
		generated, err := targets.FromPrefix(prefix.String(), desired-len(items), options.Seed+"|"+entry.AgentID+"|"+prefix.String(), seen)
		if err == nil {
			items = append(items, generated...)
		}
		if len(items) > 0 {
			queues[tier] = append(queues[tier], prefixQueue{prefix: prefix.String(), items: items})
		}
	}

	flatten := func(groups []prefixQueue) []model.ScanTarget {
		sort.Slice(groups, func(i, j int) bool {
			return stableScore(options.Seed, groups[i].prefix) < stableScore(options.Seed, groups[j].prefix)
		})
		out := make([]model.ScanTarget, 0)
		for round := 0; ; round++ {
			added := false
			for _, group := range groups {
				if round < len(group.items) {
					out = append(out, group.items[round])
					added = true
				}
			}
			if !added {
				return out
			}
		}
	}

	flat := map[string][]model.ScanTarget{
		TierChampion:    flatten(queues[TierChampion]),
		TierChallenger:  flatten(queues[TierChallenger]),
		TierObservation: flatten(queues[TierObservation]),
	}
	positions := map[string]int{TierChampion: 0, TierChallenger: 0, TierObservation: 0}
	pattern := []string{TierChampion, TierChampion, TierChampion, TierChallenger, TierChallenger, TierObservation}
	result := make([]model.ScanTarget, 0, options.Budget)
	for len(result) < options.Budget {
		added := false
		for _, tier := range pattern {
			position := positions[tier]
			if position >= len(flat[tier]) {
				continue
			}
			result = append(result, flat[tier][position])
			positions[tier] = position + 1
			added = true
			if len(result) >= options.Budget {
				break
			}
		}
		if !added {
			break
		}
	}
	return result
}

func intervalForTier(tier string) time.Duration {
	switch normalizedTier(tier) {
	case TierChampion:
		return time.Hour
	case TierChallenger:
		return 6 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func samplesForTier(tier string) int {
	switch normalizedTier(tier) {
	case TierChampion:
		return 6
	case TierChallenger:
		return 3
	default:
		return 1
	}
}

func anchorsForTier(tier string) int {
	switch normalizedTier(tier) {
	case TierChampion:
		return 2
	case TierChallenger:
		return 1
	default:
		return 0
	}
}

func normalizedTier(tier string) string {
	switch tier {
	case TierChampion, TierChallenger:
		return tier
	default:
		return TierObservation
	}
}

func positiveWithin(value, maximum float64) bool {
	return value > 0 && maximum > 0 && value <= maximum
}

func stableScore(parts ...string) uint64 {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return binary.BigEndian.Uint64(hash.Sum(nil)[:8])
}
