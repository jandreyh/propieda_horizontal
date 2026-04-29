// Package policies contiene funciones puras de logica de negocio del
// modulo notifications.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"time"

	"github.com/saas-ph/api/internal/modules/notifications/domain/entities"
)

// BackoffSchedule define los intervalos de backoff exponencial en
// segundos para reintentos de envio de notificaciones.
// Despues de cada fallo, el siguiente intento se programa segun este
// schedule: 1min, 5min, 15min, 1h, 6h, 24h.
var BackoffSchedule = []int{60, 300, 900, 3600, 21600, 86400}

// MaxAttempts es el numero maximo de intentos antes de marcar como
// failed_permanent.
const MaxAttempts = 6

// DefaultQuietHoursStart es la hora de inicio de las horas de silencio
// (22:00).
const DefaultQuietHoursStart = 22

// DefaultQuietHoursEnd es la hora de fin de las horas de silencio
// (07:00).
const DefaultQuietHoursEnd = 7

// ValidateChannel valida que un canal sea valido.
func ValidateChannel(channel entities.Channel) error {
	if !channel.IsValid() {
		return fmt.Errorf("invalid channel %q (must be email, push, whatsapp or sms)", string(channel))
	}
	return nil
}

// ValidatePlatform valida que una plataforma de push sea valida.
func ValidatePlatform(platform entities.Platform) error {
	if !platform.IsValid() {
		return fmt.Errorf("invalid platform %q (must be ios, android or web)", string(platform))
	}
	return nil
}

// RequiresConsent indica si enviar por un canal requiere consentimiento
// previo. WhatsApp y SMS requieren consentimiento explicito.
func RequiresConsent(channel entities.Channel) bool {
	return channel.RequiresConsent()
}

// IsInQuietHours verifica si una hora dada esta dentro del periodo de
// horas de silencio.
//
// Quiet hours: desde quietStart hasta quietEnd (por defecto 22:00-07:00).
// Si quietStart > quietEnd, el periodo cruza medianoche.
func IsInQuietHours(t time.Time, quietStart, quietEnd int) bool {
	hour := t.Hour()
	if quietStart > quietEnd {
		// Cruza medianoche: 22:00 -> 07:00
		return hour >= quietStart || hour < quietEnd
	}
	return hour >= quietStart && hour < quietEnd
}

// NextBackoffSeconds devuelve el intervalo de backoff para el intento
// dado (0-based). Si el intento excede el schedule, devuelve el ultimo
// valor del schedule.
func NextBackoffSeconds(attempt int32) int {
	if attempt < 0 {
		attempt = 0
	}
	idx := int(attempt)
	if idx >= len(BackoffSchedule) {
		idx = len(BackoffSchedule) - 1
	}
	return BackoffSchedule[idx]
}

// ShouldMarkPermanentFailure indica si un mensaje debe marcarse como
// fallo permanente basado en el numero de intentos.
func ShouldMarkPermanentFailure(attempts int32) bool {
	return attempts >= MaxAttempts
}

// CanTransitionOutbox valida si una transicion de status del outbox es
// legal.
//
// Transiciones permitidas:
//   - pending -> sending, scheduled, blocked_no_consent, cancelled
//   - scheduled -> pending, sending, cancelled
//   - sending -> sent, failed_retry, failed_permanent
//   - failed_retry -> sending, failed_permanent, cancelled
//   - sent -> (terminal)
//   - failed_permanent -> (terminal)
//   - blocked_no_consent -> (terminal)
//   - cancelled -> (terminal)
func CanTransitionOutbox(current, next entities.OutboxStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.OutboxStatusPending:
		return next == entities.OutboxStatusSending ||
			next == entities.OutboxStatusScheduled ||
			next == entities.OutboxStatusBlockedNoConsent ||
			next == entities.OutboxStatusCancelled
	case entities.OutboxStatusScheduled:
		return next == entities.OutboxStatusPending ||
			next == entities.OutboxStatusSending ||
			next == entities.OutboxStatusCancelled
	case entities.OutboxStatusSending:
		return next == entities.OutboxStatusSent ||
			next == entities.OutboxStatusFailedRetry ||
			next == entities.OutboxStatusFailedPermanent
	case entities.OutboxStatusFailedRetry:
		return next == entities.OutboxStatusSending ||
			next == entities.OutboxStatusFailedPermanent ||
			next == entities.OutboxStatusCancelled
	default:
		// sent, failed_permanent, blocked_no_consent, cancelled son
		// terminales.
		return false
	}
}

// ValidateEventType valida que el tipo de evento no este vacio.
func ValidateEventType(eventType string) error {
	if eventType == "" {
		return errors.New("event_type is required")
	}
	return nil
}

// ValidateBodyTemplate valida que el body de una plantilla no este vacio.
func ValidateBodyTemplate(body string) error {
	if body == "" {
		return errors.New("body_template is required")
	}
	return nil
}

// ValidateLocale valida que el locale no este vacio.
func ValidateLocale(locale string) error {
	if locale == "" {
		return errors.New("locale is required")
	}
	return nil
}

// ValidateToken valida que un push token no este vacio.
func ValidateToken(token string) error {
	if token == "" {
		return errors.New("token is required")
	}
	return nil
}

// ValidateIdempotencyKey valida que la clave de idempotencia no este
// vacia.
func ValidateIdempotencyKey(key string) error {
	if key == "" {
		return errors.New("idempotency_key is required")
	}
	return nil
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
//
// Duplicado intencional de la funcion de otros modulos: cada modulo
// posee sus propias policies para no acoplar dominios.
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

// IsCriticalEvent determina si un evento es critico. Los eventos criticos
// ignoran las preferencias del usuario (siempre se envian) pero NO
// ignoran el consentimiento legal.
func IsCriticalEvent(eventType string) bool {
	// Los eventos criticos se definen por convencion con prefijo
	// "critical." o son tipos conocidos del sistema.
	criticalPrefixes := []string{
		"critical.",
		"security.",
		"emergency.",
	}
	for _, prefix := range criticalPrefixes {
		if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
