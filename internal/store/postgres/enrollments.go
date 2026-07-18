package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
	"github.com/3011/cfscan/v2/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const enrollmentColumns = `
id::text, mode,
CASE WHEN status IN ('pending', 'approved') AND expires_at <= NOW() THEN 'expired' ELSE status END,
requested_name, os, architecture, version, requested_concurrency,
name, region, continent, concurrency, COALESCE(agent_id::text, ''),
expires_at, approved_at, claimed_at, created_at, updated_at`

func scanEnrollment(row pgx.Row) (model.AgentEnrollment, error) {
	var item model.AgentEnrollment
	err := row.Scan(
		&item.ID, &item.Mode, &item.Status, &item.RequestedName, &item.OS,
		&item.Architecture, &item.Version, &item.RequestedConcurrency,
		&item.Name, &item.Region, &item.Continent, &item.Concurrency, &item.AgentID,
		&item.ExpiresAt, &item.ApprovedAt, &item.ClaimedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) CreateAgentEnrollment(ctx context.Context, input model.CreateAgentEnrollment) (model.AgentEnrollment, error) {
	query := `INSERT INTO agent_enrollments (
    token_hash, mode, status, requested_name, os, architecture, version,
    requested_concurrency, name, region, continent, concurrency, expires_at, approved_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,CASE WHEN $3='approved' THEN NOW() ELSE NULL END)
RETURNING ` + enrollmentColumns
	item, err := scanEnrollment(s.pool.QueryRow(ctx, query,
		input.TokenHash, input.Mode, input.Status, input.RequestedName, input.OS,
		input.Architecture, input.Version, input.RequestedConcurrency,
		input.Name, input.Region, input.Continent, input.Concurrency, input.ExpiresAt,
	))
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("create agent enrollment: %w", err)
	}
	return item, nil
}

func (s *Store) GetAgentEnrollment(ctx context.Context, tokenHash string) (model.AgentEnrollment, error) {
	item, err := scanEnrollment(s.pool.QueryRow(ctx, `SELECT `+enrollmentColumns+` FROM agent_enrollments WHERE token_hash=$1`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.AgentEnrollment{}, store.ErrNotFound
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("get agent enrollment: %w", err)
	}
	return item, nil
}

func (s *Store) ListAgentEnrollments(ctx context.Context) ([]model.AgentEnrollment, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+enrollmentColumns+` FROM agent_enrollments
WHERE status IN ('pending','approved') AND expires_at > NOW()
ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("list agent enrollments: %w", err)
	}
	defer rows.Close()
	items := make([]model.AgentEnrollment, 0)
	for rows.Next() {
		item, err := scanEnrollment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan agent enrollment: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ApproveAgentEnrollment(ctx context.Context, tokenHash string, input model.ApproveAgentEnrollmentRequest) (model.AgentEnrollment, error) {
	query := `UPDATE agent_enrollments SET status='approved', name=$2, region=$3, continent=$4,
concurrency=$5, approved_at=NOW(), updated_at=NOW()
WHERE token_hash=$1 AND status='pending' AND expires_at > NOW()
RETURNING ` + enrollmentColumns
	item, err := scanEnrollment(s.pool.QueryRow(ctx, query, tokenHash, input.Name, input.Region, input.Continent, input.Concurrency))
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr := s.GetAgentEnrollment(ctx, tokenHash)
		if currentErr != nil {
			return model.AgentEnrollment{}, currentErr
		}
		return model.AgentEnrollment{}, enrollmentStateError(current.Status)
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("approve agent enrollment: %w", err)
	}
	return item, nil
}

func (s *Store) RejectAgentEnrollment(ctx context.Context, tokenHash string) (model.AgentEnrollment, error) {
	query := `UPDATE agent_enrollments SET status='rejected', updated_at=NOW()
WHERE token_hash=$1 AND status='pending' AND expires_at > NOW()
RETURNING ` + enrollmentColumns
	item, err := scanEnrollment(s.pool.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr := s.GetAgentEnrollment(ctx, tokenHash)
		if currentErr != nil {
			return model.AgentEnrollment{}, currentErr
		}
		return model.AgentEnrollment{}, enrollmentStateError(current.Status)
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("reject agent enrollment: %w", err)
	}
	return item, nil
}

