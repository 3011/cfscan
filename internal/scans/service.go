package scans

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/targets"
)

type Store interface {
	ListActivePrefixes(context.Context, bool) ([]model.Prefix, error)
	CreateScanJob(context.Context, model.CreateScanJobRequest, []string) (model.ScanJob, error)
}

type PrefixSyncer interface {
	Sync(context.Context) (model.SourceStatus, error)
	SyncEnabledASNs(context.Context) (model.ASNSyncSummary, error)
}

const (
	SamplingModeCount        = "count"
	SamplingModeOnePerPrefix = "one_per_prefix"
)

type Service struct {
	store  Store
	syncer PrefixSyncer
}

func NewService(store Store, syncer PrefixSyncer) *Service {
	return &Service{store: store, syncer: syncer}
}

func ApplyDefaults(input *model.CreateScanJobRequest) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "Cloudflare scan " + time.Now().Format("01-02 15:04")
	}
	if input.SamplingMode == "" {
		input.SamplingMode = SamplingModeCount
	}
	if input.TargetCount == 0 {
		input.TargetCount = 128
	}
	if input.Scheme == "" {
		input.Scheme = "https"
	}
	if input.Hostname == "" {
		input.Hostname = "cloudflare.com"
	}
	if input.Path == "" {
		input.Path = "/cdn-cgi/trace"
	}
	if input.Port == 0 {
		if input.Scheme == "http" {
			input.Port = 80
		} else {
			input.Port = 443
		}
	}
	if input.Attempts == 0 {
		input.Attempts = 3
	}
	if input.TimeoutMS == 0 {
		input.TimeoutMS = 5000
	}
	if input.MaxLatencyMS == 0 {
		input.MaxLatencyMS = 1000
	}
	if input.MaxPacketLoss == 0 {
		input.MaxPacketLoss = 50
	}
	if input.BlacklistMinutes == 0 {
		input.BlacklistMinutes = 60
	}
}

func Validate(input model.CreateScanJobRequest) error {
	if input.SamplingMode != SamplingModeCount && input.SamplingMode != SamplingModeOnePerPrefix {
		return fmt.Errorf("sampling_mode must be count or one_per_prefix")
	}
	if input.SamplingMode == SamplingModeCount && (input.TargetCount < 1 || input.TargetCount > 10000) {
		return fmt.Errorf("target_count must be between 1 and 10000")
	}
	if input.Scheme != "https" && input.Scheme != "http" {
		return fmt.Errorf("scheme must be http or https")
	}
	if input.Hostname == "" || strings.ContainsAny(input.Hostname, " /\\") {
		return fmt.Errorf("valid hostname is required")
	}
	if input.Path == "" || !strings.HasPrefix(input.Path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if input.Port < 1 || input.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if input.Attempts < 1 || input.Attempts > 10 {
		return fmt.Errorf("attempts must be between 1 and 10")
	}
	if input.TimeoutMS < 500 || input.TimeoutMS > 30000 {
		return fmt.Errorf("timeout_ms must be between 500 and 30000")
	}
	if input.MaxLatencyMS <= 0 || input.MaxLatencyMS > 60000 {
		return fmt.Errorf("max_latency_ms must be between 1 and 60000")
	}
	if input.MaxPacketLoss < 0 || input.MaxPacketLoss > 100 {
		return fmt.Errorf("max_packet_loss must be between 0 and 100")
	}
	if input.BlacklistMinutes < 1 || input.BlacklistMinutes > 10080 {
		return fmt.Errorf("blacklist_minutes must be between 1 and 10080")
	}
	return nil
}

func (s *Service) Create(ctx context.Context, input model.CreateScanJobRequest, kind string) (model.ScanJob, error) {
	ApplyDefaults(&input)
	if err := Validate(input); err != nil {
		return model.ScanJob{}, err
	}
	if kind == "" {
		kind = "normal"
	}
	input.Kind = kind

	prefixes, err := s.store.ListActivePrefixes(ctx, input.IncludeIPv6)
	if err != nil {
		return model.ScanJob{}, err
	}
	if len(prefixes) == 0 {
		_, officialErr := s.syncer.Sync(ctx)
		_, asnErr := s.syncer.SyncEnabledASNs(ctx)
		if officialErr != nil && asnErr != nil {
			return model.ScanJob{}, fmt.Errorf("official source: %v; ASN sources: %v", officialErr, asnErr)
		}
		prefixes, err = s.store.ListActivePrefixes(ctx, input.IncludeIPv6)
		if err != nil {
			return model.ScanJob{}, err
		}
	}

	seed := fmt.Sprintf("%s|%s|%s", kind, input.Name, time.Now().UTC().Format(time.RFC3339Nano))
	var sampled []string
	if input.SamplingMode == SamplingModeOnePerPrefix {
		sampled, err = targets.OnePerPrefix(prefixes, input.IncludeIPv6, seed)
	} else {
		sampled, err = targets.Sample(prefixes, input.TargetCount, input.IncludeIPv6, seed)
	}
	if err != nil {
		return model.ScanJob{}, err
	}
	return s.store.CreateScanJob(ctx, input, sampled)
}
