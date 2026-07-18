package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/config"
	"github.com/3011/cfscan/v2/internal/enrollment"
	"github.com/3011/cfscan/v2/internal/model"
)

type enrollmentOptions struct {
	serverURL         string
	pairingToken      string
	tokenFile         string
	tokenStdin        bool
	name              string
	concurrency       int
	identityFile      string
	allowInsecureHTTP bool
	pairOnly          bool
}

func runCLI(ctx context.Context, args []string, logger *slog.Logger) error {
	if len(args) == 0 || args[0] == "run" {
		return runSavedIdentity(ctx, logger)
	}
	switch args[0] {
	case "connect":
		return runEnrollment(ctx, args[1:], false, logger)
	case "join":
		return runEnrollment(ctx, args[1:], true, logger)
	case "version", "--version", "-version":
		fmt.Println(agentVersion())
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q; use connect, join, run, or version", args[0])
	}
}

func printUsage() {
	fmt.Print(`CF Scanner Agent

Commands:
  connect   create a pairing request and wait for Web approval
  join      use a preauthorized one-time pairing token
  run       run with the saved independent identity
  version   print the Agent version
`)
}

func runEnrollment(ctx context.Context, args []string, preauthorized bool, logger *slog.Logger) error {
	cfg := config.LoadAgent()
	hostname, _ := os.Hostname()
	options := enrollmentOptions{name: hostname, concurrency: cfg.Concurrency, serverURL: cfg.CenterURL, identityFile: defaultIdentityPath()}
	name := "connect"
	if preauthorized {
		name = "join"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.serverURL, "server", options.serverURL, "Center Agent API URL")
	flags.StringVar(&options.pairingToken, "token", "", "one-time pairing token")
	flags.StringVar(&options.tokenFile, "token-file", "", "read the one-time pairing token from a file")
	flags.BoolVar(&options.tokenStdin, "token-stdin", false, "read the one-time pairing token from stdin")
	flags.StringVar(&options.name, "name", options.name, "requested Agent name")
	flags.IntVar(&options.concurrency, "concurrency", options.concurrency, "maximum concurrent probes")
	flags.StringVar(&options.identityFile, "identity-file", options.identityFile, "path used to store the long-term identity")
	flags.BoolVar(&options.allowInsecureHTTP, "allow-insecure-http", false, "allow cleartext HTTP to a non-loopback server")
	flags.BoolVar(&options.pairOnly, "pair-only", false, "save the identity and exit instead of running")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if identity, err := loadIdentity(options.identityFile); err == nil {
		fmt.Printf("Agent identity already exists at %s\n", options.identityFile)
		if options.pairOnly {
			return nil
		}
		return runAgentLoop(ctx, &client{baseURL: identity.ServerURL, token: identity.Token, http: defaultHTTPClient()}, identity.Concurrency, logger)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load existing identity: %w", err)
	}
	options.serverURL = strings.TrimRight(strings.TrimSpace(options.serverURL), "/")
	if err := validateEnrollmentServer(options.serverURL, options.allowInsecureHTTP); err != nil {
		return err
	}
	if options.concurrency < 1 || options.concurrency > 4096 {
		return errors.New("concurrency must be between 1 and 4096")
	}
	apiClient := &client{baseURL: options.serverURL, http: defaultHTTPClient()}
	interval := 3
	if preauthorized {
		token, err := enrollmentToken(options)
		if err != nil {
			return err
		}
		options.pairingToken = token
	} else {
		var response model.CreateDeviceEnrollmentResponse
		_, err := apiClient.postPublicJSON(ctx, "/api/v1/agent/enrollments", model.CreateDeviceEnrollmentRequest{
			Name: strings.TrimSpace(options.name), OS: runtime.GOOS, Architecture: runtime.GOARCH,
			Version: agentVersion(), Concurrency: options.concurrency,
		}, &response)
		if err != nil {
			return fmt.Errorf("create pairing request: %w", err)
		}
		options.pairingToken = response.PairingToken
		interval = max(response.Interval, 1)
		fmt.Println("Waiting for administrator approval")
		fmt.Println()
		fmt.Printf("Open: %s\n", response.VerificationURIComplete)
		fmt.Printf("Pairing token: %s\n", response.PairingToken)
		fmt.Printf("Expires in: %s\n", (time.Duration(response.ExpiresIn) * time.Second).Round(time.Second))
		fmt.Println()
		fmt.Println("Waiting for approval...")
	}

	credentialID, credentialSecret, longTermToken, err := enrollment.GenerateCredential()
	if err != nil {
		return err
	}
	claimed, err := waitForEnrollment(ctx, apiClient, options.pairingToken, credentialID, credentialSecret, interval)
	if err != nil {
		return err
	}
	identity := agentIdentity{
		ServerURL: options.serverURL, AgentID: claimed.AgentID, Token: longTermToken,
		Name: claimed.Name, Concurrency: claimed.Concurrency,
	}
	if err := saveIdentity(options.identityFile, identity); err != nil {
		return err
	}
	fmt.Printf("Agent connected successfully: %s\n", claimed.Name)
	fmt.Printf("Identity saved to: %s\n", options.identityFile)
	if options.pairOnly {
		return nil
	}
	return runAgentLoop(ctx, &client{baseURL: options.serverURL, token: longTermToken, http: defaultHTTPClient()}, claimed.Concurrency, logger)
}

