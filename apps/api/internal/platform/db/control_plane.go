package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupFromCentral devuelve un TenantMetadataLookup que resuelve el
// tenant consultando la tabla `tenants` del Control Plane.
//
// La query es deliberadamente literal (no via sqlc) porque vive en el
// Control Plane y es la unica query que se ejecuta antes de saber a que
// pool conectarse. Es codigo de plataforma, no logica de negocio.
func LookupFromCentral(pool *pgxpool.Pool) TenantMetadataLookup {
	return func(ctx context.Context, slug string) (TenantMetadata, error) {
		const q = `
			SELECT id::text, slug, display_name, database_url
			FROM tenants
			WHERE slug = $1 AND status IN ('active', 'provisioning')
		`
		var meta TenantMetadata
		err := pool.QueryRow(ctx, q, slug).Scan(
			&meta.ID, &meta.Slug, &meta.DisplayName, &meta.DatabaseURL,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TenantMetadata{}, ErrTenantNotFound
			}
			return TenantMetadata{}, err
		}
		return meta, nil
	}
}
