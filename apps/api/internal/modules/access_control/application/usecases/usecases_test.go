package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/access_control/application/usecases"
	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// --- mocks ---

type fakePreReg struct {
	createFn     func(ctx context.Context, in domain.CreatePreRegistrationInput) (entities.PreRegistration, error)
	getByQRFn    func(ctx context.Context, qrHash string) (entities.PreRegistration, error)
	consumeOneFn func(ctx context.Context, qrHash string) (entities.PreRegistration, error)
}

func (f *fakePreReg) Create(ctx context.Context, in domain.CreatePreRegistrationInput) (entities.PreRegistration, error) {
	return f.createFn(ctx, in)
}
func (f *fakePreReg) GetByQRHash(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
	return f.getByQRFn(ctx, qrHash)
}
func (f *fakePreReg) ConsumeOne(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
	return f.consumeOneFn(ctx, qrHash)
}

type fakeBlacklist struct {
	getFn     func(ctx context.Context, dt entities.DocumentType, dn string) (*entities.BlacklistEntry, error)
	createFn  func(ctx context.Context, in domain.CreateBlacklistInput) (entities.BlacklistEntry, error)
	listFn    func(ctx context.Context) ([]entities.BlacklistEntry, error)
	archiveFn func(ctx context.Context, id, actor string) (entities.BlacklistEntry, error)
}

func (f *fakeBlacklist) Get(ctx context.Context, dt entities.DocumentType, dn string) (*entities.BlacklistEntry, error) {
	if f.getFn == nil {
		return nil, nil
	}
	return f.getFn(ctx, dt, dn)
}
func (f *fakeBlacklist) Create(ctx context.Context, in domain.CreateBlacklistInput) (entities.BlacklistEntry, error) {
	return f.createFn(ctx, in)
}
func (f *fakeBlacklist) List(ctx context.Context) ([]entities.BlacklistEntry, error) {
	return f.listFn(ctx)
}
func (f *fakeBlacklist) Archive(ctx context.Context, id, actor string) (entities.BlacklistEntry, error) {
	return f.archiveFn(ctx, id, actor)
}

type fakeEntries struct {
	createFn     func(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error)
	closeFn      func(ctx context.Context, id, actor string) (entities.VisitorEntry, error)
	listActiveFn func(ctx context.Context) ([]entities.VisitorEntry, error)
	getByIDFn    func(ctx context.Context, id string) (entities.VisitorEntry, error)
}

func (f *fakeEntries) Create(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
	return f.createFn(ctx, in)
}
func (f *fakeEntries) Close(ctx context.Context, id, actor string) (entities.VisitorEntry, error) {
	return f.closeFn(ctx, id, actor)
}
func (f *fakeEntries) ListActive(ctx context.Context) ([]entities.VisitorEntry, error) {
	return f.listActiveFn(ctx)
}
func (f *fakeEntries) GetByID(ctx context.Context, id string) (entities.VisitorEntry, error) {
	return f.getByIDFn(ctx, id)
}

// --- helpers ---

const validUUID = "11111111-2222-3333-4444-555555555555"
const otherUUID = "22222222-3333-4444-5555-666666666666"

func mustProblem(t *testing.T, err error, status int) {
	t.Helper()
	var p apperrors.Problem
	if !errors.As(err, &p) {
		t.Fatalf("expected apperrors.Problem, got %v", err)
	}
	if p.Status != status {
		t.Fatalf("expected status %d, got %d (%s)", status, p.Status, p.Detail)
	}
}

// --- CreatePreRegistration ---

