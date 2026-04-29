// Package policies contiene funciones puras de logica de negocio del
// modulo finance.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
)

// accountCodeRegex valida el formato del codigo de cuenta contable.
// Acepta alfanumericos, puntos y guiones; 1-30 caracteres.
var accountCodeRegex = regexp.MustCompile(`^[A-Za-z0-9.\-]{1,30}$`)

// ValidateAccountCode valida que el codigo de una cuenta contable
// sea no vacio y cumpla el formato esperado.
func ValidateAccountCode(code string) error {
	if code == "" {
		return errors.New("account code is required")
	}
	if !accountCodeRegex.MatchString(code) {
		return fmt.Errorf("account code %q is invalid (must be 1-30 alphanumeric, dots or hyphens)", code)
	}
	return nil
}

// costCenterCodeRegex valida el formato del codigo de centro de costo.
var costCenterCodeRegex = regexp.MustCompile(`^[A-Za-z0-9\-_]{1,20}$`)

// ValidateCostCenterCode valida que el codigo de un centro de costo
// sea no vacio y cumpla el formato esperado.
func ValidateCostCenterCode(code string) error {
	if code == "" {
		return errors.New("cost center code is required")
	}
	if !costCenterCodeRegex.MatchString(code) {
		return fmt.Errorf("cost center code %q is invalid (must be 1-20 alphanumeric, hyphens or underscores)", code)
	}
	return nil
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
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

// ValidatePositiveAmount valida que un monto sea mayor que cero.
func ValidatePositiveAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

// ValidateNonNegativeAmount valida que un monto sea >= 0.
func ValidateNonNegativeAmount(amount float64) error {
	if amount < 0 {
		return errors.New("amount must be non-negative")
	}
	return nil
}

// ValidatePeriod valida que un anio y mes sean razonables.
func ValidatePeriod(year, month int32) error {
	if year < 2000 || year > 2100 {
		return fmt.Errorf("period_year %d is out of range (2000-2100)", year)
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("period_month %d is out of range (1-12)", month)
	}
	return nil
}

// ValidateCurrency valida que el codigo de moneda tenga 3 caracteres.
func ValidateCurrency(currency string) error {
	if len(currency) != 3 {
		return fmt.Errorf("currency code must be 3 characters, got %q", currency)
	}
	return nil
}

// CanAllocatePayment valida que un pago pueda recibir una aplicacion.
//
// Reglas:
//   - El pago debe estar capturado o settled.
//   - El pago debe tener monto sin aplicar >= allocationAmount.
func CanAllocatePayment(payment entities.Payment, allocationAmount float64) error {
	if payment.Status != entities.PaymentStatusCaptured &&
		payment.Status != entities.PaymentStatusSettled {
		return fmt.Errorf("payment must be captured or settled, current status is %s", string(payment.Status))
	}
	if payment.UnallocatedAmount < allocationAmount {
		return fmt.Errorf("insufficient unallocated amount: available %.2f, requested %.2f",
			payment.UnallocatedAmount, allocationAmount)
	}
	return nil
}

// CanAllocateCharge valida que un cargo pueda recibir una aplicacion.
//
// Reglas:
//   - El cargo debe tener saldo pendiente >= allocationAmount.
func CanAllocateCharge(charge entities.Charge, allocationAmount float64) error {
	if !charge.IsPending() {
		return fmt.Errorf("charge is not pending (status=%s)", string(charge.Status))
	}
	if charge.Balance < allocationAmount {
		return fmt.Errorf("insufficient charge balance: available %.2f, requested %.2f",
			charge.Balance, allocationAmount)
	}
	return nil
}

// CanReversePayment valida que un pago pueda recibir un reverso.
//
// Reglas:
//   - El pago debe estar captured o settled (no ya reversed o failed).
func CanReversePayment(payment entities.Payment) error {
	if payment.Status != entities.PaymentStatusCaptured &&
		payment.Status != entities.PaymentStatusSettled {
		return fmt.Errorf("payment cannot be reversed (status=%s)", string(payment.Status))
	}
	return nil
}

// CanClosePeriodSoft valida que un periodo pueda cerrarse soft.
//
// Reglas:
//   - El periodo debe estar en estado 'open'.
func CanClosePeriodSoft(closure entities.PeriodClosure) error {
	if closure.Status != entities.PeriodClosureStatusOpen {
		return fmt.Errorf("period is not open (status=%s)", string(closure.Status))
	}
	return nil
}
