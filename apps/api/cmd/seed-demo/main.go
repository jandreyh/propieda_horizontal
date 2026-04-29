// Command seed-demo carga datos minimos para que la UI funcione contra
// el backend real. Es idempotente: se puede correr varias veces sin
// duplicar datos.
//
// Variables de entorno requeridas:
//
//	DB_CENTRAL_URL          URL de Postgres del Control Plane.
//	DB_TENANT_TEMPLATE_URL  URL de Postgres del Tenant DB que se usara
//	                        como base del tenant "demo".
//	TENANT_PUBLIC_URL       (opcional) URL que el API usara para
//	                        conectar al tenant. Por defecto = DB_TENANT_TEMPLATE_URL.
//	ADMIN_PASSWORD          (opcional) password del admin. Default "admin123".
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/platform/passwords"
)

const (
	tenantSlug          = "demo"
	tenantDisplayName   = "Conjunto Demo"
	adminEmail          = "admin@demo.ph.localhost"
	adminDocumentType   = "CC"
	adminDocumentNumber = "1000000001"
	adminNames          = "Admin"
	adminLastNames      = "Demo"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	centralURL := mustEnv("DB_CENTRAL_URL")
	tenantURL := mustEnv("DB_TENANT_TEMPLATE_URL")
	tenantPublicURL := os.Getenv("TENANT_PUBLIC_URL")
	if tenantPublicURL == "" {
		tenantPublicURL = tenantURL
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	if err := seedControlPlane(ctx, centralURL, tenantPublicURL); err != nil {
		return fmt.Errorf("control plane: %w", err)
	}
	if err := seedTenant(ctx, tenantURL, adminPassword); err != nil {
		return fmt.Errorf("tenant: %w", err)
	}

	fmt.Println("seed completed:")
	fmt.Printf("  tenant slug    : %s\n", tenantSlug)
	fmt.Printf("  admin email    : %s\n", adminEmail)
	fmt.Printf("  admin password : %s\n", adminPassword)
	return nil
}

func seedControlPlane(ctx context.Context, url, tenantPublicURL string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		INSERT INTO tenants (slug, display_name, database_url, status, plan, activated_at)
		VALUES ($1, $2, $3, 'active', 'pilot', now())
		ON CONFLICT (slug) DO UPDATE SET database_url = EXCLUDED.database_url, updated_at = now()
	`, tenantSlug, tenantDisplayName, tenantPublicURL)
	return err
}

func seedTenant(ctx context.Context, url, password string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	hash, err := passwords.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Admin user (idempotente por document_type + document_number unique).
	var adminID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (document_type, document_number, names, last_names, email, password_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'active')
		ON CONFLICT (document_type, document_number) DO UPDATE
		    SET email = EXCLUDED.email,
		        password_hash = EXCLUDED.password_hash,
		        names = EXCLUDED.names,
		        last_names = EXCLUDED.last_names,
		        updated_at = now()
		RETURNING id
	`, adminDocumentType, adminDocumentNumber, adminNames, adminLastNames, adminEmail, hash).
		Scan(&adminID)
	if err != nil {
		return fmt.Errorf("upsert admin: %w", err)
	}

	// Asignar rol tenant_admin (assume seed_001 ya inserto roles).
	var roleID string
	err = tx.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'tenant_admin'`).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("rol tenant_admin no encontrado: aplicar seed_001_roles_permissions.up.sql primero")
		}
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_role_assignments (user_id, role_id, scope_type, status)
		VALUES ($1, $2, 'tenant', 'active')
		ON CONFLICT DO NOTHING
	`, adminID, roleID)
	if err != nil {
		// Si no hay constraint unique no falla; no nos preocupa duplicado.
		_, _ = tx.Exec(ctx, `INSERT INTO user_role_assignments (user_id, role_id, scope_type, status) VALUES ($1, $2, 'tenant', 'active')`, adminID, roleID)
	}

	// Estructura "Torre A".
	var towerID string
	err = tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO residential_structures (name, type, description, order_index)
			SELECT 'Torre A', 'tower', 'Torre demo', 1
			WHERE NOT EXISTS (SELECT 1 FROM residential_structures WHERE name = 'Torre A' AND deleted_at IS NULL)
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM residential_structures WHERE name = 'Torre A' AND deleted_at IS NULL
		LIMIT 1
	`).Scan(&towerID)
	if err != nil {
		return fmt.Errorf("upsert structure: %w", err)
	}

	// Unidad Apto 101.
	var unitID string
	err = tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO units (structure_id, code, type, area_m2, bedrooms, coefficient)
			SELECT $1, '101', 'apartment', 72.5, 3, 0.012345
			WHERE NOT EXISTS (
			    SELECT 1 FROM units WHERE code = '101' AND structure_id = $1 AND deleted_at IS NULL
			)
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM units WHERE code = '101' AND structure_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`, towerID).Scan(&unitID)
	if err != nil {
		return fmt.Errorf("upsert unit: %w", err)
	}

	// Categoria de paquete (asume migration ya inserto "Estandar").
	var categoryID string
	err = tx.QueryRow(ctx, `SELECT id FROM package_categories WHERE name = 'Estandar' LIMIT 1`).Scan(&categoryID)
	if err != nil {
		return fmt.Errorf("lookup category: %w", err)
	}

	// Paquete demo.
	_, err = tx.Exec(ctx, `
		INSERT INTO packages (unit_id, recipient_name, category_id, carrier, tracking_number, received_by_user_id, status)
		SELECT $1, 'Admin Demo', $2, 'Servientrega', 'DEMO-0001', $3, 'received'
		WHERE NOT EXISTS (
		    SELECT 1 FROM packages WHERE tracking_number = 'DEMO-0001' AND deleted_at IS NULL
		)
	`, unitID, categoryID, adminID)
	if err != nil {
		return fmt.Errorf("upsert package: %w", err)
	}

	// Anuncio bienvenida.
	var announcementID string
	err = tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO announcements (title, body, published_by_user_id, pinned)
			SELECT 'Bienvenido al sistema', 'Este es el conjunto demo. Todo lo que ves esta servido por el backend Go real.', $1, true
			WHERE NOT EXISTS (
			    SELECT 1 FROM announcements WHERE title = 'Bienvenido al sistema' AND deleted_at IS NULL
			)
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM announcements WHERE title = 'Bienvenido al sistema' AND deleted_at IS NULL
		LIMIT 1
	`, adminID).Scan(&announcementID)
	if err != nil {
		return fmt.Errorf("upsert announcement: %w", err)
	}

	// Audiencia global.
	_, err = tx.Exec(ctx, `
		INSERT INTO announcement_audiences (announcement_id, target_type, target_id)
		SELECT $1, 'global', NULL
		WHERE NOT EXISTS (
		    SELECT 1 FROM announcement_audiences WHERE announcement_id = $1 AND target_type = 'global' AND deleted_at IS NULL
		)
	`, announcementID)
	if err != nil {
		return fmt.Errorf("upsert audience: %w", err)
	}

	return tx.Commit(ctx)
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("env %s requerido", name)
	}
	return v
}
