package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/3011/cfscan/internal/model"
)

func (s *Store) EnsureBootstrapAdmin(ctx context.Context, username, displayName, passwordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bootstrap admin: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(687326091)`); err != nil {
		return fmt.Errorf("lock bootstrap admin: %w", err)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count == 0 {
		if _, err := tx.Exec(ctx, `
INSERT INTO app_users (username, display_name, password_hash, role, enabled)
VALUES ($1, $2, $3, 'admin', TRUE)`, strings.ToLower(username), displayName, passwordHash); err != nil {
			return fmt.Errorf("create bootstrap admin: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) FindUserByUsername(ctx context.Context, username string) (model.UserCredential, error) {
	var item model.UserCredential
	err := s.pool.QueryRow(ctx, `
SELECT id::text, username, display_name, role, enabled, last_login_at, created_at, updated_at, password_hash
FROM app_users WHERE LOWER(username) = LOWER($1)`, username).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role, &item.Enabled,
		&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt, &item.PasswordHash,
	)
	if err != nil {
		return model.UserCredential{}, fmt.Errorf("find user by username: %w", err)
	}
	return item, nil
}

func (s *Store) CreateAuthSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin auth session: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM auth_sessions WHERE expires_at <= NOW()`); err != nil {
		return fmt.Errorf("clean expired sessions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO auth_sessions (token_hash, user_id, expires_at)
VALUES ($1, $2::uuid, $3)`, tokenHash, userID, expiresAt); err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE app_users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1::uuid`, userID); err != nil {
		return fmt.Errorf("record last login: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) GetUserBySession(ctx context.Context, tokenHash string) (model.User, error) {
	var item model.User
	err := s.pool.QueryRow(ctx, `
SELECT u.id::text, u.username, u.display_name, u.role, u.enabled, u.last_login_at, u.created_at, u.updated_at
FROM auth_sessions sess
JOIN app_users u ON u.id = sess.user_id
WHERE sess.token_hash = $1 AND sess.expires_at > NOW() AND u.enabled`, tokenHash).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role, &item.Enabled,
		&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("get user by session: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = NOW() WHERE token_hash = $1 AND last_seen_at < NOW() - INTERVAL '5 minutes'`, tokenHash)
	return item, nil
}

func (s *Store) DeleteAuthSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete auth session: %w", err)
	}
	return nil
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id::text, username, display_name, role, enabled, last_login_at, created_at, updated_at
FROM app_users ORDER BY role, username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	items := make([]model.User, 0)
	for rows.Next() {
		var item model.User
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Role, &item.Enabled,
			&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, input model.CreateUserRequest, passwordHash string) (model.User, error) {
	var item model.User
	err := s.pool.QueryRow(ctx, `
INSERT INTO app_users (username, display_name, password_hash, role, enabled)
VALUES (LOWER($1), $2, $3, $4, TRUE)
RETURNING id::text, username, display_name, role, enabled, last_login_at, created_at, updated_at`,
		input.Username, input.DisplayName, passwordHash, input.Role).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role, &item.Enabled,
		&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateUser(ctx context.Context, userID string, input model.UpdateUserRequest) (model.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.User{}, fmt.Errorf("begin update user: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(687326092)`); err != nil {
		return model.User{}, fmt.Errorf("lock administrator update: %w", err)
	}
	var currentRole string
	var currentEnabled bool
	if err := tx.QueryRow(ctx, `SELECT role, enabled FROM app_users WHERE id = $1::uuid FOR UPDATE`, userID).Scan(&currentRole, &currentEnabled); err != nil {
		return model.User{}, fmt.Errorf("load user for update: %w", err)
	}
	if currentRole == model.RoleAdmin && currentEnabled && (input.Role != model.RoleAdmin || !input.Enabled) {
		var admins int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM app_users WHERE role = 'admin' AND enabled`).Scan(&admins); err != nil {
			return model.User{}, fmt.Errorf("count administrators: %w", err)
		}
		if admins <= 1 {
			return model.User{}, errors.New("cannot disable or demote the last enabled administrator")
		}
	}
	var item model.User
	if err := tx.QueryRow(ctx, `
UPDATE app_users SET display_name = $2, role = $3, enabled = $4, updated_at = NOW()
WHERE id = $1::uuid
RETURNING id::text, username, display_name, role, enabled, last_login_at, created_at, updated_at`,
		userID, input.DisplayName, input.Role, input.Enabled).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role, &item.Enabled,
		&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return model.User{}, fmt.Errorf("update user: %w", err)
	}
	if !input.Enabled {
		if _, err := tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id = $1::uuid`, userID); err != nil {
			return model.User{}, fmt.Errorf("revoke disabled user sessions: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return model.User{}, fmt.Errorf("commit update user: %w", err)
	}
	return item, nil
}

func (s *Store) ResetUserPassword(ctx context.Context, userID, passwordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reset password: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE app_users SET password_hash = $2, updated_at = NOW() WHERE id = $1::uuid`, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("reset user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id = $1::uuid`, userID); err != nil {
		return fmt.Errorf("revoke password-reset sessions: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete user: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(687326092)`); err != nil {
		return fmt.Errorf("lock administrator deletion: %w", err)
	}
	var role string
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT role, enabled FROM app_users WHERE id = $1::uuid FOR UPDATE`, userID).Scan(&role, &enabled); err != nil {
		return fmt.Errorf("load user for deletion: %w", err)
	}
	if role == model.RoleAdmin && enabled {
		var admins int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM app_users WHERE role = 'admin' AND enabled`).Scan(&admins); err != nil {
			return fmt.Errorf("count administrators: %w", err)
		}
		if admins <= 1 {
			return errors.New("cannot delete the last enabled administrator")
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app_users WHERE id = $1::uuid`, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return tx.Commit(ctx)
}
