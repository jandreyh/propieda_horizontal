// Package policies contiene funciones puras de logica de negocio del
// modulo tenant_config: validaciones de keys, colores hex, timezones y
// locales aceptados.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones determinist as y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// settingKeyRe valida la forma `^[a-z][a-z0-9_.]*$`. Coincide con el
// CHECK constraint de la tabla `tenant_settings`.
var settingKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_.]*$`)

// hexColorRe acepta `#RRGGBB` y `#RGB` (insensible a mayusculas).
var hexColorRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// localeRe acepta IETF BCP-47 minimal (`xx` o `xx-YY`).
var localeRe = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

// allowedTimezones es la "lista alaska" pedida: zonas validas para Colombia
// (todas mapean a `America/Bogota` UTC-5) mas `UTC`. Aceptamos los nombres
// IANA de Colombia, segun tzdata.
var allowedTimezones = map[string]struct{}{
	"America/Bogota":       {},
	"America/Cali":         {},
	"America/Medellin":     {},
	"America/Barranquilla": {},
	"America/Cartagena":    {},
	"UTC":                  {},
}

// AllowedTimezones devuelve una copia del set permitido (orden no
// garantizado). Util para tests y para serializar en /docs.
func AllowedTimezones() []string {
	out := make([]string, 0, len(allowedTimezones))
	for tz := range allowedTimezones {
		out = append(out, tz)
	}
	return out
}

// ValidateSettingKey verifica que una key cumpla `^[a-z][a-z0-9_.]*$`.
func ValidateSettingKey(key string) error {
	if key == "" {
		return errors.New("setting key is required")
	}
	if len(key) > 128 {
		return fmt.Errorf("setting key too long (max 128, got %d)", len(key))
	}
	if !settingKeyRe.MatchString(key) {
		return fmt.Errorf("invalid setting key %q: must match ^[a-z][a-z0-9_.]*$", key)
	}
	return nil
}

// ValidateSettingValue solo verifica que el JSON crudo no este vacio. La
// validez sintactica del JSON se delega al deserializador del DTO; aqui
// nos limitamos a evitar guardar vacio.
func ValidateSettingValue(raw []byte) error {
	if len(raw) == 0 {
		return errors.New("setting value is required")
	}
	return nil
}

// ValidateHexColor verifica que un color sea `#RGB` o `#RRGGBB`. Acepta
// punteros: nil = no validar (campo opcional).
func ValidateHexColor(c *string) error {
	if c == nil {
		return nil
	}
	if *c == "" {
		return errors.New("hex color must not be empty (use null to omit)")
	}
	if !hexColorRe.MatchString(*c) {
		return fmt.Errorf("invalid hex color %q: expected #RGB or #RRGGBB", *c)
	}
	return nil
}

// ValidateTimezone verifica que la zona este en la lista permitida.
func ValidateTimezone(tz string) error {
	if tz == "" {
		return errors.New("timezone is required")
	}
	if _, ok := allowedTimezones[tz]; !ok {
		return fmt.Errorf("unsupported timezone %q (allowed: Colombia/UTC)", tz)
	}
	return nil
}

// ValidateLocale acepta locales BCP-47 minimal: `xx` o `xx-YY`.
func ValidateLocale(loc string) error {
	if loc == "" {
		return errors.New("locale is required")
	}
	if !localeRe.MatchString(loc) {
		return fmt.Errorf("invalid locale %q: expected BCP-47 like es-CO", loc)
	}
	return nil
}

// ValidateDisplayName valida que el nombre del tenant sea no-vacio y de
// largo razonable.
func ValidateDisplayName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("display_name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("display_name too long (max 200, got %d)", len(name))
	}
	return nil
}
