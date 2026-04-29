package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/finance/domain"
	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
	financehttp "github.com/saas-ph/api/internal/modules/finance/interfaces/http"
)

// --- in-memory fakes ---

type stubAccounts struct {
	accounts []entities.ChartOfAccount
}

func (s *stubAccounts) Create(_ context.Context, in domain.CreateAccountInput) (entities.ChartOfAccount, error) {
	for _, a := range s.accounts {
		if a.Code == in.Code {
			return entities.ChartOfAccount{}, domain.ErrAccountCodeDuplicate
		}
	}
	now := time.Now()
	acct := entities.ChartOfAccount{
		ID:          "11111111-1111-1111-1111-111111111111",
		Code:        in.Code,
		Name:        in.Name,
		AccountType: in.AccountType,
		ParentID:    in.ParentID,
		Status:      entities.AccountStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}
	s.accounts = append(s.accounts, acct)
	return acct, nil
}

func (s *stubAccounts) GetByID(_ context.Context, id string) (entities.ChartOfAccount, error) {
	for _, a := range s.accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return entities.ChartOfAccount{}, domain.ErrAccountNotFound
}

func (s *stubAccounts) List(_ context.Context) ([]entities.ChartOfAccount, error) {
	return s.accounts, nil
}

type stubCostCenters struct {
	centers []entities.CostCenter
}

