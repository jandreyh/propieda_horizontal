// Package errors implementa errores HTTP segun RFC 7807 (Problem Details
// for HTTP APIs). Todos los handlers HTTP del proyecto deben emitir
// respuestas de error usando los helpers de este paquete para garantizar
// content-type `application/problem+json` y forma consistente.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Problem es el cuerpo JSON definido por RFC 7807.
//
// Solo los campos `type`, `title`, `status`, `detail`, `instance` son
// estandar. Cualquier campo adicional se serializa via Extensions.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Extras   map[string]any `json:"-"`
}

// MarshalJSON serializa Problem aplanando Extras al objeto raiz.
func (p Problem) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, 5+len(p.Extras))
	out["type"] = p.Type
	out["title"] = p.Title
	out["status"] = p.Status
	if p.Detail != "" {
		out["detail"] = p.Detail
	}
	if p.Instance != "" {
		out["instance"] = p.Instance
	}
	for k, v := range p.Extras {
		switch k {
		case "type", "title", "status", "detail", "instance":
			// Reservados — no permitir override.
			continue
		default:
			out[k] = v
		}
	}
	return json.Marshal(out)
}

// Error implementa el tipo error de Go para que Problem pueda viajar como
// cualquier otro error.
func (p Problem) Error() string {
	return fmt.Sprintf("%d %s: %s", p.Status, p.Title, p.Detail)
}

// New construye un Problem con type relativo (`urn:ph:problem:<slug>`)
// cuando no se da uno explicito. Si type esta vacio se inserta
// `about:blank` segun la RFC.
func New(status int, slug, title, detail string) Problem {
	t := slug
	if t == "" {
		t = "about:blank"
	} else if !isAbsoluteURI(t) {
		t = "urn:ph:problem:" + slug
	}
	return Problem{Type: t, Title: title, Status: status, Detail: detail}
}

// WithExtras devuelve un Problem con campos extra (para errores de
// validacion: lista de campos invalidos, etc.).
func (p Problem) WithExtras(extras map[string]any) Problem {
	p.Extras = extras
	return p
}

// WithInstance fija el campo `instance` (ruta de la peticion que disparo
// el error). Tipicamente `r.URL.Path`.
func (p Problem) WithInstance(instance string) Problem {
	p.Instance = instance
	return p
}

// Write serializa un Problem en w con el content-type adecuado.
//
// Si w no soporta WriteHeader (por estar ya escrito), el escritor falla
// silenciosamente. Esto es aceptable porque el caller ya esta en un path
// de error.
func Write(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// Helpers comunes.

// BadRequest crea un Problem 400.
func BadRequest(detail string) Problem {
	return New(http.StatusBadRequest, "bad-request", "Bad Request", detail)
}

// Unauthorized crea un Problem 401.
func Unauthorized(detail string) Problem {
	return New(http.StatusUnauthorized, "unauthorized", "Unauthorized", detail)
}

// Forbidden crea un Problem 403.
func Forbidden(detail string) Problem {
	return New(http.StatusForbidden, "forbidden", "Forbidden", detail)
}

// NotFound crea un Problem 404.
func NotFound(detail string) Problem {
	return New(http.StatusNotFound, "not-found", "Not Found", detail)
}

// Conflict crea un Problem 409.
func Conflict(detail string) Problem {
	return New(http.StatusConflict, "conflict", "Conflict", detail)
}

// TooManyRequests crea un Problem 429.
func TooManyRequests(detail string) Problem {
	return New(http.StatusTooManyRequests, "too-many-requests", "Too Many Requests", detail)
}

// Internal crea un Problem 500. El detail debe ser GENERICO en respuesta;
// el detalle real va al log con el request_id.
func Internal(detail string) Problem {
	if detail == "" {
		detail = "internal server error"
	}
	return New(http.StatusInternalServerError, "internal", "Internal Server Error", detail)
}

// AsProblem extrae un Problem de un error encadenado. Devuelve un 500
// generico si no encuentra ninguno.
func AsProblem(err error) Problem {
	var p Problem
	if errors.As(err, &p) {
		return p
	}
	return Internal("")
}

func isAbsoluteURI(s string) bool {
	for _, prefix := range []string{"http://", "https://", "urn:", "tag:", "about:"} {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
