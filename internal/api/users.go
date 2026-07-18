package api

import (
	"net/http"
	"strings"

	authservice "github.com/3011/cfscan/v2/internal/auth"
	"github.com/3011/cfscan/v2/internal/model"
	"github.com/go-chi/chi/v5"
)

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListUsers(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var input model.CreateUserRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Username = authservice.NormalizeUsername(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if err := authservice.ValidateUsername(input.Username); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_username", err.Error())
		return
	}
	if err := authservice.ValidatePassword(input.Password); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}
	if err := authservice.ValidateRole(input.Role); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_role", err.Error())
		return
	}
	hash, err := authservice.HashPassword(input.Password)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	item, err := a.store.CreateUser(r.Context(), input, hash)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_user_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var input model.UpdateUserRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "invalid_display_name", "显示名称不能为空")
		return
	}
	if err := authservice.ValidateRole(input.Role); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_role", err.Error())
		return
	}
	actor := currentUser(r.Context())
	if actor.ID == userID && (!input.Enabled || input.Role != actor.Role) {
		writeError(w, http.StatusBadRequest, "cannot_change_self_access", "不能停用或修改自己的管理员权限")
		return
	}
	item, err := a.store.UpdateUser(r.Context(), userID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_user_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var input model.ResetUserPasswordRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := authservice.ValidatePassword(input.Password); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}
	hash, err := authservice.HashPassword(input.Password)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if err := a.store.ResetUserPassword(r.Context(), userID, hash); err != nil {
		writeError(w, http.StatusBadRequest, "reset_password_failed", err.Error())
		return
	}
	if currentUser(r.Context()).ID == userID {
		a.auth.ClearSessionCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if currentUser(r.Context()).ID == userID {
		writeError(w, http.StatusBadRequest, "cannot_delete_self", "不能删除当前登录账号")
		return
	}
	if err := a.store.DeleteUser(r.Context(), userID); err != nil {
		writeError(w, http.StatusBadRequest, "delete_user_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