func (s *stubCostCenters) Create(_ context.Context, in domain.CreateCostCenterInput) (entities.CostCenter, error) {
	for _, cc := range s.centers {
		if cc.Code == in.Code {
			return entities.CostCenter{}, domain.ErrCostCenterCodeDuplicate
		}
	}
	now := time.Now()
	cc := entities.CostCenter{
		ID:        "22222222-2222-2222-2222-222222222222",
		Code:      in.Code,
		Name:      in.Name,
		Status:    entities.CostCenterStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
	s.centers = append(s.centers, cc)
	return cc, nil
}

func (s *stubCostCenters) GetByID(_ context.Context, id string) (entities.CostCenter, error) {
	for _, cc := range s.centers {
		if cc.ID == id {
			return cc, nil
		}
	}
	return entities.CostCenter{}, domain.ErrCostCenterNotFound
}

func (s *stubCostCenters) List(_ context.Context) ([]entities.CostCenter, error) {
	return s.centers, nil
}

type stubBillingAccounts struct {
	accounts []entities.BillingAccount
}

func (s *stubBillingAccounts) GetByID(_ context.Context, id string) (entities.BillingAccount, error) {
	for _, ba := range s.accounts {
		if ba.ID == id {
			return ba, nil
		}
	}
	return entities.BillingAccount{}, domain.ErrBillingAccountNotFound
}

func (s *stubBillingAccounts) ListByUnitID(_ context.Context, _ string) ([]entities.BillingAccount, error) {
	return s.accounts, nil
}

type stubCharges struct {
	charges []entities.Charge
}

func (s *stubCharges) Create(_ context.Context, in domain.CreateChargeInput) (entities.Charge, error) {
	now := time.Now()
	charge := entities.Charge{
		ID:               "33333333-3333-3333-3333-333333333333",
		BillingAccountID: in.BillingAccountID,
		Concept:          in.Concept,
		PeriodYear:       in.PeriodYear,
		PeriodMonth:      in.PeriodMonth,
		Amount:           in.Amount,
		Balance:          in.Amount,
		DueDate:          in.DueDate,
		Status:           entities.ChargeStatusOpen,
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          1,
	}
	s.charges = append(s.charges, charge)
	return charge, nil
}

func (s *stubCharges) GetByID(_ context.Context, id string) (entities.Charge, error) {
	for _, c := range s.charges {
		if c.ID == id {
			return c, nil
		}
	}
	return entities.Charge{}, domain.ErrChargeNotFound
}

func (s *stubCharges) ListByBillingAccountID(_ context.Context, baID string) ([]entities.Charge, error) {
	var result []entities.Charge
	for _, c := range s.charges {
		if c.BillingAccountID == baID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *stubCharges) UpdateBalance(_ context.Context, id string, newBalance float64, expectedVersion int32, _ string) (entities.Charge, error) {
	for i, c := range s.charges {
		if c.ID == id {
			if c.Version != expectedVersion {
				return entities.Charge{}, domain.ErrVersionConflict
			}
			s.charges[i].Balance = newBalance
			if newBalance <= 0 {
				s.charges[i].Status = entities.ChargeStatusPaid
			} else if newBalance < c.Amount {
				s.charges[i].Status = entities.ChargeStatusPartial
			}
			s.charges[i].Version++
			return s.charges[i], nil
		}
	}
	return entities.Charge{}, domain.ErrChargeNotFound
}

type stubPayments struct {
	payments []entities.Payment
}

func (s *stubPayments) Create(_ context.Context, in domain.CreatePaymentInput) (entities.Payment, error) {
	now := time.Now()
	payment := entities.Payment{
		ID:                "44444444-4444-4444-4444-444444444444",
		BillingAccountID:  in.BillingAccountID,
		PayerUserID:       in.PayerUserID,
		MethodCode:        in.MethodCode,
		Amount:            in.Amount,
		Currency:          in.Currency,
		UnallocatedAmount: in.Amount,
		CapturedAt:        &now,
		Status:            in.Status,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           1,
	}
	s.payments = append(s.payments, payment)
	return payment, nil
}

func (s *stubPayments) GetByID(_ context.Context, id string) (entities.Payment, error) {
	for _, p := range s.payments {
		if p.ID == id {
			return p, nil
		}
	}
	return entities.Payment{}, domain.ErrPaymentNotFound
}

func (s *stubPayments) GetByIdempotencyKey(_ context.Context, _ string) (entities.Payment, error) {
	return entities.Payment{}, domain.ErrPaymentNotFound
}

func (s *stubPayments) ListByBillingAccountID(_ context.Context, baID string) ([]entities.Payment, error) {
	var result []entities.Payment
	for _, p := range s.payments {
		if p.BillingAccountID == baID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *stubPayments) UpdateUnallocated(_ context.Context, id string, newUnallocated float64, expectedVersion int32, _ string) (entities.Payment, error) {
	for i, p := range s.payments {
		if p.ID == id {
			if p.Version != expectedVersion {
				return entities.Payment{}, domain.ErrVersionConflict
			}
			s.payments[i].UnallocatedAmount = newUnallocated
			s.payments[i].Version++
			return s.payments[i], nil
		}
	}
	return entities.Payment{}, domain.ErrPaymentNotFound
}

func (s *stubPayments) UpdateStatus(_ context.Context, id string, newStatus entities.PaymentStatus, expectedVersion int32, _ string) (entities.Payment, error) {
	for i, p := range s.payments {
		if p.ID == id {
			if p.Version != expectedVersion {
				return entities.Payment{}, domain.ErrVersionConflict
			}
			s.payments[i].Status = newStatus
			s.payments[i].Version++
			return s.payments[i], nil
		}
	}
	return entities.Payment{}, domain.ErrPaymentNotFound
}

type stubAllocations struct{}

func (s *stubAllocations) Create(_ context.Context, in domain.CreateAllocationInput) (entities.PaymentAllocation, error) {
	return entities.PaymentAllocation{
		ID:        "55555555-5555-5555-5555-555555555555",
		PaymentID: in.PaymentID,
		ChargeID:  in.ChargeID,
		Amount:    in.Amount,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}, nil
}

func (s *stubAllocations) ListByPaymentID(_ context.Context, _ string) ([]entities.PaymentAllocation, error) {
	return nil, nil
}

func (s *stubAllocations) ListByChargeID(_ context.Context, _ string) ([]entities.PaymentAllocation, error) {
	return nil, nil
}

type stubReversals struct{}

func (s *stubReversals) Create(_ context.Context, in domain.CreateReversalInput) (entities.PaymentReversal, error) {
	return entities.PaymentReversal{
		ID:          "66666666-6666-6666-6666-666666666666",
		PaymentID:   in.PaymentID,
		Reason:      in.Reason,
		RequestedBy: in.RequestedBy,
		RequestedAt: time.Now(),
		Status:      entities.ReversalStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}, nil
}

func (s *stubReversals) GetByID(_ context.Context, _ string) (entities.PaymentReversal, error) {
	return entities.PaymentReversal{}, domain.ErrReversalNotFound
}

func (s *stubReversals) Approve(_ context.Context, _ string, _ string, _ int32) (entities.PaymentReversal, error) {
	return entities.PaymentReversal{}, domain.ErrReversalNotFound
}

type stubClosures struct{}

func (s *stubClosures) CreateOrGetSoftClosure(_ context.Context, in domain.CreatePeriodClosureInput) (entities.PeriodClosure, error) {
	now := time.Now()
	return entities.PeriodClosure{
		ID:           "77777777-7777-7777-7777-777777777777",
		PeriodYear:   in.PeriodYear,
		PeriodMonth:  in.PeriodMonth,
		ClosedSoftAt: &now,
		Status:       entities.PeriodClosureStatusClosedSoft,
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      1,
	}, nil
}

func (s *stubClosures) GetByPeriod(_ context.Context, _, _ int32) (entities.PeriodClosure, error) {
	return entities.PeriodClosure{}, domain.ErrPeriodClosureNotFound
}

type stubWebhooks struct{}

func (s *stubWebhooks) Create(_ context.Context, in domain.CreateWebhookInput) (entities.WebhookIdempotency, error) {
	return entities.WebhookIdempotency{
		ID:             "88888888-8888-8888-8888-888888888888",
		Gateway:        in.Gateway,
		IdempotencyKey: in.IdempotencyKey,
		ReceivedAt:     time.Now(),
	}, nil
}

func (s *stubWebhooks) MarkProcessed(_ context.Context, _, _ string) error { return nil }
func (s *stubWebhooks) MarkFailed(_ context.Context, _, _ string) error    { return nil }

type stubOutbox struct{}

func (s *stubOutbox) Enqueue(_ context.Context, _ domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	return entities.OutboxEvent{}, nil
}
func (s *stubOutbox) LockPending(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
	return nil, nil
}
func (s *stubOutbox) MarkDelivered(_ context.Context, _ string) error        { return nil }
func (s *stubOutbox) MarkFailed(_ context.Context, _, _ string, _ int) error { return nil }

// --- test billing account for charge/payment tests ---

var testBillingAccount = entities.BillingAccount{
	ID:           "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	UnitID:       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	HolderUserID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
	OpenedAt:     time.Now(),
	Status:       entities.BillingAccountStatusActive,
	CreatedAt:    time.Now(),
	UpdatedAt:    time.Now(),
	Version:      1,
}

func mountTest(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	financehttp.Mount(r, financehttp.Dependencies{
		Accounts:        &stubAccounts{},
		CostCenters:     &stubCostCenters{},
		BillingAccounts: &stubBillingAccounts{accounts: []entities.BillingAccount{testBillingAccount}},
		Charges:         &stubCharges{},
		Payments:        &stubPayments{},
		Allocations:     &stubAllocations{},
		Reversals:       &stubReversals{},
		Closures:        &stubClosures{},
		Webhooks:        &stubWebhooks{},
		Outbox:          &stubOutbox{},
	})
	return r
}

// TestCreateAccount_Success verifica que POST /chart-of-accounts con datos
// validos devuelve 201.
func TestCreateAccount_Success(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"code": "1.1",
		"name": "Caja General",
		"account_type": "asset"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/chart-of-accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["code"] != "1.1" {
		t.Errorf("expected code 1.1, got %v", resp["code"])
	}
	if resp["status"] != "active" {
		t.Errorf("expected status active, got %v", resp["status"])
	}
}

// TestCreateAccount_InvalidCode verifica que POST /chart-of-accounts con
// codigo vacio devuelve 400.
func TestCreateAccount_InvalidCode(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"code": "",
		"name": "Caja General",
		"account_type": "asset"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/chart-of-accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("expected problem+json, got %q", ct)
	}
}

// TestCreateCostCenter_Success verifica que POST /cost-centers con datos
// validos devuelve 201.
func TestCreateCostCenter_Success(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"code": "CC-001",
		"name": "Torre A"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/cost-centers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestCreateCharge_Success verifica que POST /charges con datos validos
// devuelve 201.
func TestCreateCharge_Success(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"billing_account_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"concept": "admin_fee",
		"amount": 150000,
		"due_date": "2026-05-15"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/charges", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["concept"] != "admin_fee" {
		t.Errorf("expected concept admin_fee, got %v", resp["concept"])
	}
	if resp["status"] != "open" {
		t.Errorf("expected status open, got %v", resp["status"])
	}
}

// TestCreatePayment_Success verifica que POST /payments con datos validos
// devuelve 201.
func TestCreatePayment_Success(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"billing_account_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"method_code": "cash",
		"amount": 150000,
		"currency": "COP"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "captured" {
		t.Errorf("expected status captured, got %v", resp["status"])
	}
}

// TestClosePeriodSoft_Success verifica que POST /periods/2026/4/close-soft
// devuelve 200.
func TestClosePeriodSoft_Success(t *testing.T) {
	r := mountTest(t)
	req := httptest.NewRequest(http.MethodPost, "/periods/2026/4/close-soft", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "closed_soft" {
		t.Errorf("expected status closed_soft, got %v", resp["status"])
	}
}

// TestWebhook_Idempotent verifica que POST /payments/webhook/{gateway}
// devuelve 200.
func TestWebhook_Idempotent(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"transaction_id": "txn-123",
		"amount": 100000,
		"currency": "COP",
		"status": "approved",
		"idempotency_key": "wh-key-001",
		"merchant_ref": "ref-001"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/payments/webhook/pse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
