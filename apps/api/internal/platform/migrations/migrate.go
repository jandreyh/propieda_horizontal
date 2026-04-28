// Package migrations envuelve `golang-migrate` para correr las
// migraciones del Control Plane y de cada Tenant DB.
//
// Uso tipico:
//
//	m, _ := migrations.New("file:///abs/path/migrations/central", centralURL)
//	defer m.Close()
//	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) { ... }
//
// Uso por-tenant (provisioning): mismo patron pero con la carpeta
// `migrations/tenant` y la URL del Tenant DB.
package migrations

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Drivers requeridos por golang-migrate (init).
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator envuelve un *migrate.Migrate para exponer un API minimo.
type Migrator struct {
	m *migrate.Migrate
}

// New construye un Migrator. `sourceURL` debe usar el esquema `file://`
// y apuntar a un directorio absoluto. `dbURL` es la URL de Postgres.
func New(sourceURL, dbURL string) (*Migrator, error) {
	if sourceURL == "" {
		return nil, errors.New("migrations: sourceURL requerido")
	}
	if dbURL == "" {
		return nil, errors.New("migrations: dbURL requerido")
	}
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return nil, fmt.Errorf("migrations: new: %w", err)
	}
	return &Migrator{m: m}, nil
}

// Up aplica todas las migraciones pendientes. ErrNoChange no es error.
func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations: up: %w", err)
	}
	return nil
}

// Down revierte una migracion. ErrNoChange no es error.
func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations: down: %w", err)
	}
	return nil
}

// Steps ejecuta n pasos (positivo = up, negativo = down).
func (m *Migrator) Steps(n int) error {
	if err := m.m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations: steps(%d): %w", n, err)
	}
	return nil
}

// Version devuelve la version actual aplicada y si esta dirty.
func (m *Migrator) Version() (uint, bool, error) {
	v, dirty, err := m.m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("migrations: version: %w", err)
	}
	return v, dirty, nil
}

// Close cierra ambos drivers.
func (m *Migrator) Close() error {
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		return fmt.Errorf("migrations: close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("migrations: close db: %w", dbErr)
	}
	return nil
}
