// Package policies contiene funciones puras de logica de negocio del
// modulo incidents.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/incidents/domain/entities"
)

// CanTransitionStatus indica si una transicion de status de incidente
// es legal.
//
// Transiciones permitidas:
//   - reported   -> assigned, cancelled
//   - assigned   -> in_progress, cancelled
//   - in_progress -> resolved, cancelled
//   - resolved   -> closed
//   - closed     -> (terminal, no transiciones)
//   - cancelled  -> (terminal, no transiciones)
func CanTransitionStatus(current, next entities.IncidentStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.IncidentStatusReported:
		return next == entities.IncidentStatusAssigned ||
			next == entities.IncidentStatusCancelled
	case entities.IncidentStatusAssigned:
		return next == entities.IncidentStatusInProgress ||
			next == entities.IncidentStatusCancelled
	case entities.IncidentStatusInProgress:
		return next == entities.IncidentStatusResolved ||
			next == entities.IncidentStatusCancelled
	case entities.IncidentStatusResolved:
		return next == entities.IncidentStatusClosed
	default:
		// closed, cancelled son terminales.
		return false
	}
}

// ValidateResolutionNotes valida que las notas de resolucion no esten
// vacias cuando el status destino es resolved o closed.
func ValidateResolutionNotes(targetStatus entities.IncidentStatus, notes *string) error {
	if targetStatus == entities.IncidentStatusResolved ||
		targetStatus == entities.IncidentStatusClosed {
		if notes == nil || strings.TrimSpace(*notes) == "" {
			return errors.New("resolution_notes is required for resolved/closed status")
		}
	}
	return nil
}

// SLADurations define los limites de SLA por severidad (V1 hardcoded).
//
// Formato: (assign_hours, resolve_hours).
type SLADurations struct {
	AssignHours  int
	ResolveHours int
}

// SLABySeverity devuelve las duraciones SLA para una severidad dada.
//
// Valores V1:
//   - critical: 1h asignacion / 4h resolucion
//   - high:     4h asignacion / 24h resolucion
//   - medium:   24h asignacion / 72h resolucion
//   - low:      72h asignacion / 168h resolucion
func SLABySeverity(severity entities.Severity) SLADurations {
	switch severity {
	case entities.SeverityCritical:
		return SLADurations{AssignHours: 1, ResolveHours: 4}
	case entities.SeverityHigh:
		return SLADurations{AssignHours: 4, ResolveHours: 24}
	case entities.SeverityMedium:
		return SLADurations{AssignHours: 24, ResolveHours: 72}
	case entities.SeverityLow:
		return SLADurations{AssignHours: 72, ResolveHours: 168}
	default:
		// Fallback a low.
		return SLADurations{AssignHours: 72, ResolveHours: 168}
	}
}

// CalculateSLADueDates calcula las fechas de vencimiento de SLA a partir
// del momento del reporte y la severidad.
func CalculateSLADueDates(reportedAt time.Time, severity entities.Severity) (assignDue, resolveDue time.Time) {
	sla := SLABySeverity(severity)
	assignDue = reportedAt.Add(time.Duration(sla.AssignHours) * time.Hour)
	resolveDue = reportedAt.Add(time.Duration(sla.ResolveHours) * time.Hour)
	return assignDue, resolveDue
}

// IsSLABreached determina si el SLA de un incidente ha sido violado en
// el momento dado.
func IsSLABreached(incident entities.Incident, now time.Time) bool {
	// Si ya esta escalado, no cambiar.
	if incident.Escalated {
		return true
	}
	// Verificar SLA de asignacion: si aun no asignado y vencio.
	if incident.Status == entities.IncidentStatusReported &&
		incident.SLAAssignDueAt != nil &&
		now.After(*incident.SLAAssignDueAt) {
		return true
	}
	// Verificar SLA de resolucion: si no resuelto y vencio.
	if incident.Status != entities.IncidentStatusResolved &&
		incident.Status != entities.IncidentStatusClosed &&
		incident.Status != entities.IncidentStatusCancelled &&
		incident.SLAResolveDueAt != nil &&
		now.After(*incident.SLAResolveDueAt) {
		return true
	}
	return false
}

// MaxAttachmentsPerIncident es el limite maximo de adjuntos por incidente.
const MaxAttachmentsPerIncident = 10

// ValidateAttachmentCount valida que no se exceda el limite de adjuntos.
func ValidateAttachmentCount(current int) error {
	if current >= MaxAttachmentsPerIncident {
		return fmt.Errorf("incident has reached the maximum of %d attachments", MaxAttachmentsPerIncident)
	}
	return nil
}

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
