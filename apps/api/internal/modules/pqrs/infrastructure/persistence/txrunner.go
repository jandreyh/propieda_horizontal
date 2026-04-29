package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/pqrs/application/usecases"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// TenantTxRunner implementa usecases.TxRunner resolviendo el pool del
// tenant del contexto y abriendo una transaccion con el nivel de
// aislamiento dado. La tx se inyecta en el contexto hijo via WithTx para
// que los repos del modulo la usen automaticamente.
type TenantTxRunner struct{}

// NewTenantTxRunner construye una instancia stateless.
func NewTenantTxRunner() *TenantTxRunner { return &TenantTxRunner{} }

// RunInTx implementa usecases.TxRunner.
func (TenantTxRunner) RunInTx(ctx context.Context, level pgx.TxIsoLevel, fn func(ctx context.Context) error) error {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return err
	}
	if t.Pool == nil {
		return errors.New("pqrs: tenant pool is nil")
	}
	tx, err := t.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(WithTx(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Compile-time check.
var _ usecases.TxRunner = (*TenantTxRunner)(nil)
