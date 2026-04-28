// Package jwtsign emite y valida JWTs cortos para sesiones operativas.
//
// Algoritmo: EdDSA (Ed25519) por simplicidad y seguridad. Las claves se
// cargan en hex desde env vars (PEM tambien soportado).
//
// Claims estandar: `iss`, `sub`, `aud`, `iat`, `nbf`, `exp`, `jti`.
// Claims custom: `tid` (tenant id), `sid` (session id), `roles`, `amr`.
//
// El token se valida con tolerancia de skew configurable (default 30s).
package jwtsign

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Signer emite y verifica tokens. Inmutable tras construccion.
type Signer struct {
	priv      ed25519.PrivateKey
	pub       ed25519.PublicKey
	issuer    string
	audience  string
	keyID     string
	clockSkew time.Duration
}

// SignerConfig agrupa parametros del Signer.
type SignerConfig struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Issuer     string
	Audience   string
	KeyID      string
	ClockSkew  time.Duration
}

// NewSigner construye un Signer. Si Public/Private son nil, genera un
// par efimero (util para tests, NO para produccion).
func NewSigner(cfg SignerConfig) (*Signer, error) {
	priv, pub := cfg.PrivateKey, cfg.PublicKey
	if priv == nil || pub == nil {
		var err error
		pub, priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("jwtsign: generate key: %w", err)
		}
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "ph-saas"
	}
	if cfg.Audience == "" {
		cfg.Audience = "ph-tenant"
	}
	if cfg.KeyID == "" {
		cfg.KeyID = "default"
	}
	if cfg.ClockSkew <= 0 {
		cfg.ClockSkew = 30 * time.Second
	}
	return &Signer{
		priv:      priv,
		pub:       pub,
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
		keyID:     cfg.KeyID,
		clockSkew: cfg.ClockSkew,
	}, nil
}

// SessionClaims es el payload estandar del JWT operativo del proyecto.
type SessionClaims struct {
	TenantID  string   `json:"tid"`
	SessionID string   `json:"sid"`
	Roles     []string `json:"roles,omitempty"`
	AMR       []string `json:"amr,omitempty"`
	jwt.RegisteredClaims
}

// Sign emite un token con expiracion `ttl`. `subject` es el user id.
func (s *Signer) Sign(subject, tenantID, sessionID string, roles, amr []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := SessionClaims{
		TenantID:  tenantID,
		SessionID: sessionID,
		Roles:     roles,
		AMR:       amr,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        randID(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = s.keyID
	signed, err := token.SignedString(s.priv)
	if err != nil {
		return "", fmt.Errorf("jwtsign: sign: %w", err)
	}
	return signed, nil
}

// Verify decodifica y valida un token. Devuelve los claims si valido.
func (s *Signer) Verify(tokenStr string) (*SessionClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithLeeway(s.clockSkew),
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
		jwt.WithExpirationRequired(),
	)
	claims := &SessionClaims{}
	_, err := parser.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
		return s.pub, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwtsign: verify: %w", err)
	}
	if claims.SessionID == "" {
		return nil, errors.New("jwtsign: claim sid faltante")
	}
	return claims, nil
}

func randID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, x := range b {
		out[i*2] = hexChars[x>>4]
		out[i*2+1] = hexChars[x&0x0f]
	}
	return string(out)
}
