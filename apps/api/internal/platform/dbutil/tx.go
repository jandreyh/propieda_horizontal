// Package dbutil expone helpers transversales para trabajar con pgxpool
// y transacciones, segun los lineamientos del ADR 0005.
//
// Patron principal: `WithTx(ctx, pool, opts, fn)` abre una transaccion,
// ejecuta `fn(tx)`, y commitea o hace rollback segun el resultado. Asi
// los usecases reciben siempre un `pgx.Tx` y los repositorios pueden
// componerse con `Querier` polimorfico (sqlc.Querier).
package dbutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx ejecuta fn dentro de una transaccion y maneja commit/rollback.
//
// Si fn devuelve error, se hace rollback y se devuelve ese error
// envolviendo cualquier error de rollback secundario.
func WithTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return errors.New("dbutil: pool nil")
	}
	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("dbutil: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // no-op si ya fue commit.
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("dbutil: commit: %w", err)
	}
	return nil
}

// WithTxRetry intenta WithTx hasta maxAttempts veces si el error es de
// serializacion (`40001`). Pensado para operaciones SERIALIZABLE.
func WithTxRetry(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, maxAttempts int, fn func(tx pgx.Tx) error) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var lastErr error
	for range maxAttempts {
		err := WithTx(ctx, pool, opts, fn)
		if err == nil {
			return nil
		}
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "40001" {
			lastErr = err
			continue
		}
		return err
	}
	return fmt.Errorf("dbutil: max retries reached: %w", lastErr)
}