func (s *Store) ClaimAgentEnrollment(ctx context.Context, tokenHash, credentialID, credentialHash, osName, architecture, version string) (model.AgentEnrollment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("begin claim enrollment: %w", err)
	}
	defer tx.Rollback(ctx)

	item, err := scanEnrollment(tx.QueryRow(ctx, `SELECT `+enrollmentColumns+` FROM agent_enrollments WHERE token_hash=$1 FOR UPDATE`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.AgentEnrollment{}, store.ErrNotFound
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("load enrollment for claim: %w", err)
	}
	if item.Status == model.AgentEnrollmentClaimed {
		var existingAgentID string
		err := tx.QueryRow(ctx, `SELECT agent_id::text FROM agent_credentials
WHERE id=$1::uuid AND secret_hash=$2 AND revoked_at IS NULL`, credentialID, credentialHash).Scan(&existingAgentID)
		if err == nil && existingAgentID == item.AgentID {
			return item, nil
		}
		return model.AgentEnrollment{}, store.ErrEnrollmentClaimed
	}

	if item.Status == model.AgentEnrollmentExpired || !item.ExpiresAt.After(time.Now()) {
		return model.AgentEnrollment{}, store.ErrEnrollmentExpired
	}
	if item.Status == model.AgentEnrollmentPending {
		return model.AgentEnrollment{}, store.ErrEnrollmentPending
	}
	if item.Status == model.AgentEnrollmentRejected {
		return model.AgentEnrollment{}, store.ErrEnrollmentRejected
	}
	if item.Status == model.AgentEnrollmentRevoked {
		return model.AgentEnrollment{}, store.ErrEnrollmentRevoked
	}
	if item.Status != model.AgentEnrollmentApproved {
		return model.AgentEnrollment{}, enrollmentStateError(item.Status)
	}

	if strings.TrimSpace(osName) != "" {
		item.OS = strings.TrimSpace(osName)
	}
	if strings.TrimSpace(architecture) != "" {
		item.Architecture = strings.TrimSpace(architecture)
	}
	if strings.TrimSpace(version) != "" {
		item.Version = strings.TrimSpace(version)
	}
	var agent model.Agent
	err = tx.QueryRow(ctx, `INSERT INTO agents (
name, region, continent, concurrency, status, last_seen_at, os, architecture, version, updated_at
) VALUES ($1,$2,$3,$4,'online',NOW(),$5,$6,$7,NOW())
RETURNING id::text, name, region, continent, concurrency, status, os, architecture, version, last_seen_at, created_at`,
		item.Name, item.Region, item.Continent, item.Concurrency, item.OS, item.Architecture, item.Version,
	).Scan(&agent.ID, &agent.Name, &agent.Region, &agent.Continent, &agent.Concurrency,
		&agent.Status, &agent.OS, &agent.Architecture, &agent.Version,
		&agent.LastSeenAt, &agent.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.AgentEnrollment{}, store.ErrEnrollmentConflict
		}
		return model.AgentEnrollment{}, fmt.Errorf("create enrolled agent: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO agent_credentials (id, agent_id, secret_hash) VALUES ($1::uuid,$2::uuid,$3)`, credentialID, agent.ID, credentialHash); err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("create agent credential: %w", err)
	}
	item, err = scanEnrollment(tx.QueryRow(ctx, `UPDATE agent_enrollments SET status='claimed', agent_id=$2::uuid,
os=$3, architecture=$4, version=$5, claimed_at=NOW(), updated_at=NOW()
WHERE id=$1::uuid RETURNING `+enrollmentColumns, item.ID, agent.ID, item.OS, item.Architecture, item.Version))
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("finish agent enrollment: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("commit agent enrollment: %w", err)
	}
	return item, nil
}

func (s *Store) AuthenticateAgentCredential(ctx context.Context, credentialID, credentialHash string) (string, error) {
	var agentID string
	err := s.pool.QueryRow(ctx, `UPDATE agent_credentials SET last_used_at=NOW()
WHERE id=$1::uuid AND secret_hash=$2 AND revoked_at IS NULL
RETURNING agent_id::text`, credentialID, credentialHash).Scan(&agentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", store.ErrInvalidAgentCredential
	}
	if err != nil {
		return "", fmt.Errorf("authenticate agent credential: %w", err)
	}
	return agentID, nil
}

func enrollmentStateError(status string) error {
	switch status {
	case model.AgentEnrollmentPending:
		return store.ErrEnrollmentPending
	case model.AgentEnrollmentExpired:
		return store.ErrEnrollmentExpired
	case model.AgentEnrollmentRejected:
		return store.ErrEnrollmentRejected
	case model.AgentEnrollmentRevoked:
		return store.ErrEnrollmentRevoked
	case model.AgentEnrollmentClaimed:
		return store.ErrEnrollmentClaimed
	default:
		return store.ErrEnrollmentConflict
	}
}

func (s *Store) GetAgentEnrollmentByID(ctx context.Context, enrollmentID string) (model.AgentEnrollment, error) {
	item, err := scanEnrollment(s.pool.QueryRow(ctx, `SELECT `+enrollmentColumns+` FROM agent_enrollments WHERE id=$1::uuid`, enrollmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.AgentEnrollment{}, store.ErrNotFound
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("get agent enrollment by id: %w", err)
	}
	return item, nil
}

func (s *Store) ApproveAgentEnrollmentByID(ctx context.Context, enrollmentID string, input model.ApproveAgentEnrollmentRequest) (model.AgentEnrollment, error) {
	query := `UPDATE agent_enrollments SET status='approved', name=$2, region=$3, continent=$4,
concurrency=$5, approved_at=NOW(), updated_at=NOW()
WHERE id=$1::uuid AND status='pending' AND expires_at > NOW()
RETURNING ` + enrollmentColumns
	item, err := scanEnrollment(s.pool.QueryRow(ctx, query, enrollmentID, input.Name, input.Region, input.Continent, input.Concurrency))
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr := s.GetAgentEnrollmentByID(ctx, enrollmentID)
		if currentErr != nil {
			return model.AgentEnrollment{}, currentErr
		}
		return model.AgentEnrollment{}, enrollmentStateError(current.Status)
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("approve agent enrollment by id: %w", err)
	}
	return item, nil
}

func (s *Store) RejectAgentEnrollmentByID(ctx context.Context, enrollmentID string) (model.AgentEnrollment, error) {
	query := `UPDATE agent_enrollments SET status='rejected', updated_at=NOW()
WHERE id=$1::uuid AND status='pending' AND expires_at > NOW()
RETURNING ` + enrollmentColumns
	item, err := scanEnrollment(s.pool.QueryRow(ctx, query, enrollmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr := s.GetAgentEnrollmentByID(ctx, enrollmentID)
		if currentErr != nil {
			return model.AgentEnrollment{}, currentErr
		}
		return model.AgentEnrollment{}, enrollmentStateError(current.Status)
	}
	if err != nil {
		return model.AgentEnrollment{}, fmt.Errorf("reject agent enrollment by id: %w", err)
	}
	return item, nil
}
