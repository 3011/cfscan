package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/3011/cfscan/internal/enrollment"
	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/store"
	"github.com/go-chi/chi/v5"
)

const (
	maxAgentConcurrency  = 4096
	maxAgentNameLength   = 128
	maxAgentRegionLength = 128
	maxAgentLabelLength  = 64
	maxCredentialLength  = 256
)

func (a *API) createDeviceEnrollment(w http.ResponseWriter, r *http.Request) {
	if allowed, retry := a.enrollmentLimiter.Allow(loginClientKey(r)); !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(max(int(retry.Seconds()), 1)))
		writeError(w, http.StatusTooManyRequests, "too_many_enrollment_requests", "too many enrollment requests")
		return
	}
	var input model.CreateDeviceEnrollmentRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.OS = strings.TrimSpace(input.OS)
	input.Architecture = strings.TrimSpace(input.Architecture)
	input.Version = strings.TrimSpace(input.Version)
	if !textLengthBetween(input.Name, 1, maxAgentNameLength) ||
		!textLengthBetween(input.OS, 0, maxAgentLabelLength) ||
		!textLengthBetween(input.Architecture, 0, maxAgentLabelLength) ||
		!textLengthBetween(input.Version, 0, maxAgentLabelLength) ||
		input.Concurrency < 1 || input.Concurrency > maxAgentConcurrency {
		writeError(w, http.StatusBadRequest, "invalid_request", "Agent metadata is too long or concurrency is outside 1 to 4096")
		return
	}
	pairingToken, err := enrollment.GeneratePairingToken()
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	ttl := time.Duration(a.enrollmentConfig.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	_, err = a.store.CreateAgentEnrollment(r.Context(), model.CreateAgentEnrollment{
		TokenHash: enrollment.HashSecret(pairingToken), Mode: model.AgentEnrollmentModeDevice,
		Status: model.AgentEnrollmentPending, RequestedName: input.Name, OS: input.OS,
		Architecture: input.Architecture, Version: input.Version, RequestedConcurrency: input.Concurrency,
		Concurrency: input.Concurrency, ExpiresAt: time.Now().UTC().Add(ttl),
	})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	verificationURI := strings.TrimRight(a.enrollmentConfig.PublicURL, "/") + "/agents/pair"
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, model.CreateDeviceEnrollmentResponse{
		PairingToken: pairingToken, VerificationURI: verificationURI,
		VerificationURIComplete: verificationURI + "/" + pairingToken,
		ExpiresIn:               int(ttl.Seconds()), Interval: max(a.enrollmentConfig.PollInterval, 1),
	})
}

func (a *API) claimAgentEnrollment(w http.ResponseWriter, r *http.Request) {
	var input model.ClaimAgentEnrollmentRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.PairingToken = strings.TrimSpace(input.PairingToken)
	if !enrollment.LooksLikeUUID(input.PairingToken) || !enrollment.LooksLikeUUID(input.CredentialID) || strings.TrimSpace(input.CredentialSecret) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "valid pairing_token, credential_id and credential_secret are required")
		return
	}
	item, err := a.store.ClaimAgentEnrollment(r.Context(), enrollment.HashSecret(input.PairingToken), input.CredentialID,
		enrollment.HashSecret(input.CredentialSecret), input.OS, input.Architecture, input.Version)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrEnrollmentPending):
			writeJSON(w, http.StatusAccepted, model.ClaimAgentEnrollmentResponse{Status: model.AgentEnrollmentPending, Interval: max(a.enrollmentConfig.PollInterval, 1)})
		case errors.Is(err, store.ErrEnrollmentExpired):
			writeError(w, http.StatusGone, "enrollment_expired", "this pairing request has expired")
		case errors.Is(err, store.ErrEnrollmentRejected):
			writeError(w, http.StatusForbidden, "enrollment_rejected", "this pairing request was rejected")
		case errors.Is(err, store.ErrEnrollmentRevoked):
			writeError(w, http.StatusGone, "enrollment_revoked", "this pairing request was revoked")
		case errors.Is(err, store.ErrEnrollmentClaimed):
			writeError(w, http.StatusConflict, "enrollment_claimed", "this pairing request was already used")
		case errors.Is(err, store.ErrEnrollmentConflict):
			writeError(w, http.StatusConflict, "agent_name_conflict", "an Agent with this name already exists")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		default:
			a.internalError(w, r, err)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, model.ClaimAgentEnrollmentResponse{
		Status: model.AgentEnrollmentClaimed, AgentID: item.AgentID, Name: item.Name, Concurrency: item.Concurrency,
	})
}

func (a *API) agentEnrollmentConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, a.enrollmentConfig)
}

