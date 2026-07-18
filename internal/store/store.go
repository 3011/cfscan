package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
)

type Store interface {
	EnsureBootstrapAdmin(context.Context, string, string, string) error
	FindUserByUsername(context.Context, string) (model.UserCredential, error)
	CreateAuthSession(context.Context, string, string, time.Time) error
	GetUserBySession(context.Context, string) (model.User, error)
	DeleteAuthSession(context.Context, string) error
	ListUsers(context.Context) ([]model.User, error)
	CreateUser(context.Context, model.CreateUserRequest, string) (model.User, error)
	UpdateUser(context.Context, string, model.UpdateUserRequest) (model.User, error)
	ResetUserPassword(context.Context, string, string) error
	DeleteUser(context.Context, string) error

	CreateAgentEnrollment(context.Context, model.CreateAgentEnrollment) (model.AgentEnrollment, error)
	GetAgentEnrollment(context.Context, string) (model.AgentEnrollment, error)
	GetAgentEnrollmentByID(context.Context, string) (model.AgentEnrollment, error)
	ListAgentEnrollments(context.Context) ([]model.AgentEnrollment, error)
	ApproveAgentEnrollment(context.Context, string, model.ApproveAgentEnrollmentRequest) (model.AgentEnrollment, error)
	ApproveAgentEnrollmentByID(context.Context, string, model.ApproveAgentEnrollmentRequest) (model.AgentEnrollment, error)
	RejectAgentEnrollment(context.Context, string) (model.AgentEnrollment, error)
	RejectAgentEnrollmentByID(context.Context, string) (model.AgentEnrollment, error)
	ClaimAgentEnrollment(context.Context, string, string, string, string, string, string) (model.AgentEnrollment, error)
	AuthenticateAgentCredential(context.Context, string, string) (string, error)
	Heartbeat(context.Context, string) error
	ListAgents(context.Context) ([]model.Agent, error)
	Overview(context.Context) (model.Overview, error)

	ReplaceCloudflarePrefixes(context.Context, []model.Prefix) (model.SourceStatus, error)
	RecordSourceError(context.Context, string, error) error
	SourceStatus(context.Context, string) (model.SourceStatus, error)
	ListASNSources(context.Context, bool) ([]model.ASNSource, error)
	CreateASNSource(context.Context, model.CreateASNSourceRequest) (model.ASNSource, error)
	UpdateASNSource(context.Context, int64, model.UpdateASNSourceRequest) (model.ASNSource, error)
	DeleteASNSource(context.Context, int64) error
	ReplaceASNPrefixes(context.Context, int64, []model.Prefix) (model.ASNSource, error)
	RecordASNError(context.Context, int64, error) error
	ListActivePrefixes(context.Context, bool) ([]model.Prefix, error)
	ReplaceColoLocations(context.Context, []model.ColoLocation) (model.ColoSyncSummary, error)
	ListColoLocations(context.Context) ([]model.ColoLocation, error)

	CreateScanJob(context.Context, model.CreateScanJobRequest, []string) (model.ScanJob, error)
	ListScanJobs(context.Context, int) ([]model.ScanJob, error)
	GetScanJob(context.Context, string) (model.ScanJob, error)
	StopScanJob(context.Context, string) (model.ScanJob, error)
	ListScanSchedules(context.Context) ([]model.ScanSchedule, error)
	GetScanSchedule(context.Context, string) (model.ScanSchedule, error)
	CreateScanSchedule(context.Context, model.UpsertScanScheduleRequest, time.Time) (model.ScanSchedule, error)
	UpdateScanSchedule(context.Context, string, model.UpsertScanScheduleRequest, time.Time) (model.ScanSchedule, error)
	DeleteScanSchedule(context.Context, string) error
	ClaimDueScanSchedules(context.Context, time.Time, time.Time, int) ([]model.ScanSchedule, error)
	RecordScanScheduleRun(context.Context, string, *time.Time, *string, error) error
	ClaimTasks(context.Context, string, int) (*model.TaskBatch, error)
	SubmitResults(context.Context, model.ResultBatch) error
	ListResults(context.Context, model.ResultFilter) (model.ResultPage, error)
	ListResultColoFacets(context.Context, model.ResultFilter) ([]model.ResultColoFacet, error)
	ListResultJobFacets(context.Context, model.ResultFilter) ([]model.ResultJobFacet, error)
	ListBlacklist(context.Context, int) ([]model.BlacklistEntry, error)
	GetBlacklistRecheckSettings(context.Context) (model.BlacklistRecheckSettings, error)
	UpdateBlacklistRecheckSettings(context.Context, model.UpdateBlacklistRecheckSettingsRequest, time.Time) (model.BlacklistRecheckSettings, error)
	ClaimDueBlacklistRecheck(context.Context, time.Time, time.Time) (*model.BlacklistRecheckSettings, error)
	RecordBlacklistRecheckRun(context.Context, *time.Time, error) error
	CreateBlacklistRechecks(context.Context, model.BlacklistRecheckSettings) (model.BlacklistRecheckResult, error)

	ListSourceSyncSchedules(context.Context) ([]model.SourceSyncSchedule, error)
	GetSourceSyncSchedule(context.Context, string) (model.SourceSyncSchedule, error)
	UpdateSourceSyncSchedule(context.Context, string, model.UpdateSourceSyncScheduleRequest, time.Time) (model.SourceSyncSchedule, error)
	ClaimDueSourceSyncSchedules(context.Context, time.Time, time.Time, int) ([]model.SourceSyncSchedule, error)
	RecordSourceSyncRun(context.Context, string, *time.Time, error) error

	StartAutomationRun(context.Context, model.StartAutomationRunRequest) (model.AutomationRun, error)
	FinishAutomationRun(context.Context, string, string, json.RawMessage, error) error
	ListAutomationRuns(context.Context, int) ([]model.AutomationRun, error)
}
