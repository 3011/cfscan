package league

import (
	"testing"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestEvaluatePromotesWithEnoughDiverseGoodSamples(t *testing.T) {
	now := time.Now().UTC()
	entry := model.PrefixLeagueEntry{
		Tier: TierObservation, SampleCount: 4, DistinctIPCount: 3,
		AvailabilityRate: 100, LatencyP95MS: 80, PacketLossAvg: 0,
		RecentSampleCount: 4, RecentAvailabilityRate: 100, RecentLatencyP95MS: 80,
		LastResultAt: &now,
	}
	got := Evaluate(entry, Thresholds{MaxLatencyMS: 100, MaxPacketLoss: 5})
	if got.Tier != TierChallenger {
		t.Fatalf("got tier %q", got.Tier)
	}
}

func TestEvaluateDemotionRequiresRepeatedBadEvidence(t *testing.T) {
	first := time.Now().UTC()
	entry := model.PrefixLeagueEntry{
		Tier: TierChampion, SampleCount: 20, DistinctIPCount: 10,
		AvailabilityRate: 95, LatencyP95MS: 80,
		RecentSampleCount: 4, RecentAvailabilityRate: 50, RecentLatencyP95MS: 200,
		LastResultAt: &first,
	}
	entry = Evaluate(entry, Thresholds{MaxLatencyMS: 100, MaxPacketLoss: 5})
	if entry.Tier != TierChampion || entry.BadStreak != 1 {
		t.Fatalf("unexpected first evaluation: %+v", entry)
	}
	second := first.Add(time.Hour)
	entry.LastResultAt = &second
	entry = Evaluate(entry, Thresholds{MaxLatencyMS: 100, MaxPacketLoss: 5})
	if entry.Tier != TierChallenger || entry.BadStreak != 0 {
		t.Fatalf("unexpected second evaluation: %+v", entry)
	}
}

func TestEvaluateSevereOutageDropsChampionToObservation(t *testing.T) {
	now := time.Now().UTC()
	entry := model.PrefixLeagueEntry{
		Tier: TierChampion, RecentSampleCount: 3, RecentAvailabilityRate: 0,
		LastResultAt: &now,
	}
	got := Evaluate(entry, Thresholds{MaxLatencyMS: 100, MaxPacketLoss: 5})
	if got.Tier != TierObservation {
		t.Fatalf("got tier %q", got.Tier)
	}
}

func TestPlanAgentKeepsBudgetAndUsesCandidateAnchor(t *testing.T) {
	now := time.Now().UTC()
	entries := []model.PrefixLeagueEntry{
		{AgentID: "agent", PrefixCIDR: "104.16.0.0/24", Tier: TierChampion, Active: true},
		{AgentID: "agent", PrefixCIDR: "172.64.0.0/24", Tier: TierObservation, Active: true},
	}
	items := PlanAgent(entries, map[string][]model.LeagueCandidate{
		"104.16.0.0/24": {{TargetIP: "104.16.0.10"}},
	}, PlanOptions{Now: now, Seed: "plan", Budget: 4, Force: true})
	if len(items) != 4 {
		t.Fatalf("got %d targets: %+v", len(items), items)
	}
	found := false
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.TargetIP] {
			t.Fatalf("duplicate %s", item.TargetIP)
		}
		seen[item.TargetIP] = true
		if item.TargetIP == "104.16.0.10" {
			found = true
		}
	}
	if !found {
		t.Fatalf("candidate anchor missing: %+v", items)
	}
}

func TestDueUsesTierIntervals(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		tier        string
		last        time.Time
		want        bool
		description string
	}{
		{TierChampion, now.Add(-59 * time.Minute), false, "champion before one hour"},
		{TierChampion, now.Add(-time.Hour), true, "champion at one hour"},
		{TierChallenger, now.Add(-5 * time.Hour), false, "challenger before six hours"},
		{TierChallenger, now.Add(-6 * time.Hour), true, "challenger at six hours"},
		{TierObservation, now.Add(-23 * time.Hour), false, "observation before one day"},
		{TierObservation, now.Add(-24 * time.Hour), true, "observation at one day"},
	}
	for _, item := range cases {
		last := item.last
		entry := model.PrefixLeagueEntry{Tier: item.tier, Active: true, LastScheduledAt: &last}
		if got := Due(entry, now, false); got != item.want {
			t.Fatalf("%s: got %v want %v", item.description, got, item.want)
		}
	}
	inactive := model.PrefixLeagueEntry{Tier: TierChampion, Active: false}
	if Due(inactive, now, true) {
		t.Fatal("inactive prefix must not be scheduled even when forced")
	}
}
