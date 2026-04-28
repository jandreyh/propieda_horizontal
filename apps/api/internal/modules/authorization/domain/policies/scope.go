package policies

import "github.com/saas-ph/api/internal/modules/authorization/domain/entities"

// MatchesScope decide si un scope concedido (grantedType, grantedID)
// cubre un scope requerido (reqType, reqID).
//
// Reglas (ADR 0003 — Prevalencia):
//   - Si el grant es nil o de tipo "tenant", cubre cualquier scope
//     requerido (ya que "tenant" es la raiz que abarca todo el tenant).
//   - "tower" cubre solo el mismo tower.id requerido.
//   - "unit" cubre solo la misma unit.id requerida.
//   - Si reqType es "" (sin scope explicito), se considera cubierto.
//
// La verificacion cross-tower (ej. tower:T1 cubre unit U si U
// pertenece a T1) se resuelve en el build del cache de permisos
// efectivos, NO aqui (esta funcion es puramente sintactica).
func MatchesScope(grantedType, grantedID, reqType, reqID string) bool {
	if reqType == "" {
		return true
	}
	if grantedType == "" || grantedType == entities.ScopeTenant {
		return true
	}
	if grantedType != reqType {
		return false
	}
	return grantedID == reqID
}
