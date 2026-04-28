package db

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"
)

// TenantMetadataLookup es la firma que el Registry usa para resolver el
// metadata + URL de la base del tenant a partir de su slug. La fuente
// real es el Control Plane (tabla `tenants`); en tests se mockea.
type TenantMetadataLookup func(ctx context.Context, slug string) (TenantMetadata, error)

// TenantMetadata es la informacion minima necesaria para abrir el pool
// del tenant.
type TenantMetadata struct {
	ID          string
	Slug        string
	DisplayName string
	DatabaseURL string
}

// Registry mantiene un cache (con TTL) de pools pgx por tenant.
//
// Coalesce de carga: si N goroutines piden el mismo slug en frio, solo
// UNA ejecuta lookup + open; las demas reciben el resultado (o error) de
// esa carga via `singleflight.Group`.
type Registry struct {
	lookup     TenantMetadataLookup
	cfg        PoolConfig
	ttl        time.Duration
	maxEntries int

	mu      sync.Mutex
	entries map[string]*entry
	sfg     singleflight.Group
}

type entry struct {
	meta      TenantMetadata
	pool      *pgxpool.Pool
	expiresAt time.Time
}

// RegistryConfig agrupa parametros del Registry.
type RegistryConfig struct {
	Lookup     TenantMetadataLookup
	PoolConfig PoolConfig
	CacheTTL   time.Duration
	MaxEntries int
}

// NewRegistry construye un Registry vacio.
func NewRegistry(cfg RegistryConfig) (*Registry, error) {
	if cfg.Lookup == nil {
		return nil, errors.New("db: Registry requiere Lookup")
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 256
	}
	return &Registry{
		lookup:     cfg.Lookup,
		cfg:        cfg.PoolConfig,
		ttl:        cfg.CacheTTL,
		maxEntries: cfg.MaxEntries,
		entries:    make(map[string]*entry, cfg.MaxEntries),
	}, nil
}

// ErrTenantNotFound se devuelve cuando el lookup no encuentra el slug.
var ErrTenantNotFound = errors.New("db: tenant not found")

// Get retorna (metadata, pool) del tenant identificado por slug. Si no
// hay entrada vigente abre una nueva. Cierra y reemplaza si esta caducada.
func (r *Registry) Get(ctx context.Context, slug string) (TenantMetadata, *pgxpool.Pool, error) {
	if slug == "" {
		return TenantMetadata{}, nil, errors.New("db: slug vacio")
	}

	// Lectura optimista.
	if meta, pool, ok := r.cachedHit(slug); ok {
		return meta, pool, nil
	}

	// Coalesce: solo una goroutine por slug abre la conexion en frio.
	v, err, _ := r.sfg.Do(slug, func() (any, error) {
		// Re-check tras adquirir el slot single-flight.
		if meta, pool, ok := r.cachedHit(slug); ok {
			return loaded{meta: meta, pool: pool}, nil
		}

		meta, err := r.lookup(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("db: lookup tenant %q: %w", slug, err)
		}

		poolCfg := r.cfg
		poolCfg.URL = meta.DatabaseURL
		pool, err := NewPool(ctx, poolCfg)
		if err != nil {
			return nil, fmt.Errorf("db: open tenant %q: %w", slug, err)
		}

		r.store(slug, meta, pool)
		return loaded{meta: meta, pool: pool}, nil
	})
	if err != nil {
		return TenantMetadata{}, nil, err
	}
	l := v.(loaded)
	return l.meta, l.pool, nil
}

type loaded struct {
	meta TenantMetadata
	pool *pgxpool.Pool
}

func (r *Registry) cachedHit(slug string) (TenantMetadata, *pgxpool.Pool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[slug]
	if !ok || !time.Now().Before(e.expiresAt) {
		return TenantMetadata{}, nil, false
	}
	return e.meta, e.pool, true
}

func (r *Registry) store(slug string, meta TenantMetadata, pool *pgxpool.Pool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.entries[slug]; ok && old.pool != nil && old.pool != pool {
		old.pool.Close()
	}
	if len(r.entries) >= r.maxEntries {
		r.evictOldestLocked()
	}
	r.entries[slug] = &entry{
		meta:      meta,
		pool:      pool,
		expiresAt: time.Now().Add(r.ttl),
	}
}

// Invalidate fuerza la expulsion de un slug del cache (cierra su pool).
// Util tras renombrar/desactivar un tenant en el Control Plane.
func (r *Registry) Invalidate(slug string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[slug]; ok {
		if e.pool != nil {
			e.pool.Close()
		}
		delete(r.entries, slug)
	}
}

// Close cierra todos los pools cacheados.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for slug, e := range r.entries {
		if e.pool != nil {
			e.pool.Close()
		}
		delete(r.entries, slug)
	}
}

// evictOldestLocked expulsa la entrada con expiresAt mas viejo. Caller
// debe sostener r.mu.
func (r *Registry) evictOldestLocked() {
	var oldestSlug string
	var oldestTime time.Time
	for slug, e := range r.entries {
		if oldestSlug == "" || e.expiresAt.Before(oldestTime) {
			oldestSlug = slug
			oldestTime = e.expiresAt
		}
	}
	if oldestSlug != "" {
		if e := r.entries[oldestSlug]; e != nil && e.pool != nil {
			e.pool.Close()
		}
		delete(r.entries, oldestSlug)
	}
}
