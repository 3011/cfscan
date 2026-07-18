package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	authservice "github.com/3011/cfscan/v2/internal/auth"
	"github.com/3011/cfscan/v2/internal/model"
)

type currentUserKey struct{}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	clientKey := loginClientKey(r)
	if allowed, retry := a.loginLimiter.Allow(clientKey); !allowed {
		seconds := max(int(retry.Seconds()), 1)
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		writeError(w, http.StatusTooManyRequests, "too_many_login_attempts", "登录尝试过多，请稍后再试")
		return
	}
	var input model.LoginRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, token, expiresAt, err := a.auth.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			a.loginLimiter.Failure(clientKey)
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
			return
		}
		a.internalError(w, r, err)
		return
	}
	a.loginLimiter.Success(clientKey)
	a.auth.SetSessionCookie(w, token, expiresAt)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, user)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, currentUser(r.Context()))
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.Logout(r.Context(), a.auth.TokenFromRequest(r)); err != nil {
		a.internalError(w, r, err)
		return
	}
	a.auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.auth.Authenticate(r.Context(), a.auth.TokenFromRequest(r))
		if err != nil {
			a.auth.ClearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "authentication_required", "请先登录")
			return
		}
		ctx := context.WithValue(r.Context(), currentUserKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r.Context()).Role != model.RoleAdmin {
			writeError(w, http.StatusForbidden, "permission_denied", "当前账号只有查看权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(ctx context.Context) model.User {
	user, _ := ctx.Value(currentUserKey{}).(model.User)
	return user
}

func loginClientKey(r *http.Request) string {
	value := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	if value == "" {
		return "unknown"
	}
	return value
}