func TestCreatePreRegistration_Golden(t *testing.T) {
	var captured domain.CreatePreRegistrationInput
	repo := &fakePreReg{
		createFn: func(ctx context.Context, in domain.CreatePreRegistrationInput) (entities.PreRegistration, error) {
			captured = in
			return entities.PreRegistration{
				ID:              "11111111-2222-3333-4444-555555555555",
				UnitID:          in.UnitID,
				CreatedByUserID: in.CreatedByUserID,
				VisitorFullName: in.VisitorFullName,
				ExpiresAt:       in.ExpiresAt,
				MaxUses:         in.MaxUses,
				QRCodeHash:      in.QRCodeHash,
				Status:          entities.PreRegistrationStatusActive,
				Version:         1,
			}, nil
		},
	}
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	uc := usecases.CreatePreRegistration{
		Repo:  repo,
		Now:   func() time.Time { return now },
		QRGen: func() (string, error) { return "RAW-QR", nil },
	}
	out, err := uc.Execute(context.Background(), usecases.CreatePreRegistrationInput{
		UnitID:          validUUID,
		CreatedByUserID: otherUUID,
		VisitorFullName: " Juan Perez ",
		ExpiresAt:       now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.QRCode != "RAW-QR" {
		t.Errorf("expected raw QR returned, got %q", out.QRCode)
	}
	if captured.QRCodeHash != usecases.HashQRCode("RAW-QR") {
		t.Errorf("expected hashed QR persisted, got %q", captured.QRCodeHash)
	}
	if captured.MaxUses != 1 {
		t.Errorf("expected MaxUses default=1, got %d", captured.MaxUses)
	}
	if captured.VisitorFullName != "Juan Perez" {
		t.Errorf("expected trimmed name, got %q", captured.VisitorFullName)
	}
}

func TestCreatePreRegistration_BadUnitID(t *testing.T) {
	uc := usecases.CreatePreRegistration{Repo: &fakePreReg{}}
	_, err := uc.Execute(context.Background(), usecases.CreatePreRegistrationInput{
		UnitID:          "bad",
		CreatedByUserID: validUUID,
		VisitorFullName: "X",
		ExpiresAt:       time.Now().Add(time.Hour),
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreatePreRegistration_PastExpiresAt(t *testing.T) {
	uc := usecases.CreatePreRegistration{
		Repo: &fakePreReg{},
		Now:  func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) },
	}
	_, err := uc.Execute(context.Background(), usecases.CreatePreRegistrationInput{
		UnitID:          validUUID,
		CreatedByUserID: otherUUID,
		VisitorFullName: "X",
		ExpiresAt:       time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC),
	})
	mustProblem(t, err, http.StatusBadRequest)
}

// --- CheckinByQR ---

func TestCheckinByQR_Golden(t *testing.T) {
	pre := entities.PreRegistration{
		ID:              validUUID,
		UnitID:          otherUUID,
		VisitorFullName: "Juan",
		Status:          entities.PreRegistrationStatusActive,
	}
	preRepo := &fakePreReg{
		consumeOneFn: func(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
			return pre, nil
		},
	}
	bl := &fakeBlacklist{
		getFn: func(ctx context.Context, dt entities.DocumentType, dn string) (*entities.BlacklistEntry, error) {
			return nil, nil
		},
	}
	var captured domain.CreateVisitorEntryInput
	er := &fakeEntries{
		createFn: func(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
			captured = in
			return entities.VisitorEntry{
				ID:                    validUUID,
				UnitID:                in.UnitID,
				PreRegistrationID:     in.PreRegistrationID,
				VisitorFullName:       in.VisitorFullName,
				VisitorDocumentNumber: in.VisitorDocumentNumber,
				GuardID:               in.GuardID,
				Source:                in.Source,
				Status:                in.Status,
			}, nil
		},
	}
	uc := usecases.CheckinByQR{PreRegRepo: preRepo, BlacklistRepo: bl, EntryRepo: er}
	out, err := uc.Execute(context.Background(), usecases.CheckinByQRInput{
		QRCode:  "RAW",
		GuardID: validUUID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if captured.Source != entities.VisitorEntrySourceQR {
		t.Errorf("expected qr source, got %q", captured.Source)
	}
	if captured.Status != entities.VisitorEntryStatusActive {
		t.Errorf("expected active status, got %q", captured.Status)
	}
	if out.Status != entities.VisitorEntryStatusActive {
		t.Errorf("output status: got %q", out.Status)
	}
}

func TestCheckinByQR_Exhausted_Maps410(t *testing.T) {
	preRepo := &fakePreReg{
		consumeOneFn: func(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
			return entities.PreRegistration{}, domain.ErrPreregistrationExhausted
		},
	}
	uc := usecases.CheckinByQR{
		PreRegRepo:    preRepo,
		BlacklistRepo: &fakeBlacklist{},
		EntryRepo:     &fakeEntries{},
	}
	_, err := uc.Execute(context.Background(), usecases.CheckinByQRInput{
		QRCode:  "RAW",
		GuardID: validUUID,
	})
	mustProblem(t, err, http.StatusGone)
}

func TestCheckinByQR_Expired_Maps410(t *testing.T) {
	preRepo := &fakePreReg{
		consumeOneFn: func(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
			return entities.PreRegistration{}, domain.ErrPreregistrationExpired
		},
	}
	uc := usecases.CheckinByQR{
		PreRegRepo:    preRepo,
		BlacklistRepo: &fakeBlacklist{},
		EntryRepo:     &fakeEntries{},
	}
	_, err := uc.Execute(context.Background(), usecases.CheckinByQRInput{
		QRCode:  "RAW",
		GuardID: validUUID,
	})
	mustProblem(t, err, http.StatusGone)
}

func TestCheckinByQR_BlacklistedRegistersRejectedAnd403(t *testing.T) {
	dt := "CC"
	dn := "12345"
	pre := entities.PreRegistration{
		ID:                    validUUID,
		UnitID:                otherUUID,
		VisitorFullName:       "Juan",
		VisitorDocumentType:   &dt,
		VisitorDocumentNumber: &dn,
		Status:                entities.PreRegistrationStatusActive,
	}
	preRepo := &fakePreReg{
		consumeOneFn: func(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
			return pre, nil
		},
	}
	bl := &fakeBlacklist{
		getFn: func(ctx context.Context, _ entities.DocumentType, _ string) (*entities.BlacklistEntry, error) {
			return &entities.BlacklistEntry{
				ID:     "blid",
				Reason: "vandalism",
				Status: entities.BlacklistStatusActive,
			}, nil
		},
	}
	createCalls := 0
	er := &fakeEntries{
		createFn: func(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
			createCalls++
			if in.Status != entities.VisitorEntryStatusRejected {
				t.Errorf("expected rejected status, got %q", in.Status)
			}
			if in.Notes == nil || *in.Notes == "" {
				t.Errorf("expected notes with reason, got nil/empty")
			}
			return entities.VisitorEntry{ID: "rejected"}, nil
		},
	}
	uc := usecases.CheckinByQR{PreRegRepo: preRepo, BlacklistRepo: bl, EntryRepo: er}
	_, err := uc.Execute(context.Background(), usecases.CheckinByQRInput{
		QRCode:  "RAW",
		GuardID: validUUID,
	})
	mustProblem(t, err, http.StatusForbidden)
	if createCalls != 1 {
		t.Errorf("expected 1 audit create, got %d", createCalls)
	}
}

// --- CheckinManual ---

func TestCheckinManual_Golden(t *testing.T) {
	bl := &fakeBlacklist{getFn: func(_ context.Context, _ entities.DocumentType, _ string) (*entities.BlacklistEntry, error) {
		return nil, nil
	}}
	var captured domain.CreateVisitorEntryInput
	er := &fakeEntries{
		createFn: func(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
			captured = in
			return entities.VisitorEntry{ID: validUUID, Source: in.Source, Status: in.Status}, nil
		},
	}
	uc := usecases.CheckinManual{BlacklistRepo: bl, EntryRepo: er}
	dt := "CC"
	out, err := uc.Execute(context.Background(), usecases.CheckinManualInput{
		VisitorFullName:       "Juan",
		VisitorDocumentType:   &dt,
		VisitorDocumentNumber: "12345",
		PhotoURL:              "https://x/photo.jpg",
		GuardID:               validUUID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if captured.Source != entities.VisitorEntrySourceManual {
		t.Errorf("expected manual source, got %q", captured.Source)
	}
	if out.Status != entities.VisitorEntryStatusActive {
		t.Errorf("got %q want active", out.Status)
	}
}

func TestCheckinManual_PhotoRequired(t *testing.T) {
	uc := usecases.CheckinManual{BlacklistRepo: &fakeBlacklist{}, EntryRepo: &fakeEntries{}}
	_, err := uc.Execute(context.Background(), usecases.CheckinManualInput{
		VisitorFullName:       "Juan",
		VisitorDocumentNumber: "12345",
		PhotoURL:              "",
		GuardID:               validUUID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCheckinManual_Blacklisted(t *testing.T) {
	bl := &fakeBlacklist{
		getFn: func(_ context.Context, _ entities.DocumentType, _ string) (*entities.BlacklistEntry, error) {
			return &entities.BlacklistEntry{
				ID:     "x",
				Reason: "thief",
				Status: entities.BlacklistStatusActive,
			}, nil
		},
	}
	createCalls := 0
	er := &fakeEntries{
		createFn: func(_ context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
			createCalls++
			if in.Status != entities.VisitorEntryStatusRejected {
				t.Errorf("expected rejected, got %q", in.Status)
			}
			return entities.VisitorEntry{ID: "x"}, nil
		},
	}
	uc := usecases.CheckinManual{BlacklistRepo: bl, EntryRepo: er}
	dt := "CC"
	_, err := uc.Execute(context.Background(), usecases.CheckinManualInput{
		VisitorFullName:       "Juan",
		VisitorDocumentType:   &dt,
		VisitorDocumentNumber: "12345",
		PhotoURL:              "https://x/photo.jpg",
		GuardID:               validUUID,
	})
	mustProblem(t, err, http.StatusForbidden)
	if createCalls != 1 {
		t.Errorf("expected audit create, got %d", createCalls)
	}
}

func TestCheckinManual_BadGuardID(t *testing.T) {
	uc := usecases.CheckinManual{BlacklistRepo: &fakeBlacklist{}, EntryRepo: &fakeEntries{}}
	_, err := uc.Execute(context.Background(), usecases.CheckinManualInput{
		VisitorFullName:       "Juan",
		VisitorDocumentNumber: "12345",
		PhotoURL:              "https://x/p.jpg",
		GuardID:               "bad",
	})
	mustProblem(t, err, http.StatusBadRequest)
}

// --- Checkout ---

func TestCheckout_Golden(t *testing.T) {
	er := &fakeEntries{
		closeFn: func(_ context.Context, id, actor string) (entities.VisitorEntry, error) {
			now := time.Now()
			return entities.VisitorEntry{ID: id, ExitTime: &now, Status: entities.VisitorEntryStatusClosed}, nil
		},
	}
	uc := usecases.Checkout{EntryRepo: er}
	out, err := uc.Execute(context.Background(), usecases.CheckoutInput{EntryID: validUUID})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.ExitTime == nil {
		t.Error("expected exit_time set")
	}
}

func TestCheckout_BadID(t *testing.T) {
	uc := usecases.Checkout{EntryRepo: &fakeEntries{}}
	_, err := uc.Execute(context.Background(), usecases.CheckoutInput{EntryID: "bad"})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCheckout_NotFound(t *testing.T) {
	er := &fakeEntries{
		closeFn: func(_ context.Context, _, _ string) (entities.VisitorEntry, error) {
			return entities.VisitorEntry{}, domain.ErrEntryNotFound
		},
	}
	uc := usecases.Checkout{EntryRepo: er}
	_, err := uc.Execute(context.Background(), usecases.CheckoutInput{EntryID: validUUID})
	mustProblem(t, err, http.StatusNotFound)
}

// --- HashQRCode determinism ---

func TestHashQRCode_Deterministic(t *testing.T) {
	a := usecases.HashQRCode("ABC")
	b := usecases.HashQRCode("ABC")
	c := usecases.HashQRCode("XYZ")
	if a != b {
		t.Error("hash must be deterministic")
	}
	if a == c {
		t.Error("hash must differ for different inputs")
	}
	if len(a) != 64 {
		t.Errorf("sha256 hex must be 64 chars, got %d", len(a))
	}
}
