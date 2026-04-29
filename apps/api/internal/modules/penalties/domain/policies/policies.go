// Package policies contiene funciones puras de logica de negocio del
// modulo penalties.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"math"

	"github.com/saas-ph/api/internal/modules/penalties/domain/entities"
)

// ---------------------------------------------------------------------------
// Penalty status transitions
// ---------------------------------------------------------------------------

// CanTransitionPenalty indica si una transicion de status de sancion
// es legal.
//
// Transiciones permitidas:
//   - drafted   -> notified, cancelled
//   - notified  -> in_appeal, confirmed
//   - in_appeal -> confirmed, dismissed
//   - confirmed -> settled
//   - settled   -> (terminal)
//   - dismissed -> (terminal)
//   - cancelled -> (terminal)
func CanTransitionPenalty(current, next entities.PenaltyStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.PenaltyStatusDrafted:
		return next == entities.PenaltyStatusNotified ||
			next == entities.PenaltyStatusCancelled
	case entities.PenaltyStatusNotified:
		return next == entities.PenaltyStatusInAppeal ||
			next == entities.PenaltyStatusConfirmed
	case entities.PenaltyStatusInAppeal:
		return next == entities.PenaltyStatusConfirmed ||
			next == entities.PenaltyStatusDismissed
	case entities.PenaltyStatusConfirmed:
		return next == entities.PenaltyStatusSettled
	default:
		// settled, dismissed, cancelled son terminales.
		return false
	}
}

// ---------------------------------------------------------------------------
// Reincidence calculation
// ---------------------------------------------------------------------------

// CalculateReincidenceAmount calcula el monto de una sancion teniendo
// en cuenta la reincidencia.
//
// Formula:
//
//	amount = baseAmount * multiplier^priorCount
//	cap    = baseAmount * capMultiplier
//	result = min(amount, cap)
//
// priorCount = numero de sanciones confirmed/settled para el mismo
// (debtor, catalog) en los ultimos 365 dias.
func CalculateReincidenceAmount(baseAmount, multiplier, capMultiplier float64, priorCount int) float64 {
	if priorCount <= 0 {
		return baseAmount
	}
	amount := baseAmount * math.Pow(multiplier, float64(priorCount))
	cap := baseAmount * capMultiplier
	if amount > cap {
		amount = cap
	}
	// Redondear a 2 decimales.
	return math.Round(amount*100) / 100
}

// ---------------------------------------------------------------------------
// Council approval
// ---------------------------------------------------------------------------

// RequiresCouncilApproval indica si una sancion requiere aprobacion del
// consejo basandose en el monto y el umbral del catalogo.
func RequiresCouncilApproval(amount float64, threshold *float64) bool {
	if threshold == nil {
		return false
	}
	return amount >= *threshold
}

// ---------------------------------------------------------------------------
// Appeal validation
// ---------------------------------------------------------------------------

// CanAppeal indica si una sancion puede recibir una apelacion.
// Solo se puede apelar si el penalty esta en estado 'notified'.
func CanAppeal(penaltyStatus entities.PenaltyStatus) error {
	if penaltyStatus != entities.PenaltyStatusNotified {
		return fmt.Errorf("cannot appeal penalty in status %q; must be notified", penaltyStatus)
	}
	return nil
}

// CanResolveAppeal indica si una apelacion puede resolverse.
func CanResolveAppeal(appealStatus entities.AppealStatus) error {
	if appealStatus != entities.AppealStatusSubmitted && appealStatus != entities.AppealStatusUnderReview {
		return fmt.Errorf("cannot resolve appeal in status %q", appealStatus)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UUID validation
// ---------------------------------------------------------------------------

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
//
// Es un duplicado intencional de la funcion de otros modulos: cada
// modulo posee sus propias policies para no acoplar dominios.
func ValidateUUID(id string) error {
	if len(id) != 36 {
		return fmt.Errorf("invalid uuid length (expected 36, got %d)", len(id))
	}
	for i, c := range id {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return errors.New("invalid uuid format")
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return errors.New("invalid uuid format")
			}
		}
	}
	return nil
}
