// Package http (modulo platform_identity) implementa los handlers chi
// del modulo. Los handlers son delgados: parsean el request, llaman al
// usecase y traducen los errores tipados a Problem+JSON.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/application/usecases"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

type handlers struct {
	logger            *slog.Logger
	signer            *jwtsign.Signer
	loginUC           *usecases.LoginUseCase
	meUC              *usecases.MeUseCase
	listMembershipsUC *usecases.ListMembershipsUseCase
	switchTenantUC    *usecases.SwitchTenantUseCase
	registerDeviceUC  *usecases.RegisterPushDeviceUseCase
	removeDeviceUC    *usecases.RemovePushDeviceUseCase
}

// authContextKey es la clave para colocar las claims del access token
// validado en el contexto del request hacia el handler.
type authContextKey struct{}

func authClaimsFromCtx(ctx context.Context) (*jwtsign.SessionClaims, bool) {
	c, ok := ctx.Value(authContextKey{}).(*jwtsign.SessionClaims)
	return c, ok
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.loginUC.Execute(r.Context(), req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromCtx(r.Context())
	if !ok || claims == nil || claims.Subject == "" {
		apperrors.Write(w, apperrors.Unauthorized("missing or invalid authentication").WithInstance(r.URL.Path))
		return
	}
	resp, err := h.meUC.Execute(r.Context(), claims.Subject)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) memberships(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromCtx(r.Context())
	if !ok || claims == nil || claims.Subject == "" {
		apperrors.Write(w, apperrors.Unauthorized("missing or invalid authentication").WithInstance(r.URL.Path))
		return
	}
	resp, err := h.listMembershipsUC.Execute(r.Context(), claims.Subject)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) registerPushDevice(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromCtx(r.Context())
	if !ok || claims == nil || claims.Subject == "" {
		apperrors.Write(w, apperrors.Unauthorized("missing or invalid authentication").WithInstance(r.URL.Path))
		return
	}
	var req dto.RegisterPushDeviceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.registerDeviceUC.Execute(r.Context(), claims.Subject, req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *handlers) removePushDevice(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromCtx(r.Context())
	if !ok || claims == nil || claims.Subject == "" {
		apperrors.Write(w, apperrors.Unauthorized("missing or invalid authentication").WithInstance(r.URL.Path))
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	if err := h.removeDeviceUC.Execute(r.Context(), claims.Subject, deviceID); err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) switchTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromCtx(r.Context())
	if !ok || claims == nil || claims.Subject == "" {
		apperrors.Write(w, apperrors.Unauthorized("missing or invalid authentication").WithInstance(r.URL.Path))
		return
	}
	var req dto.SwitchTenantRequest
	if err := decodeJSONBody(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.switchTenantUC.Execute(r.Context(), claims.Subject, req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// authMiddleware valida el JWT del header Authorization y rechaza
// pre-auth tokens (estos solo sirven para /auth/mfa/verify).
func (h *handlers) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := bearerToken(r)
		if tok == "" {
			apperrors.Write(w, apperrors.Unauthorized("missing bearer token").WithInstance(r.URL.Path))
			return
		}
		claims, err := h.signer.Verify(tok)
		if err != nil {
			apperrors.Write(w, apperrors.Unauthorized("invalid or expired token").WithInstance(r.URL.Path))
			return
		}
		if claims.SessionID == usecases.PreAuthSessionMarker {
			apperrors.Write(w, apperrors.Unauthorized("pre-auth token cannot access this endpoint").WithInstance(r.URL.Path))
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	v := strings.TrimSpace(r.Header.Get("Authorization"))
	if v == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(v) <= len(prefix) || !strings.EqualFold(v[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(v[len(prefix):])
}

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid json body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *handlers) writeUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecases.ErrInvalidInput):
		apperrors.Write(w, apperrors.BadRequest("invalid input").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrInvalidCredentials):
		apperrors.Write(w, apperrors.Unauthorized("invalid credentials").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrAccountLocked):
		apperrors.Write(w, apperrors.New(http.StatusLocked, "account-locked", "Account Locked",
			"too many failed attempts; try again later").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrAccountInactive):
		apperrors.Write(w, apperrors.Forbidden("account is not active").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrUserMismatch):
		apperrors.Write(w, apperrors.Unauthorized("user no longer accessible").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrMembershipMissing):
		apperrors.Write(w, apperrors.Forbidden("no membership in tenant").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrInvalidDevice):
		apperrors.Write(w, apperrors.BadRequest("invalid push device").WithInstance(r.URL.Path))
	default:
		if h.logger != nil {
			h.logger.ErrorContext(r.Context(), "platform_identity: internal error",
				slog.String("error", err.Error()),
				slog.String("path", r.URL.Path),
			)
		}
		apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
	}
}
