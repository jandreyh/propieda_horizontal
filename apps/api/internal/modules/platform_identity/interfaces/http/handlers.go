// Package http (modulo platform_identity) implementa los handlers chi
// del modulo. Los handlers son delgados: parsean el request, llaman al
// usecase y traducen los errores tipados a Problem+JSON.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/application/usecases"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

type handlers struct {
	logger  *slog.Logger
	loginUC *usecases.LoginUseCase
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
