package usecases

import (
	"sync"
	"time"
)

// idempotencyEntry guarda una respuesta cacheada con timestamp.
type idempotencyEntry struct {
	storedAt time.Time
	value    any
}

// IdempotencyCache es un cache en-memoria por proceso para deduplicar
// respuestas usando un Idempotency-Key. TTL fijo configurado por el
// orquestador (default 24h). Suficiente para MVP; un sistema robusto
// usaria una tabla `idempotency_records` modulo-local (ADR 0005).
type IdempotencyCache struct {
	mu  sync.Mutex
	m   map[string]idempotencyEntry
	ttl time.Duration
	now func() time.Time
}

// NewIdempotencyCache construye un cache con el TTL dado. Si ttl <= 0,
// se aplica el default (24h). Si nowFn es nil, usa time.Now.
func NewIdempotencyCache(ttl time.Duration, nowFn func() time.Time) *IdempotencyCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if nowFn == nil {
		nowFn = time.Now
	}
	return &IdempotencyCache{
		m:   make(map[string]idempotencyEntry),
		ttl: ttl,
		now: nowFn,
	}
}

// Get devuelve el valor cacheado para key si esta presente y no expiro.
func (c *IdempotencyCache) Get(key string) (any, bool) {
	if key == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok {
		return nil, false
	}
	if c.now().Sub(e.storedAt) > c.ttl {
		delete(c.m, key)
		return nil, false
	}
	return e.value, true
}

// Set guarda value para key. No-op si key esta vacio.
func (c *IdempotencyCache) Set(key string, value any) {
	if key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = idempotencyEntry{storedAt: c.now(), value: value}
}

// Delete elimina la entrada para key.
func (c *IdempotencyCache) Delete(key string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}
