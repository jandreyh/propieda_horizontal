// Command seed-demo siembra el conjunto demo (slug=demo) usando el
// modelo post-Fase 16: identidad global en platform_users + tenant_user_links.
//
// Es idempotente: si el tenant `demo` ya existe lo borra (DROP DATABASE
// + DELETE tenants) y vuelve a crearlo via el modulo provisioning.
//
// Variables de entorno requeridas:
//   - DATABASE_URL_CENTRAL: pool al Control Plane (ph_central).
//   - PROVISIONING_MAINTENANCE_URL: URL postgres a la DB administrativa
//     del cluster del tenant (ej. .../postgres) para emitir CREATE/DROP.
//   - PROVISIONING_TENANT_URL_TEMPLATE: plantilla con `{dbname}` que el
//     provisioner sustituye por `ph_tenant_demo`.
//   - PROVISIONING_TENANT_MIGRATIONS_PATH: file:// a migrations/tenant.
//
// Uso:
//
//	go run ./cmd/seed-demo
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/provisioning"
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("seed-demo starting")

	centralURL := os.Getenv("DATABASE_URL_CENTRAL")
	maintURL := os.Getenv("PROVISIONING_MAINTENANCE_URL")
	urlTpl := os.Getenv("PROVISIONING_TENANT_URL_TEMPLATE")
	migPath := os.Getenv("PROVISIONING_TENANT_MIGRATIONS_PATH")
	if centralURL == "" || maintURL == "" || urlTpl == "" || migPath == "" {
		return errors.New("missing env: DATABASE_URL_CENTRAL, PROVISIONING_MAINTENANCE_URL, PROVISIONING_TENANT_URL_TEMPLATE, PROVISIONING_TENANT_MIGRATIONS_PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	central, err := pgxpool.New(ctx, centralURL)
	if err != nil {
		return fmt.Errorf("open central pool: %w", err)
	}
	defer central.Close()

	const slug = "demo"
	const dbName = "ph_tenant_demo"

	// Idempotencia: si el tenant ya existe, dropear DB y borrar fila
	// antes de re-provisionar. Esto materializa la decision
	// "borrar `demo` y resembrar" de migration_demo_decisions.md.
	if err := dropDemoIfExists(ctx, logger, central, maintURL, slug, dbName); err != nil {
		return fmt.Errorf("clean previous: %w", err)
	}

	prov := provisioning.New(provisioning.Config{
		CentralPool:        central,
		MaintenanceURL:     maintURL,
		AdminURLTemplate:   urlTpl,
		MigrationsPathFile: migPath,
	})

	logger.Info("provisioning tenant", slog.String("slug", slug))
	out, err := prov.Create(ctx, provisioning.CreateTenantInput{
		Slug:        slug,
		DisplayName: "Conjunto Demo",
		Plan:        "pilot",
		Country:     "CO",
		Currency:    "COP",
		Timezone:    "America/Bogota",
		Admin: provisioning.AdminInput{
			Email:          "admin@demo.ph.localhost",
			DocumentType:   "CC",
			DocumentNumber: "1000000001",
			Names:          "Admin",
			LastNames:      "Demo",
			Password:       "admin123",
			PublicCode:     "DEMO-ADMN-0001",
		},
	})
	if err != nil {
		return fmt.Errorf("provision: %w", err)
	}

	logger.Info("seed-demo done",
		slog.String("tenant_id", out.TenantID.String()),
		slog.String("admin_user_id", out.AdminUserID.String()),
		slog.String("link_id", out.LinkID.String()),
		slog.Bool("admin_reused", out.Reused),
		slog.String("database_url", out.DatabaseURL),
	)
	return nil
}

func dropDemoIfExists(ctx context.Context, logger *slog.Logger, central *pgxpool.Pool, maintURL, slug, dbName string) error {
	var found bool
	if err := central.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`, slug).Scan(&found); err != nil {
		// Si la tabla no existe (central no migrado), continuar sin
		// limpiar — provisioner fallara con un error claro adelante.
		if strings.Contains(err.Error(), "does not exist") {
			logger.Warn("tenants table missing — assuming fresh central")
			return nil
		}
		return fmt.Errorf("check tenant: %w", err)
	}
	if !found {
		logger.Info("no previous demo tenant — skipping cleanup")
		return nil
	}
	logger.Info("dropping previous demo tenant")
	// Drop DB fisica via conn de mantenimiento.
	if err := dropDatabase(ctx, maintURL, dbName); err != nil {
		// No abortar si la DB ya no existe — solo borrar la fila.
		logger.Warn("drop database (non-fatal)", slog.String("error", err.Error()))
	}
	if _, err := central.Exec(ctx, `DELETE FROM platform_user_memberships WHERE tenant_id IN (SELECT id FROM tenants WHERE slug=$1)`, slug); err != nil {
		return fmt.Errorf("delete memberships: %w", err)
	}
	if _, err := central.Exec(ctx, `DELETE FROM tenants WHERE slug = $1`, slug); err != nil {
		return fmt.Errorf("delete tenant row: %w", err)
	}
	return nil
}

func dropDatabase(ctx context.Context, maintURL, dbName string) error {
	pool, err := pgxpool.New(ctx, maintURL)
	if err != nil {
		return fmt.Errorf("connect maintenance: %w", err)
	}
	defer pool.Close()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName)); err != nil {
		return err
	}
	return nil
}
