// Package policies contiene las funciones puras de logica de negocio
// del modulo authorization. NO toca DB ni HTTP.
package policies

import "strings"

// HasPermission devuelve true si el set de namespaces concedidos a un
// actor cubre `required`. Soporta wildcards con sufijo ".*".
//
// Reglas:
//   - Match exacto: si granted contiene exactamente required.
//   - Wildcard explicito: si granted contiene el prefijo `<recurso>.*`
//     (ej. "package.*" cubre "package.deliver", "package.read", etc).
//   - Wildcard global: si granted contiene "*", cubre cualquier permiso
//     (utilidad reservada — no se popula desde la seed por defecto).
//
// Notas de seguridad:
//   - Deny by default: granted vacio implica denegacion.
//   - Required vacio se considera siempre denegado (programacion
//     defensiva: ningun handler debe pedir permiso "").
func HasPermission(granted []string, required string) bool {
	if required == "" {
		return false
	}
	for _, g := range granted {
		if g == "" {
			continue
		}
		if g == "*" {
			return true
		}
		if g == required {
			return true
		}
		if strings.HasSuffix(g, ".*") {
			prefix := strings.TrimSuffix(g, "*") // conserva el punto final
			if strings.HasPrefix(required, prefix) {
				return true
			}
		}
	}
	return false
}

// HasAnyPermission es un helper de conveniencia: true si granted cubre
// AL MENOS uno de los required.
func HasAnyPermission(granted []string, required ...string) bool {
	for _, r := range required {
		if HasPermission(granted, r) {
			return true
		}
	}
	return false
}
