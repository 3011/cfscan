package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
)

type Config struct {
	Scheme    string
	Hostname  string
	Path      string
	Port      int
	Attempts  int
	TimeoutMS int
}

type timings struct {
	connectStart time.Time
	tlsStart     time.Time
	requestStart time.Time
	tcp          time.Duration
	tls          time.Duration
	ttfb         time.Duration
}

func Run(ctx context.Context, task model.ScanTask, cfg Config) model.ProbeResult {
	if cfg.Attempts < 1 {
		cfg.Attempts = 1
	}
	if cfg.TimeoutMS < 500 {
		cfg.TimeoutMS = 5000
	}

	result := model.ProbeResult{TaskID: task.ID, TargetIP: task.TargetIP, Attempts: cfg.Attempts}
	var tcpTotal, tlsTotal, ttfbTotal, totalTotal float64
	var lastError error
	var lastCode string
	for i := 0; i < cfg.Attempts; i++ {
		attempt, err := runAttempt(ctx, task.TargetIP, cfg)
		if err != nil {
			lastError = err
			lastCode = classifyError(err)
			continue
		}
		result.SuccessfulTries++
		tcpTotal += attempt.TCPConnectMS
		tlsTotal += attempt.TLSHandshakeMS
		ttfbTotal += attempt.TTFBMS
		totalTotal += attempt.TotalMS
		result.HTTPStatus = attempt.HTTPStatus
		result.HTTPVersion = attempt.HTTPVersion
		result.TLSVersion = attempt.TLSVersion
		if attempt.Colo != "" {
			result.Colo = attempt.Colo
		}
		if attempt.CFRay != "" {
			result.CFRay = attempt.CFRay
		}
	}

	result.PacketLoss = float64(cfg.Attempts-result.SuccessfulTries) / float64(cfg.Attempts) * 100
	result.Available = result.SuccessfulTries > 0
	if result.SuccessfulTries > 0 {
		divisor := float64(result.SuccessfulTries)
		result.TCPConnectMS = tcpTotal / divisor
		result.TLSHandshakeMS = tlsTotal / divisor
		result.TTFBMS = ttfbTotal / divisor
		result.TotalMS = totalTotal / divisor
		result.LatencyMS = result.TTFBMS
	} else {
		result.ErrorCode = lastCode
		if lastError != nil {
			result.ErrorMessage = truncate(lastError.Error(), 500)
		}
	}
	return result
}

type attemptResult struct {
	TCPConnectMS   float64
	TLSHandshakeMS float64
	TTFBMS         float64
	TotalMS        float64
	HTTPStatus     int
	HTTPVersion    string
	TLSVersion     string
	Colo           string
	CFRay          string
}

func runAttempt(parent context.Context, targetIP string, cfg Config) (attemptResult, error) {
	ctx, cancel := context.WithTimeout(parent, time.Duration(cfg.TimeoutMS)*time.Millisecond)
	defer cancel()

	port := cfg.Port
	if port == 0 {
		if cfg.Scheme == "http" {
			port = 80
		} else {
			port = 443
		}
	}
	dialAddress := net.JoinHostPort(targetIP, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond}
	transport := &http.Transport{
		Proxy:               nil,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		TLSClientConfig: &tls.Config{
			ServerName: cfg.Hostname,
			MinVersion: tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, dialAddress)
		},
	}
	defer transport.CloseIdleConnections()

	path := cfg.Path
	if path == "" {
		path = "/cdn-cgi/trace"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	host := cfg.Hostname
	if port != 80 && port != 443 {
		host = net.JoinHostPort(host, strconv.Itoa(port))
	}
	u := url.URL{Scheme: cfg.Scheme, Host: host, Path: path}

	timing := &timings{requestStart: time.Now()}
	trace := &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) { timing.connectStart = time.Now() },
		ConnectDone: func(_, _ string, _ error) {
			if !timing.connectStart.IsZero() {
				timing.tcp = time.Since(timing.connectStart)
			}
		},
		TLSHandshakeStart: func() { timing.tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			if !timing.tlsStart.IsZero() {
				timing.tls = time.Since(timing.tlsStart)
			}
		},
		GotFirstResponseByte: func() { timing.ttfb = time.Since(timing.requestStart) },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, u.String(), nil)
	if err != nil {
		return attemptResult{}, fmt.Errorf("create probe request: %w", err)
	}
	req.Host = cfg.Hostname
	req.Header.Set("User-Agent", "cfscan-agent/1.0")
	req.Header.Set("Accept", "text/plain, */*")

	started := time.Now()
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return attemptResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	if err != nil {
		return attemptResult{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		return attemptResult{}, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	cfRay := resp.Header.Get("CF-RAY")
	colo := parseColo(string(body), cfRay)
	tlsVersion := ""
	if resp.TLS != nil {
		tlsVersion = tlsVersionName(resp.TLS.Version)
	}
	return attemptResult{
		TCPConnectMS:   milliseconds(timing.tcp),
		TLSHandshakeMS: milliseconds(timing.tls),
		TTFBMS:         milliseconds(timing.ttfb),
		TotalMS:        milliseconds(time.Since(started)),
		HTTPStatus:     resp.StatusCode,
		HTTPVersion:    resp.Proto,
		TLSVersion:     tlsVersion,
		Colo:           colo,
		CFRay:          cfRay,
	}, nil
}

func parseColo(body, cfRay string) string {
	for _, line := range strings.Split(body, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "colo="); ok {
			value = strings.ToUpper(strings.TrimSpace(value))
			if len(value) == 3 {
				return value
			}
		}
	}
	if index := strings.LastIndex(cfRay, "-"); index >= 0 && len(cfRay[index+1:]) == 3 {
		return strings.ToUpper(cfRay[index+1:])
	}
	return ""
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") {
		return "TIMEOUT"
	}
	var netErr net.Error
	if ok := errorAs(err, &netErr); ok {
		if netErr.Timeout() {
			return "TIMEOUT"
		}
		return "NETWORK_ERROR"
	}
	if strings.Contains(err.Error(), "HTTP status") {
		return "HTTP_ERROR"
	}
	return "PROBE_ERROR"
}

func errorAs(err error, target any) bool {
	switch value := target.(type) {
	case *net.Error:
		for err != nil {
			if candidate, ok := err.(net.Error); ok {
				*value = candidate
				return true
			}
			unwrapper, ok := err.(interface{ Unwrap() error })
			if !ok {
				break
			}
			err = unwrapper.Unwrap()
		}
	}
	return false
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}

func milliseconds(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
