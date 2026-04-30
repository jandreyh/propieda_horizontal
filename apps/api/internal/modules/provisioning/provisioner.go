// Package provisioning materializa la creacion de un nuevo tenant.
//
// La operacion es transaccional con compensaciones (no atomica entre
// operaciones DDL — Postgres no permite CREATE DATABASE dentro de una
// transaccion). El orden y la compensacion en caso de fallo son:
//
//  1. INSERT tenants (status=provisioning).
//  2. CREATE DATABASE ph_tenant_<slug>.
//  3. Correr migraciones contra la DB nueva.
//  4. Upsert platform_user admin (DB central).
//  5. INSERT tenant_user_links (DB del tenant) con role=tenant_admin.
//  6. INSERT platform_user_memberships (DB central).
//  7. UPDATE tenants status=active.
//
// Si falla en el paso N, deshace los pasos 1..N-1 (DROP DATABASE +
// DELETE tenants) — los pasos 5/6/7 se compensan con DROP DATABASE
// porque es la accion mas barata. El usuario en platform_users SE PRESERVA
// si ya existia (idempotencia: re-provisioning con el mismo admin reusa).
//
// Spec: docs/specs/fase-16-cross-tenant-identity-spec.md seccion 16.5.
package provisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/platform/migrations"
	"github.com/saas-ph/api/internal/platform/passwords"
)

// AdminInput agrupa los datos del admin que se crea (o reusa) junto con
// el tenant.
type AdminInput struct {
	Email          string
	DocumentType   string
	DocumentNumber string
	Names          string
	LastNames      string
	Password       string // plain
	Phone          *string
	PublicCode     string // si vacio, se genera
}

// CreateTenantInput agrupa todos los parametros del provisioning.
type CreateTenantInput struct {
	Slug          string
	DisplayName   string
	AdministratorID *uuid.UUID
	Plan          string
	Country       string
	Currency      string
	Timezone      string
	ExpectedUnits *int32
	Admin         AdminInput
}

// CreateTenantOutput resume el resultado.
type CreateTenantOutput struct {
	TenantID    uuid.UUID
	DatabaseURL string
	AdminUserID uuid.UUID
	LinkID      uuid.UUID
	Reused      bool // true si reuso un platform_user existente
}

// Config son los parametros de cableado del provisioner.
type Config struct {
	// CentralPool es el pool a la DB de control. Se usa para DDL
	// (CREATE DATABASE) y para los inserts en platform_*.
	CentralPool *pgxpool.Pool
	// AdminURLTemplate es la URL postgres con placeholder `{dbname}` que
	// se sustituira por el nombre fisico de la DB del tenant.
	// Ej: "postgres://app:app@localhost:5433/{dbname}?sslmode=disable".
	AdminURLTemplate string
	// MaintenanceURL es una URL Postgres a una DB administrativa (ej.
	// "postgres" o "template1") usada para ejecutar el CREATE DATABASE
	// (que no se puede correr dentro de la DB que se va a crear ni
	// dentro de una transaccion).
	MaintenanceURL string
	// MigrationsPathFile es el path file:// a migrations/tenant. Ej:
	// "file:///D:/propieda_horizontal/migrations/tenant".
	MigrationsPathFile string
}

// Provisioner ejecuta CreateTenant.
type Provisioner struct {
	cfg Config
}

// New construye un Provisioner.
func New(cfg Config) *Provisioner {
	return &Provisioner{cfg: cfg}
}

