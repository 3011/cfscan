package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
	"github.com/3011/cfscan/v2/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

const defaultCookieName = "cfscan_session"

type Config struct {
	SessionTTL   time.Duration
	CookieName   string
	CookieSecure bool
}

type Service struct {
	store        store.Store
	sessionTTL   time.Duration
	cookieName   string
	cookieSecure bool
	dummyHash    []byte
}

func New(dataStore store.Store, cfg Config) *Service {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}
	if strings.TrimSpace(cfg.CookieName) == "" {
		cfg.CookieName = defaultCookieName
	}
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("invalid-login-password"), bcrypt.DefaultCost)
	return &Service{store: dataStore, sessionTTL: cfg.SessionTTL, cookieName: cfg.CookieName, cookieSecure: cfg.CookieSecure, dummyHash: dummyHash}
}

func (s *Service) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users before bootstrap: %w", err)
	}
	if len(users) > 0 {
		return nil
	}
	username = NormalizeUsername(username)
	if username == "" || password == "" {
		return errors.New("bootstrap administrator username and password are required when no users exist")
	}
	if err := ValidateUsername(username); err != nil {
		return fmt.Errorf("bootstrap administrator: %w", err)
	}
	if err := ValidatePassword(password); err != nil {
		return fmt.Errorf("bootstrap administrator: %w", err)
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.store.EnsureBootstrapAdmin(ctx, username, "Administrator", hash)
}

func (s *Service) Login(ctx context.Context, username, password string) (model.User, string, time.Time, error) {
	credential, err := s.store.FindUserByUsername(ctx, NormalizeUsername(username))
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return model.User{}, "", time.Time{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(password)) != nil || !credential.Enabled {
		return model.User{}, "", time.Time{}, ErrInvalidCredentials
	}
	token, err := randomToken()
	if err != nil {
		return model.User{}, "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	if err := s.store.CreateAuthSession(ctx, hashToken(token), credential.ID, expiresAt); err != nil {
		return model.User{}, "", time.Time{}, err
	}
	credential.PasswordHash = ""
	credential.User.LastLoginAt = pointerTime(time.Now().UTC())
	return credential.User, token, expiresAt, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (model.User, error) {
	if token == "" {
		return model.User{}, ErrInvalidCredentials
	}
	return s.store.GetUserBySession(ctx, hashToken(token))
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteAuthSession(ctx, hashToken(token))
}

func (s *Service) TokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cookieName, Value: token, Path: "/", HttpOnly: true, Secure: s.cookieSecure,
		SameSite: http.SameSiteLaxMode, Expires: expiresAt, MaxAge: max(int(time.Until(expiresAt).Seconds()), 1),
	})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.cookieSecure,
		SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0), MaxAge: -1,
	})
}

func NormalizeUsername(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func ValidateUsername(value string) error {
	value = NormalizeUsername(value)
	if len(value) < 3 || len(value) > 64 {
		return errors.New("username must contain 3 to 64 characters")
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return errors.New("username may only contain lowercase letters, numbers, dot, underscore, and hyphen")
	}
	return nil
}

func ValidatePassword(value string) error {
	if len(value) < 8 || len(value) > 128 {
		return errors.New("password must contain 8 to 128 characters")
	}
	return nil
}

func ValidateRole(role string) error {
	if role != model.RoleAdmin && role != model.RoleViewer {
		return errors.New("role must be admin or viewer")
	}
	return nil
}

func HashPassword(value string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func pointerTime(value time.Time) *time.Time { return &value }
