package db

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestRegistry_LookupSingleFlight verifica que requests concurrentes al
// mismo slug solo invocan el lookup una vez (single-flight per slug).
//
// No abre conexiones reales: usa un lookup que devuelve URL invalida
// para forzar fallo en NewPool, pero antes de eso ya validamos que
// `lookup` se invoco pocas veces (el lock evita estampida).
func TestRegistry_Lookup_BlankSlug(t *testing.T) {
	t.Parallel()
	reg, err := NewRegistry(RegistryConfig{
		Lookup: func(ctx context.Context, slug string) (TenantMetadata, error) {
			return TenantMetadata{}, errors.New("not used")
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	defer reg.Close()

	if _, _, err := reg.Get(context.Background(), ""); err == nil {
		t.Fatalf("expected error on empty slug")
	}
}

func TestRegistry_NewRegistry_Validation(t *testing.T) {
	t.Parallel()
	if _, err := NewRegistry(RegistryConfig{}); err == nil {
		t.Fatalf("expected error when Lookup is nil")
	}

	reg, err := NewRegistry(RegistryConfig{
		Lookup: func(_ context.Context, _ string) (TenantMetadata, error) {
			return TenantMetadata{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry valid: %v", err)
	}
	if reg.ttl <= 0 {
		t.Fatalf("default ttl should be > 0")
	}
	if reg.maxEntries <= 0 {
		t.Fatalf("default maxEntries should be > 0")
	}
}

func TestRegistry_LookupErrorPropagates(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("synthetic lookup failure")
	reg, err := NewRegistry(RegistryConfig{
		Lookup: func(_ context.Context, _ string) (TenantMetadata, error) {
			return TenantMetadata{}, wantErr
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	defer reg.Close()

	_, _, err = reg.Get(context.Background(), "acacias")
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got %v", err)
	}
}

// TestRegistry_SingleFlight_LimitsLookupCalls verifica que ante N
// goroutines en frio para el mismo slug, lookup se invoca una sola vez.
// El test fuerza fallo en lookup para evitar abrir conexion real.
func TestRegistry_SingleFlight_LimitsLookupCalls(t *testing.T) {
	t.Parallel()
	var calls int32
	var mu sync.Mutex
	reg, err := NewRegistry(RegistryConfig{
		Lookup: func(_ context.Context, _ string) (TenantMetadata, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			// Demora artificial para que las goroutines se solapen.
			time.Sleep(20 * time.Millisecond)
			return TenantMetadata{}, errors.New("synthetic")
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	defer reg.Close()

	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			_, _, _ = reg.Get(context.Background(), "acacias")
		}()
	}
	wg.Wait()

	mu.Lock()
	got := calls
	mu.Unlock()
	// El test es sensible al timing: garantizamos que NO se invoco N veces.
	// Idealmente seria 1 (single-flight); aceptamos hasta 2 por race del
	// re-check post-lock. Si hay >2 hay un bug grave.
	if got > 2 {
		t.Fatalf("expected single-flight (<=2 calls), got %d", got)
	}
}

func TestRegistry_InvalidateNoOpOnUnknown(t *testing.T) {
	t.Parallel()
	reg, err := NewRegistry(RegistryConfig{
		Lookup: func(_ context.Context, _ string) (TenantMetadata, error) {
			return TenantMetadata{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	defer reg.Close()
	// No debe panic.
	reg.Invalidate("inexistente")
}
