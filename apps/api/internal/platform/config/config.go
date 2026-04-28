// Package config carga la configuracion de la aplicacion desde variables
// de entorno con valores por defecto para desarrollo local.
//
// Estilo: structs simples, sin reflection ni ciclos magicos. Toda variable
// nueva se documenta en el README y aparece aqui explicita.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config agrupa toda la configuracion del proceso `api`.
type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
	Tenant   TenantConfig
	Log      LogConfig
}

// HTTPConfig contiene el listener HTTP.
type HTTPConfig struct {
	Addr            string        // ":8080" por defecto
	ReadTimeout     time.Duration // 10s
	WriteTimeout    time.Duration // 30s
	IdleTimeout     time.Duration // 120s
	ShutdownTimeout time.Duration // 15s
}

// DatabaseConfig agrupa las URLs de los pools.
type DatabaseConfig struct {
	// CentralURL apunta al Control Plane (Postgres).
	CentralURL string
	// TenantTemplateURL es la URL de la base "template" usada para clonar
	// tenants en desarrollo. En produccion cada tenant tiene su URL en
	// el Control Plane.
	TenantTemplateURL string
	// MaxConns por pool.
	MaxConns int32
	// MinConns por pool.
	MinConns int32
	// MaxConnLifetime para reciclaje.
	MaxConnLifetime time.Duration
}

// TenantConfig controla la resolucion y cache de tenants.
type TenantConfig struct {
	// BaseDomain es el dominio raiz; los subdominios `<slug>.<base>` se
	// resuelven como tenant. Ej. "ph.app" -> "acacias.ph.app".
	BaseDomain string
	// CacheTTL del registro de pools por tenant.
	CacheTTL time.Duration
	// CacheSize maximo (numero de tenants concurrentes en cache).
	CacheSize int
}

// LogConfig controla nivel y formato de logs.
type LogConfig struct {
	Level  string // "debug" | "info" | "warn" | "error"
	Format string // "json" | "text"
}

// FromEnv carga Config desde os.Getenv.
func FromEnv() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:            getStr("HTTP_ADDR", ":8080"),
			ReadTimeout:     getDur("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDur("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDur("HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getDur("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			CentralURL:        getStr("DB_CENTRAL_URL", "postgres://ph:ph@localhost:5432/ph_central?sslmode=disable"),
			TenantTemplateURL: getStr("DB_TENANT_TEMPLATE_URL", "postgres://ph:ph@localhost:5433/ph_tenant_template?sslmode=disable"),
			MaxConns:          getInt32("DB_MAX_CONNS", 10),
			MinConns:          getInt32("DB_MIN_CONNS", 1),
			MaxConnLifetime:   getDur("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		},
		Tenant: TenantConfig{
			BaseDomain: getStr("TENANT_BASE_DOMAIN", "ph.localhost"),
			CacheTTL:   getDur("TENANT_CACHE_TTL", 5*time.Minute),
			CacheSize:  getInt("TENANT_CACHE_SIZE", 256),
		},
		Log: LogConfig{
			Level:  getStr("LOG_LEVEL", "info"),
			Format: getStr("LOG_FORMAT", "json"),
		},
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate verifica las reglas minimas de cordura.
func (c Config) Validate() error {
	if c.HTTP.Addr == "" {
		return errors.New("config: HTTP_ADDR requerido")
	}
	if c.Database.CentralURL == "" {
		return errors.New("config: DB_CENTRAL_URL requerido")
	}
	if c.Database.MaxConns < 1 {
		return errors.New("config: DB_MAX_CONNS debe ser >= 1")
	}
	if c.Tenant.BaseDomain == "" {
		return errors.New("config: TENANT_BASE_DOMAIN requerido")
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		return fmt.Errorf("config: LOG_FORMAT invalido: %q", c.Log.Format)
	}
	return nil
}

func getStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getInt32(key string, def int32) int32 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return def
}

func getDur(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
