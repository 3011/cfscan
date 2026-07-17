package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	authservice "github.com/3011/cfscan/internal/auth"
	"github.com/3011/cfscan/internal/automation"
	"github.com/3011/cfscan/internal/cloudflare"
	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/scans"
	"github.com/3011/cfscan/internal/scheduling"
	"github.com/3011/cfscan/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type API struct {
	store        store.Store
	syncer       *cloudflare.Syncer
	agentToken   string
	logger       *slog.Logger
	scanService  *scans.Service
	automation   *automation.Service
	auth         *authservice.Service
	loginLimiter *authservice.LoginLimiter
}

func New(dataStore store.Store, syncer *cloudflare.Syncer, automationService *automation.Service, authService *authservice.Service, agentToken string, logger *slog.Logger) http.Handler {
	api := &API{store: dataStore, syncer: syncer, automation: automationService, auth: authService, loginLimiter: authservice.NewLoginLimiter(), agentToken: agentToken, logger: logger, scanService: scans.NewService(dataStore, syncer)}
	router := chi.NewRouter()
	router.Use(api.serverTime, middleware.RequestID, middleware.RealIP, middleware.Recoverer, api.accessLog)
	router.Get("/healthz", api.health)
	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", api.login)

		r.Route("/agent", func(r chi.Router) {
			r.Use(api.requireAgentToken)
			r.Post("/register", api.registerAgent)
			r.Post("/heartbeat", api.heartbeat)
			r.Post("/tasks/claim", api.claimTask)
			r.Post("/tasks/results", api.submitResults)
		})

		r.Group(func(r chi.Router) {
			r.Use(api.requireSession)
			r.Get("/auth/me", api.me)
			r.Post("/auth/logout", api.logout)

			r.Get("/overview", api.overview)
			r.Get("/agents", api.listAgents)
			r.Get("/sources/cloudflare", api.sourceStatus)
			r.Get("/sources/asns", api.listASNSources)
			r.Get("/jobs", api.listJobs)
			r.Get("/jobs/{jobID}", api.getJob)
			r.Get("/schedules", api.listSchedules)
			r.Get("/results", api.listResults)
			r.Get("/results/facets", api.listResultFacets)
			r.Get("/results/jobs", api.listResultJobs)
			r.Get("/colos", api.listColos)
			r.Get("/blacklist", api.listBlacklist)
			r.Get("/automation/blacklist-recheck", api.getBlacklistRecheckSettings)
			r.Get("/automation/source-syncs", api.listSourceSyncSchedules)
			r.Get("/automation/runs", api.listAutomationRuns)

			r.With(api.requireAdmin).Post("/sources/cloudflare/sync", api.syncSource)
			r.With(api.requireAdmin).Post("/sources/asns", api.createASNSource)
			r.With(api.requireAdmin).Post("/sources/asns/sync", api.syncASNSources)
			r.With(api.requireAdmin).Patch("/sources/asns/{asn}", api.updateASNSource)
			r.With(api.requireAdmin).Delete("/sources/asns/{asn}", api.deleteASNSource)
			r.With(api.requireAdmin).Post("/sources/asns/{asn}/sync", api.syncASNSource)
			r.With(api.requireAdmin).Post("/jobs", api.createJob)
			r.With(api.requireAdmin).Post("/jobs/{jobID}/stop", api.stopJob)
			r.With(api.requireAdmin).Post("/schedules", api.createSchedule)
			r.With(api.requireAdmin).Put("/schedules/{scheduleID}", api.updateSchedule)
			r.With(api.requireAdmin).Delete("/schedules/{scheduleID}", api.deleteSchedule)
			r.With(api.requireAdmin).Post("/schedules/{scheduleID}/run", api.runScheduleNow)
			r.With(api.requireAdmin).Post("/blacklist/recheck", api.recheckBlacklist)
			r.With(api.requireAdmin).Put("/automation/blacklist-recheck", api.updateBlacklistRecheckSettings)
			r.With(api.requireAdmin).Post("/automation/blacklist-recheck/run", api.runBlacklistRecheck)
			r.With(api.requireAdmin).Put("/automation/source-syncs/{source}", api.updateSourceSyncSchedule)
			r.With(api.requireAdmin).Post("/automation/source-syncs/{source}/run", api.runSourceSyncSchedule)

			r.With(api.requireAdmin).Route("/users", func(r chi.Router) {
				r.Get("/", api.listUsers)
				r.Post("/", api.createUser)
				r.Put("/{userID}", api.updateUser)
				r.Post("/{userID}/reset-password", api.resetUserPassword)
				r.Delete("/{userID}", api.deleteUser)
			})
		})
	})
	return router
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	result, err := a.store.Overview(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) listAgents(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListAgents(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) sourceStatus(w http.ResponseWriter, r *http.Request) {
	result, err := a.store.SourceStatus(r.Context(), cloudflare.SourceOfficial)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) syncSource(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 30*time.Second)
	defer cancel()
	result, err := a.syncer.Sync(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) listASNSources(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListASNSources(r.Context(), false)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createASNSource(w http.ResponseWriter, r *http.Request) {
	var input model.CreateASNSourceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := a.store.CreateASNSource(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_asn_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateASNSource(w http.ResponseWriter, r *http.Request) {
	asn, ok := parseASNParam(w, r)
	if !ok {
		return
	}
	var input model.UpdateASNSourceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := a.store.UpdateASNSource(r.Context(), asn, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_asn_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) deleteASNSource(w http.ResponseWriter, r *http.Request) {
	asn, ok := parseASNParam(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteASNSource(r.Context(), asn); err != nil {
		writeError(w, http.StatusBadRequest, "delete_asn_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) syncASNSource(w http.ResponseWriter, r *http.Request) {
	asn, ok := parseASNParam(w, r)
	if !ok {
		return
	}
	ctx, cancel := contextWithTimeout(r, 2*time.Minute)
	defer cancel()
	item, err := a.syncer.SyncASN(ctx, asn)
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_asn_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) syncASNSources(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 5*time.Minute)
	defer cancel()
	summary, err := a.syncer.SyncEnabledASNs(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_asns_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) createJob(w http.ResponseWriter, r *http.Request) {
	var input model.CreateScanJobRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Minute)
	defer cancel()
	job, err := a.scanService.Create(ctx, input, "normal")
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_job_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func normalizeSchedule(input *model.UpsertScanScheduleRequest) (time.Time, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.CronExpression == "" {
		input.CronExpression = "0 */6 * * *"
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	job := input.JobRequest()
	scans.ApplyDefaults(&job)
	if err := scans.Validate(job); err != nil {
		return time.Time{}, err
	}
	input.ApplyJobRequest(job)
	return scheduling.Next(input.CronExpression, input.Timezone, time.Now().UTC())
}

func (a *API) listSchedules(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListScanSchedules(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createSchedule(w http.ResponseWriter, r *http.Request) {
	var input model.UpsertScanScheduleRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	nextRun, err := normalizeSchedule(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
		return
	}
	item, err := a.store.CreateScanSchedule(r.Context(), input, nextRun)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_schedule_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateSchedule(w http.ResponseWriter, r *http.Request) {
	var input model.UpsertScanScheduleRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	nextRun, err := normalizeSchedule(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
		return
	}
	item, err := a.store.UpdateScanSchedule(r.Context(), chi.URLParam(r, "scheduleID"), input, nextRun)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_schedule_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteScanSchedule(r.Context(), chi.URLParam(r, "scheduleID")); err != nil {
		writeError(w, http.StatusBadRequest, "delete_schedule_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) runScheduleNow(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.GetScanSchedule(r.Context(), chi.URLParam(r, "scheduleID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "scan schedule not found")
		return
	}
	config, _ := json.Marshal(item)
	automationRun, startErr := a.store.StartAutomationRun(r.Context(), model.StartAutomationRunRequest{
		AutomationType: "scan_schedule", AutomationKey: item.ID, Name: item.Name,
		Trigger: "manual", ConfigSnapshot: config,
	})
	if startErr != nil {
		a.internalError(w, r, startErr)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Minute)
	defer cancel()
	job, runErr := a.scanService.Create(ctx, item.JobRequest(), "scheduled")
	var jobID *string
	summary := json.RawMessage(`{}`)
	if runErr == nil {
		jobID = &job.ID
		summary, _ = json.Marshal(map[string]any{"job_id": job.ID, "tasks": job.TotalTargets})
	}
	_ = a.store.FinishAutomationRun(r.Context(), automationRun.ID, "completed", summary, runErr)
	if err := a.store.RecordScanScheduleRun(r.Context(), item.ID, nil, jobID, runErr); err != nil {
		a.internalError(w, r, err)
		return
	}
	if runErr != nil {
		writeError(w, http.StatusBadRequest, "run_schedule_failed", runErr.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListScanJobs(r.Context(), queryInt(r, "limit", 50))
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.GetScanJob(r.Context(), chi.URLParam(r, "jobID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "scan job not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) stopJob(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.StopScanJob(r.Context(), chi.URLParam(r, "jobID"))
	if err != nil {
		writeError(w, http.StatusConflict, "stop_job_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) listColos(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListColoLocations(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func resultFilterFromRequest(r *http.Request, includeGeo bool) (model.ResultFilter, error) {
	filter := model.ResultFilter{
		View:      strings.TrimSpace(r.URL.Query().Get("view")),
		AgentID:   strings.TrimSpace(r.URL.Query().Get("agent_id")),
		JobID:     strings.TrimSpace(r.URL.Query().Get("job_id")),
		Region:    strings.TrimSpace(r.URL.Query().Get("region")),
		Continent: strings.TrimSpace(r.URL.Query().Get("continent")),
		TargetIP:  strings.TrimSpace(r.URL.Query().Get("search")),
		Page:      queryInt(r, "page", 1),
		PageSize:  queryInt(r, "page_size", 50),
		Sort:      strings.TrimSpace(r.URL.Query().Get("sort")),
		Order:     strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order"))),
	}
	if filter.View == "" {
		filter.View = "latest"
	}
	if filter.View != "latest" && filter.View != "history" {
		return model.ResultFilter{}, fmt.Errorf("view must be latest or history")
	}
	if filter.PageSize > 200 {
		return model.ResultFilter{}, fmt.Errorf("page_size must not exceed 200")
	}
	if filter.Order == "" {
		filter.Order = "asc"
	}
	if filter.Order != "asc" && filter.Order != "desc" {
		return model.ResultFilter{}, fmt.Errorf("order must be asc or desc")
	}
	if filter.Sort == "" {
		if filter.View == "history" {
			filter.Sort, filter.Order = "scanned_at", "desc"
		} else {
			filter.Sort, filter.Order = "latency_ms", "asc"
		}
	}
	allowedSorts := map[string]bool{
		"target_ip": true, "agent_name": true, "colo": true, "available": true,
		"latency_ms": true, "packet_loss": true, "http_status": true, "scanned_at": true,
	}
	if !allowedSorts[filter.Sort] {
		return model.ResultFilter{}, fmt.Errorf("unsupported result sort %q", filter.Sort)
	}
	if includeGeo {
		filter.Colo = strings.TrimSpace(r.URL.Query().Get("colo"))
		filter.ColoCity = strings.TrimSpace(r.URL.Query().Get("colo_city"))
		filter.ColoCountry = strings.TrimSpace(r.URL.Query().Get("colo_country"))
		filter.ColoContinent = strings.TrimSpace(r.URL.Query().Get("colo_continent"))
	}
	if raw := r.URL.Query().Get("available"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return model.ResultFilter{}, fmt.Errorf("available must be true or false")
		}
		filter.Available = &value
	}
	now := time.Now().UTC()
	switch strings.TrimSpace(r.URL.Query().Get("time_range")) {
	case "", "24h":
		since := now.Add(-24 * time.Hour)
		filter.Since = &since
	case "1h":
		since := now.Add(-time.Hour)
		filter.Since = &since
	case "7d":
		since := now.Add(-7 * 24 * time.Hour)
		filter.Since = &since
	case "30d":
		since := now.Add(-30 * 24 * time.Hour)
		filter.Since = &since
	case "all":
		filter.Since = nil
	default:
		return model.ResultFilter{}, fmt.Errorf("time_range must be 1h, 24h, 7d, 30d, or all")
	}
	return filter, nil
}

func (a *API) listResults(w http.ResponseWriter, r *http.Request) {
	filter, err := resultFilterFromRequest(r, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	page, err := a.store.ListResults(r.Context(), filter)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) listResultFacets(w http.ResponseWriter, r *http.Request) {
	filter, err := resultFilterFromRequest(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	items, err := a.store.ListResultColoFacets(r.Context(), filter)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listResultJobs(w http.ResponseWriter, r *http.Request) {
	filter, err := resultFilterFromRequest(r, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	// A selected job must not remove every other job from the selector itself.
	filter.JobID = ""
	items, err := a.store.ListResultJobFacets(r.Context(), filter)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listBlacklist(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListBlacklist(r.Context(), queryInt(r, "limit", 200))
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) recheckBlacklist(w http.ResponseWriter, r *http.Request) {
	result, err := a.automation.RunBlacklistNow(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func normalizeBlacklistRecheck(input *model.UpdateBlacklistRecheckSettingsRequest) (time.Time, error) {
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.CronExpression == "" {
		input.CronExpression = "*/15 * * * *"
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	if input.Fraction <= 0 || input.Fraction > 1 {
		return time.Time{}, fmt.Errorf("fraction must be greater than 0 and no more than 1")
	}
	if input.MaxTargets < 1 || input.MaxTargets > 5000 {
		return time.Time{}, fmt.Errorf("max_targets must be between 1 and 5000")
	}
	if input.Attempts < 1 || input.Attempts > 10 {
		return time.Time{}, fmt.Errorf("attempts must be between 1 and 10")
	}
	if input.TimeoutMS < 500 || input.TimeoutMS > 30000 {
		return time.Time{}, fmt.Errorf("timeout_ms must be between 500 and 30000")
	}
	if input.MaxLatencyMS <= 0 || input.MaxLatencyMS > 60000 {
		return time.Time{}, fmt.Errorf("max_latency_ms must be between 1 and 60000")
	}
	if input.MaxPacketLoss < 0 || input.MaxPacketLoss > 100 {
		return time.Time{}, fmt.Errorf("max_packet_loss must be between 0 and 100")
	}
	if input.RetryMinutes < 1 || input.RetryMinutes > 10080 {
		return time.Time{}, fmt.Errorf("retry_minutes must be between 1 and 10080")
	}
	return scheduling.Next(input.CronExpression, input.Timezone, time.Now().UTC())
}

func (a *API) getBlacklistRecheckSettings(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.GetBlacklistRecheckSettings(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) updateBlacklistRecheckSettings(w http.ResponseWriter, r *http.Request) {
	var input model.UpdateBlacklistRecheckSettingsRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	next, err := normalizeBlacklistRecheck(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_blacklist_recheck", err.Error())
		return
	}
	item, err := a.store.UpdateBlacklistRecheckSettings(r.Context(), input, next)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) runBlacklistRecheck(w http.ResponseWriter, r *http.Request) {
	result, err := a.automation.RunBlacklistNow(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "blacklist_recheck_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func normalizeSourceSync(input *model.UpdateSourceSyncScheduleRequest) (time.Time, error) {
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.CronExpression == "" {
		input.CronExpression = "0 */6 * * *"
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	return scheduling.Next(input.CronExpression, input.Timezone, time.Now().UTC())
}

func validSourceSyncType(source string) bool {
	return source == "official" || source == "asn" || source == "colo"
}

func (a *API) listSourceSyncSchedules(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListSourceSyncSchedules(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) updateSourceSyncSchedule(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")
	if !validSourceSyncType(source) {
		writeError(w, http.StatusBadRequest, "invalid_source", "source must be official, asn, or colo")
		return
	}
	var input model.UpdateSourceSyncScheduleRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	next, err := normalizeSourceSync(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_source_sync", err.Error())
		return
	}
	item, err := a.store.UpdateSourceSyncSchedule(r.Context(), source, input, next)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) runSourceSyncSchedule(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")
	if !validSourceSyncType(source) {
		writeError(w, http.StatusBadRequest, "invalid_source", "source must be official, asn, or colo")
		return
	}
	summary, err := a.automation.RunSourceNow(r.Context(), source)
	if err != nil {
		writeError(w, http.StatusBadGateway, "source_sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"summary": summary})
}

func (a *API) listAutomationRuns(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListAutomationRuns(r.Context(), queryInt(r, "limit", 100))
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) registerAgent(w http.ResponseWriter, r *http.Request) {
	var input model.AgentRegistration
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name, input.Region, input.Continent = strings.TrimSpace(input.Name), strings.TrimSpace(input.Region), strings.TrimSpace(input.Continent)
	if input.Name == "" || input.Region == "" || input.Continent == "" || input.Concurrency <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name, region, continent and positive concurrency are required")
		return
	}
	agent, err := a.store.RegisterAgent(r.Context(), input)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) heartbeat(w http.ResponseWriter, r *http.Request) {
	var input model.AgentHeartbeat
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(input.AgentID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "agent_id is required")
		return
	}
	if err := a.store.Heartbeat(r.Context(), input.AgentID); err != nil {
		writeError(w, http.StatusNotFound, "agent_not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) claimTask(w http.ResponseWriter, r *http.Request) {
	var input model.TaskClaimRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(input.AgentID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "agent_id is required")
		return
	}
	batch, err := a.store.ClaimTasks(r.Context(), input.AgentID, input.Limit)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if batch == nil || len(batch.Tasks) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (a *API) submitResults(w http.ResponseWriter, r *http.Request) {
	var input model.ResultBatch
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.AgentID == "" || input.JobID == "" || len(input.Results) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "agent_id, job_id and results are required")
		return
	}
	if err := a.store.SubmitResults(r.Context(), input); err != nil {
		a.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) requireAgentToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(a.agentToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid agent token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) serverTime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CFScan-Server-Time", strconv.FormatInt(time.Now().UnixMilli(), 10))
		next.ServeHTTP(w, r)
	})
}

func (a *API) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		a.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}

func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "request failed")
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func parseASNParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	value, err := strconv.ParseInt(chi.URLParam(r, "asn"), 10, 64)
	if err != nil || value <= 0 || value > 4294967295 {
		writeError(w, http.StatusBadRequest, "invalid_asn", "ASN must be between 1 and 4294967295")
		return 0, false
	}
	return value, true
}

func queryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func contextWithTimeout(r *http.Request, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), duration)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
