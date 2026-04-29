package usecases

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TxRunner es la abstraccion que permite a los usecases ejecutar logica
// dentro de una transaccion sin acoplar la capa de aplicacion a pgx.
//
// La implementacion default vive en infra y usa pgxpool del tenant
// resuelto del contexto + el helper persistence.WithTx para inyectar la
// tx en el contexto interno que ven los repos. Para tests, se inyecta
// una implementacion no-tx que solo invoca fn(ctx, nil).
type TxRunner interface {
	// RunInTx ejecuta fn dentro de una transaccion con el nivel de
	// aislamiento dado. La implementacion DEBE colocar la tx en el ctx
	// hijo que recibe fn (para que los repos la usen).
	RunInTx(ctx context.Context, level pgx.TxIsoLevel, fn func(ctx context.Context) error) error
}
