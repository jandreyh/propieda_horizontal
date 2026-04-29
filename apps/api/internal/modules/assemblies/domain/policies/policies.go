// Package policies contiene funciones puras de logica de negocio del
// modulo assemblies.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/saas-ph/api/internal/modules/assemblies/domain/entities"
)

// ---------------------------------------------------------------------------
// Assembly status transitions
// ---------------------------------------------------------------------------

// CanTransitionAssembly indica si una transicion de status de asamblea
// es legal.
//
// Transiciones permitidas:
//   - draft       -> called, archived
//   - called      -> in_progress, quorum_failed, archived
//   - in_progress -> closed, quorum_failed, archived
//   - closed      -> archived
//   - quorum_failed -> archived
//   - archived    -> (terminal)
func CanTransitionAssembly(current, next entities.AssemblyStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.AssemblyStatusDraft:
		return next == entities.AssemblyStatusCalled ||
			next == entities.AssemblyStatusArchived
	case entities.AssemblyStatusCalled:
		return next == entities.AssemblyStatusInProgress ||
			next == entities.AssemblyStatusQuorumFailed ||
			next == entities.AssemblyStatusArchived
	case entities.AssemblyStatusInProgress:
		return next == entities.AssemblyStatusClosed ||
			next == entities.AssemblyStatusQuorumFailed ||
			next == entities.AssemblyStatusArchived
	case entities.AssemblyStatusClosed:
		return next == entities.AssemblyStatusArchived
	case entities.AssemblyStatusQuorumFailed:
		return next == entities.AssemblyStatusArchived
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Motion status transitions
// ---------------------------------------------------------------------------

// CanTransitionMotion indica si una transicion de status de mocion es
// legal.
//
// Transiciones permitidas:
//   - draft     -> open, cancelled, archived
//   - open      -> closed, cancelled, archived
//   - closed    -> archived
//   - cancelled -> archived
//   - archived  -> (terminal)
func CanTransitionMotion(current, next entities.MotionStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.MotionStatusDraft:
		return next == entities.MotionStatusOpen ||
			next == entities.MotionStatusCancelled ||
			next == entities.MotionStatusArchived
	case entities.MotionStatusOpen:
		return next == entities.MotionStatusClosed ||
			next == entities.MotionStatusCancelled ||
			next == entities.MotionStatusArchived
	case entities.MotionStatusClosed:
		return next == entities.MotionStatusArchived
	case entities.MotionStatusCancelled:
		return next == entities.MotionStatusArchived
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Quorum
// ---------------------------------------------------------------------------

// QuorumReached calcula si el quorum fue alcanzado dados los coeficientes
// de las unidades presentes y el porcentaje requerido.
//
// totalCoefficient: suma de coeficientes de unidades presentes con derecho
// a voto.
// quorumRequiredPct: porcentaje requerido (ej. 0.51 = 51%).
func QuorumReached(totalCoefficient, quorumRequiredPct float64) bool {
	return totalCoefficient >= quorumRequiredPct
}

// ---------------------------------------------------------------------------
// Vote hash chain
// ---------------------------------------------------------------------------

// ComputeVoteHash calcula el hash de un voto para la cadena de hashes.
//
// vote_hash = SHA256(prev_vote_hash || motion_id || voter_id || option ||
//
//	timestamp || nonce)
//
// Donde || es concatenacion con separador "|".
func ComputeVoteHash(prevVoteHash, motionID, voterID, option string, castAt time.Time, nonce string) string {
	input := prevVoteHash + "|" +
		motionID + "|" +
		voterID + "|" +
		option + "|" +
		castAt.UTC().Format(time.RFC3339Nano) + "|" +
		nonce
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// ---------------------------------------------------------------------------
// Proxy rules
// ---------------------------------------------------------------------------

// ValidateMaxProxies valida que un apoderado no exceda el maximo de poderes
// permitido por persona en una asamblea.
func ValidateMaxProxies(currentProxyCount, maxProxies int) error {
	if currentProxyCount >= maxProxies {
		return fmt.Errorf("proxy user already has %d proxies (max %d)", currentProxyCount, maxProxies)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Act immutability
// ---------------------------------------------------------------------------

// CanModifyAct verifica que un acta pueda ser modificada.
// Solo las actas en draft son editables.
func CanModifyAct(act entities.Act) error {
	if act.IsSigned() {
		return errors.New("act is signed and immutable")
	}
	return nil
}

// CanSignAct verifica que un acta pueda ser firmada.
// Solo las actas en draft pueden ser firmadas.
func CanSignAct(act entities.Act) error {
	if act.Status != entities.ActStatusDraft {
		return fmt.Errorf("act in status %s cannot be signed", string(act.Status))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Validation helpers (modulo-local, no importa otros dominios)
// ---------------------------------------------------------------------------

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

// CanVoteOnMotion verifica que una mocion esta abierta para recibir votos.
func CanVoteOnMotion(motion entities.AssemblyMotion) error {
	if motion.Status != entities.MotionStatusOpen {
		return fmt.Errorf("motion is not open for voting (status: %s)", string(motion.Status))
	}
	return nil
}

// CanAssemblyBeStarted verifica que la asamblea puede ser iniciada.
func CanAssemblyBeStarted(assembly entities.Assembly) error {
	if assembly.Status != entities.AssemblyStatusCalled {
		return fmt.Errorf("assembly must be in 'called' status to start (current: %s)", string(assembly.Status))
	}
	return nil
}

// CanAssemblyBeClosed verifica que la asamblea puede ser cerrada.
func CanAssemblyBeClosed(assembly entities.Assembly) error {
	if assembly.Status != entities.AssemblyStatusInProgress {
		return fmt.Errorf("assembly must be in 'in_progress' status to close (current: %s)", string(assembly.Status))
	}
	return nil
}