func (a *API) listAgentEnrollments(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListAgentEnrollments(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getAgentEnrollment(w http.ResponseWriter, r *http.Request) {
	pairingToken := normalizePairingToken(chi.URLParam(r, "pairingToken"))
	if !enrollment.LooksLikeUUID(pairingToken) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	item, err := a.store.GetAgentEnrollment(r.Context(), enrollment.HashSecret(pairingToken))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) approveAgentEnrollment(w http.ResponseWriter, r *http.Request) {
	pairingToken := normalizePairingToken(chi.URLParam(r, "pairingToken"))
	if !enrollment.LooksLikeUUID(pairingToken) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	input, ok := decodeEnrollmentApproval(w, r)
	if !ok {
		return
	}
	item, err := a.store.ApproveAgentEnrollment(r.Context(), enrollment.HashSecret(pairingToken), input)
	if err != nil {
		a.writeEnrollmentAdminError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) rejectAgentEnrollment(w http.ResponseWriter, r *http.Request) {
	pairingToken := normalizePairingToken(chi.URLParam(r, "pairingToken"))
	if !enrollment.LooksLikeUUID(pairingToken) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	item, err := a.store.RejectAgentEnrollment(r.Context(), enrollment.HashSecret(pairingToken))
	if err != nil {
		a.writeEnrollmentAdminError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) createPreauthorizedEnrollment(w http.ResponseWriter, r *http.Request) {
	var input model.CreatePreauthorizedEnrollmentRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name, input.Region, input.Continent = strings.TrimSpace(input.Name), strings.TrimSpace(input.Region), strings.TrimSpace(input.Continent)
	if !validApprovedAgent(input.Name, input.Region, input.Continent, input.Concurrency) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Agent name, region, continent, or concurrency is invalid")
		return
	}
	if input.TTLMinutes == 0 {
		input.TTLMinutes = 30
	}
	if input.TTLMinutes < 5 || input.TTLMinutes > 1440 {
		writeError(w, http.StatusBadRequest, "invalid_request", "ttl_minutes must be between 5 and 1440")
		return
	}
	pairingToken, err := enrollment.GeneratePairingToken()
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	ttl := time.Duration(input.TTLMinutes) * time.Minute
	item, err := a.store.CreateAgentEnrollment(r.Context(), model.CreateAgentEnrollment{
		TokenHash: enrollment.HashSecret(pairingToken), Mode: model.AgentEnrollmentModePreauthorized,
		Status: model.AgentEnrollmentApproved, RequestedName: input.Name, RequestedConcurrency: input.Concurrency,
		Name: input.Name, Region: input.Region, Continent: input.Continent, Concurrency: input.Concurrency,
		ExpiresAt: time.Now().UTC().Add(ttl),
	})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, model.CreatePreauthorizedEnrollmentResponse{Enrollment: item, PairingToken: pairingToken, ExpiresIn: int(ttl.Seconds())})
}

func (a *API) writeEnrollmentAdminError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
	case errors.Is(err, store.ErrEnrollmentExpired):
		writeError(w, http.StatusGone, "enrollment_expired", "this pairing request has expired")
	case errors.Is(err, store.ErrEnrollmentRejected):
		writeError(w, http.StatusConflict, "enrollment_rejected", "this pairing request was already rejected")
	case errors.Is(err, store.ErrEnrollmentClaimed):
		writeError(w, http.StatusConflict, "enrollment_claimed", "this pairing request was already used")
	default:
		a.internalError(w, r, err)
	}
}

func (a *API) getAgentEnrollmentByID(w http.ResponseWriter, r *http.Request) {
	enrollmentID := strings.TrimSpace(chi.URLParam(r, "enrollmentID"))
	if !enrollment.LooksLikeUUID(enrollmentID) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	item, err := a.store.GetAgentEnrollmentByID(r.Context(), enrollmentID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) approveAgentEnrollmentByID(w http.ResponseWriter, r *http.Request) {
	enrollmentID := strings.TrimSpace(chi.URLParam(r, "enrollmentID"))
	if !enrollment.LooksLikeUUID(enrollmentID) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	input, ok := decodeEnrollmentApproval(w, r)
	if !ok {
		return
	}
	item, err := a.store.ApproveAgentEnrollmentByID(r.Context(), enrollmentID, input)
	if err != nil {
		a.writeEnrollmentAdminError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) rejectAgentEnrollmentByID(w http.ResponseWriter, r *http.Request) {
	enrollmentID := strings.TrimSpace(chi.URLParam(r, "enrollmentID"))
	if !enrollment.LooksLikeUUID(enrollmentID) {
		writeError(w, http.StatusNotFound, "enrollment_not_found", "pairing request not found")
		return
	}
	item, err := a.store.RejectAgentEnrollmentByID(r.Context(), enrollmentID)
	if err != nil {
		a.writeEnrollmentAdminError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func normalizePairingToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func textLengthBetween(value string, minimum, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func validApprovedAgent(name, region, continent string, concurrency int) bool {
	return textLengthBetween(name, 1, maxAgentNameLength) &&
		textLengthBetween(region, 1, maxAgentRegionLength) &&
		textLengthBetween(continent, 1, maxAgentLabelLength) &&
		concurrency >= 1 && concurrency <= maxAgentConcurrency
}

func decodeEnrollmentApproval(w http.ResponseWriter, r *http.Request) (model.ApproveAgentEnrollmentRequest, bool) {
	var input model.ApproveAgentEnrollmentRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return input, false
	}
	input.Name, input.Region, input.Continent = strings.TrimSpace(input.Name), strings.TrimSpace(input.Region), strings.TrimSpace(input.Continent)
	if !validApprovedAgent(input.Name, input.Region, input.Continent, input.Concurrency) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Agent name, region, continent, or concurrency is invalid")
		return input, false
	}
	return input, true
}
