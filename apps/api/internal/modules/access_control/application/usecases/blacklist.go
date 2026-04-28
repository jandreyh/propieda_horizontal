package usecases

import (
	"context"
	"errors"
	"strings"

	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	"github.com/saas-ph/api/internal/modules/access_control/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// CreateBlacklistEntry crea una nueva entrada en la blacklist.
type CreateBlacklistEntry struct {
	Repo domain.BlacklistRepository
}

// CreateBlacklistInput es el input del usecase.
type CreateBlacklistInput = domain.CreateBlacklistInput

// Execute valida y delega.
func (u CreateBlacklistEntry) Execute(ctx context.Context, in CreateBlacklistInput) (entities.BlacklistEntry, error) {
	if err := policies.ValidateDocumentType(string(in.DocumentType)); err != nil {
		return entities.BlacklistEntry{}, apperrors.BadRequest("document_type: " + err.Error())
	}
	if err := policies.ValidateDocumentNumber(in.DocumentNumber); err != nil {
		return entities.BlacklistEntry{}, apperrors.BadRequest("document_number: " + err.Error())
	}
	if strings.TrimSpace(in.Reason) == "" {
		return entities.BlacklistEntry{}, apperrors.BadRequest("reason is required")
	}
	if in.ReportedByUnitID != nil && *in.ReportedByUnitID != "" {
		if err := policies.ValidateUUID(*in.ReportedByUnitID); err != nil {
			return entities.BlacklistEntry{}, apperrors.BadRequest("reported_by_unit_id: " + err.Error())
		}
	}
	entry, err := u.Repo.Create(ctx, in)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBlacklistAlreadyExists):
			return entities.BlacklistEntry{}, apperrors.Conflict("blacklist entry already exists for this document")
		default:
			return entities.BlacklistEntry{}, apperrors.Internal("failed to create blacklist entry")
		}
	}
	return entry, nil
}

// ListBlacklist lista las entradas activas de blacklist.
type ListBlacklist struct {
	Repo domain.BlacklistRepository
}

// Execute delega al repo.
func (u ListBlacklist) Execute(ctx context.Context) ([]entities.BlacklistEntry, error) {
	out, err := u.Repo.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list blacklist")
	}
	return out, nil
}