// Create ejecuta el flujo completo. Devuelve el output o un error
// (despues de aplicar compensaciones).
func (p *Provisioner) Create(ctx context.Context, in CreateTenantInput) (*CreateTenantOutput, error) {
	if err := validateInput(in); err != nil {
		return nil, err
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	dbName := "ph_tenant_" + strings.ReplaceAll(slug, "-", "_")
	dbURL := strings.ReplaceAll(p.cfg.AdminURLTemplate, "{dbname}", dbName)

	tenantID, err := p.insertTenantRow(ctx, in, slug, dbURL)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	// Si algo falla despues, compensar con DROP DATABASE + DELETE tenants.
	createdDB := false
	rollback := func(stage string, original error) error {
		var msgs []string
		if createdDB {
			if dropErr := p.dropDatabase(ctx, dbName); dropErr != nil {
				msgs = append(msgs, fmt.Sprintf("drop db: %v", dropErr))
			}
		}
		if delErr := p.deleteTenantRow(ctx, tenantID); delErr != nil {
			msgs = append(msgs, fmt.Sprintf("delete tenant row: %v", delErr))
		}
		if len(msgs) > 0 {
			return fmt.Errorf("%s: %w (compensation issues: %s)", stage, original, strings.Join(msgs, "; "))
		}
		return fmt.Errorf("%s: %w", stage, original)
	}

	if err := p.createDatabase(ctx, dbName); err != nil {
		return nil, rollback("create database", err)
	}
	createdDB = true

	if err := p.runMigrations(dbURL); err != nil {
		return nil, rollback("run migrations", err)
	}

	adminID, reused, err := p.upsertAdmin(ctx, in.Admin)
	if err != nil {
		return nil, rollback("upsert admin", err)
	}

	linkID, err := p.insertTenantUserLink(ctx, dbURL, adminID)
	if err != nil {
		return nil, rollback("insert tenant_user_link", err)
	}

	if err := p.insertCentralMembership(ctx, adminID, tenantID); err != nil {
		return nil, rollback("insert central membership", err)
	}

	if err := p.activateTenant(ctx, tenantID); err != nil {
		return nil, rollback("activate tenant", err)
	}

	return &CreateTenantOutput{
		TenantID:    tenantID,
		DatabaseURL: dbURL,
		AdminUserID: adminID,
		LinkID:      linkID,
		Reused:      reused,
	}, nil
}

func validateInput(in CreateTenantInput) error {
	if in.Slug == "" {
		return errors.New("provisioning: slug required")
	}
	if in.DisplayName == "" {
		return errors.New("provisioning: display_name required")
	}
	if in.Admin.Email == "" || in.Admin.Password == "" {
		return errors.New("provisioning: admin email + password required")
	}
	if in.Admin.DocumentType == "" || in.Admin.DocumentNumber == "" {
		return errors.New("provisioning: admin document required")
	}
	if in.Admin.Names == "" || in.Admin.LastNames == "" {
		return errors.New("provisioning: admin names required")
	}
	return nil
}

func (p *Provisioner) insertTenantRow(ctx context.Context, in CreateTenantInput, slug, dbURL string) (uuid.UUID, error) {
	plan := defaultStr(in.Plan, "pilot")
	country := defaultStr(in.Country, "CO")
	currency := defaultStr(in.Currency, "COP")
	tz := defaultStr(in.Timezone, "America/Bogota")
	var adminID *uuid.UUID
	if in.AdministratorID != nil {
		adminID = in.AdministratorID
	}
	var id uuid.UUID
	row := p.cfg.CentralPool.QueryRow(ctx, `
		INSERT INTO tenants (slug, display_name, database_url, plan, status,
			administrator_id, timezone, country, currency, expected_units, activated_at)
		VALUES ($1, $2, $3, $4, 'provisioning', $5, $6, $7, $8, $9, NULL)
		RETURNING id
	`, slug, in.DisplayName, dbURL, plan, adminID, tz, country, currency, in.ExpectedUnits)
	if err := row.Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (p *Provisioner) deleteTenantRow(ctx context.Context, tenantID uuid.UUID) error {
	_, err := p.cfg.CentralPool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	return err
}

func (p *Provisioner) activateTenant(ctx context.Context, tenantID uuid.UUID) error {
	_, err := p.cfg.CentralPool.Exec(ctx, `
		UPDATE tenants
		SET status = 'active',
		    activated_at = COALESCE(activated_at, now()),
		    updated_at = now()
		WHERE id = $1`, tenantID)
	return err
}

// createDatabase abre una conexion a la URL de mantenimiento y emite
// CREATE DATABASE. Usa pgx.Connect (no pool) porque es DDL one-shot.
func (p *Provisioner) createDatabase(ctx context.Context, dbName string) error {
	if err := validateIdentifier(dbName); err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, p.cfg.MaintenanceURL)
	if err != nil {
		return fmt.Errorf("connect maintenance: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}

func (p *Provisioner) dropDatabase(ctx context.Context, dbName string) error {
	if err := validateIdentifier(dbName); err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, p.cfg.MaintenanceURL)
	if err != nil {
		return fmt.Errorf("connect maintenance: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName)); err != nil {
		return fmt.Errorf("drop database %s: %w", dbName, err)
	}
	return nil
}

func (p *Provisioner) runMigrations(dbURL string) error {
	m, err := migrations.New(p.cfg.MigrationsPathFile, dbURL)
	if err != nil {
		return fmt.Errorf("new migrator: %w", err)
	}
	defer func() { _ = m.Close() }()
	return m.Up()
}

func (p *Provisioner) upsertAdmin(ctx context.Context, a AdminInput) (uuid.UUID, bool, error) {
	hash, err := passwords.Hash(a.Password)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("hash password: %w", err)
	}
	publicCode := a.PublicCode
	if publicCode == "" {
		publicCode = generatePublicCode()
	}
	// Intentar insert. Si choca por email/document existente, leer y
	// reusar.
	var id uuid.UUID
	err = p.cfg.CentralPool.QueryRow(ctx, `
		INSERT INTO platform_users (
			document_type, document_number, names, last_names,
			email, phone, password_hash, public_code, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active')
		RETURNING id
	`, a.DocumentType, a.DocumentNumber, a.Names, a.LastNames,
		strings.ToLower(a.Email), a.Phone, hash, publicCode).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	// Conflicto: buscar por email.
	if !isUniqueViolation(err) {
		return uuid.Nil, false, fmt.Errorf("insert admin: %w", err)
	}
	row := p.cfg.CentralPool.QueryRow(ctx, `
		SELECT id FROM platform_users WHERE lower(email) = lower($1) AND deleted_at IS NULL
	`, a.Email)
	if err := row.Scan(&id); err != nil {
		return uuid.Nil, false, fmt.Errorf("read existing admin: %w", err)
	}
	return id, true, nil
}

func (p *Provisioner) insertTenantUserLink(ctx context.Context, dbURL string, adminID uuid.UUID) (uuid.UUID, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return uuid.Nil, fmt.Errorf("connect tenant db: %w", err)
	}
	defer pool.Close()
	var id uuid.UUID
	row := pool.QueryRow(ctx, `
		INSERT INTO tenant_user_links (platform_user_id, role, status)
		VALUES ($1, 'tenant_admin', 'active')
		RETURNING id
	`, adminID)
	if err := row.Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("insert link: %w", err)
	}
	return id, nil
}

func (p *Provisioner) insertCentralMembership(ctx context.Context, adminID, tenantID uuid.UUID) error {
	_, err := p.cfg.CentralPool.Exec(ctx, `
		INSERT INTO platform_user_memberships (platform_user_id, tenant_id, role, status)
		VALUES ($1, $2, 'tenant_admin', 'active')
		ON CONFLICT (platform_user_id, tenant_id) DO UPDATE
		SET status = 'active', role = 'tenant_admin', updated_at = now()
	`, adminID, tenantID)
	return err
}

// validateIdentifier evita SQL injection en nombres de DB. Permite solo
// [a-z0-9_] (los slugs son DNS-friendly + reemplazo de `-` por `_`).
func validateIdentifier(s string) error {
	if s == "" {
		return errors.New("identifier empty")
	}
	for _, c := range s {
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
		if !ok {
			return fmt.Errorf("identifier %q contains invalid char %q", s, c)
		}
	}
	return nil
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// generatePublicCode produce un codigo XXXX-XXXX-XXXX con alfabeto
// reducido (sin O/0/I/l/1).
func generatePublicCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	now := time.Now().UnixNano()
	out := make([]byte, 14)
	pos := 0
	for i := range 12 {
		if i > 0 && i%4 == 0 {
			out[pos] = '-'
			pos++
		}
		idx := int(now % int64(len(alphabet)))
		now /= int64(len(alphabet))
		if now == 0 {
			now = time.Now().UnixNano() ^ int64(i)
		}
		out[pos] = alphabet[idx]
		pos++
	}
	return string(out[:pos])
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint")
}
