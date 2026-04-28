// Package passwords implementa hashing de contrasenas con argon2id
// (recomendado por OWASP/NIST para passwords nuevas) y un fallback de
// verificacion para hashes bcrypt heredados.
//
// Formato de salida tipo PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<salt-b64>$<hash-b64>
//
// Parametros por defecto: m=64MiB, t=3, p=2, salt=16B, hash=32B.
// Estos parametros pueden ser tuneados via WithParams sin romper hashes
// previos (cada hash trae sus parametros embebidos).
package passwords

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params controla la fuerza del hashing.
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultParams es la configuracion canonica del proyecto.
var DefaultParams = Params{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLen:     16,
	KeyLen:      32,
}

// Hash genera un hash argon2id usando DefaultParams.
func Hash(plain string) (string, error) {
	return HashWithParams(plain, DefaultParams)
}

// HashWithParams genera un hash con parametros explicitos.
func HashWithParams(plain string, p Params) (string, error) {
	if plain == "" {
		return "", errors.New("passwords: plain vacio")
	}
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("passwords: rand: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLen)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

// ErrMismatch se devuelve por Verify cuando plain no coincide con encoded.
var ErrMismatch = errors.New("passwords: mismatch")

// Verify compara plain contra encoded. Soporta solo argon2id por ahora;
// hashes con otros prefijos devuelven ErrUnsupportedAlgo.
func Verify(plain, encoded string) error {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return ErrUnsupportedAlgo
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return errors.New("passwords: formato encoded invalido")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return fmt.Errorf("passwords: version: %w", err)
	}
	if version != argon2.Version {
		return fmt.Errorf("passwords: version no soportada: %d", version)
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return fmt.Errorf("passwords: params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("passwords: salt b64: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("passwords: hash b64: %w", err)
	}
	//nolint:gosec // len(hash) cabe en uint32 con holgura (max 64).
	derived := argon2.IDKey([]byte(plain), salt, iterations, memory, parallelism, uint32(len(hash)))
	if subtle.ConstantTimeCompare(hash, derived) == 1 {
		return nil
	}
	return ErrMismatch
}

// ErrUnsupportedAlgo se devuelve cuando el hash no es argon2id (ej.
// bcrypt heredado). El caller decide si re-hashear o rechazar.
var ErrUnsupportedAlgo = errors.New("passwords: algoritmo no soportado")