func waitForEnrollment(ctx context.Context, apiClient *client, pairingToken, credentialID, credentialSecret string, interval int) (model.ClaimAgentEnrollmentResponse, error) {
	for {
		var result model.ClaimAgentEnrollmentResponse
		status, err := apiClient.postPublicJSON(ctx, "/api/v1/agent/enrollments/claim", model.ClaimAgentEnrollmentRequest{
			PairingToken: pairingToken, CredentialID: credentialID, CredentialSecret: credentialSecret,
			OS: runtime.GOOS, Architecture: runtime.GOARCH, Version: agentVersion(),
		}, &result)
		if err == nil && status == 200 && result.Status == model.AgentEnrollmentClaimed {
			return result, nil
		}
		if err != nil {
			var remote *apiError
			if errors.As(err, &remote) {
				switch remote.Code {
				case "enrollment_rejected", "enrollment_expired", "enrollment_revoked", "enrollment_claimed", "agent_name_conflict", "enrollment_not_found":
					return model.ClaimAgentEnrollmentResponse{}, err
				}
			}
		}
		if result.Interval > 0 {
			interval = result.Interval
		}
		timer := time.NewTimer(time.Duration(max(interval, 1)) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return model.ClaimAgentEnrollmentResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func enrollmentToken(options enrollmentOptions) (string, error) {
	values := 0
	if strings.TrimSpace(options.pairingToken) != "" {
		values++
	}
	if options.tokenFile != "" {
		values++
	}
	if options.tokenStdin {
		values++
	}
	if values != 1 {
		return "", errors.New("join requires exactly one of --token, --token-file, or --token-stdin")
	}
	var value string
	var err error
	switch {
	case options.tokenFile != "":
		data, readErr := os.ReadFile(options.tokenFile)
		value, err = string(data), readErr
	case options.tokenStdin:
		reader := bufio.NewReader(io.LimitReader(os.Stdin, 4096))
		value, err = reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			err = nil
		}
	default:
		value = options.pairingToken
	}
	if err != nil {
		return "", fmt.Errorf("read pairing token: %w", err)
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if !enrollment.LooksLikeUUID(value) {
		return "", errors.New("pairing token must be a UUID")
	}
	return value, nil
}

func validateEnrollmentServer(value string, allowInsecure bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return errors.New("server must be a complete http:// or https:// URL")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" {
		return errors.New("server URL must use https://; http:// is only allowed for local development")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && ip.IsLoopback()) || allowInsecure {
		return nil
	}
	return errors.New("refusing cleartext HTTP to a non-loopback server; use HTTPS or explicitly pass --allow-insecure-http")
}
