// Package db gestiona los pools pgx para el Control Plane y para cada
// Tenant DB resuelto.
//
// Reglas:
//   - Solo `pgx.Pool` (jackc/pgx/v5/pgxpool). Nunca `database/sql` directo.
//   - Todas las queries pasan por sqlc (no inline SQL en Go).
//   - El pool del Control Plane es unico para todo el proceso. Los pools
//     de Tenant DBs viven en un Registry con expiracion (ver tenant_pool.go).
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig configura un pool pgx genericamente.
type PoolConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	HealthCheck     time.Duration
}

// NewPool crea un *pgxpool.Pool y lo verifica con un Ping.
//
// Es responsabilidad del caller llamar a Pool.Close() en el shutdown.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, errors.New("db: URL requerida")
	}
	pgxCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}
	if cfg.MaxConns > 0 {
		pgxCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		pgxCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		pgxCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.HealthCheck > 0 {
		pgxCfg.HealthCheckPeriod = cfg.HealthCheck
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}
