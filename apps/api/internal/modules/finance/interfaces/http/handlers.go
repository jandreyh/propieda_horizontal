package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/finance/application/dto"
	"github.com/saas-ph/api/internal/modules/finance/application/usecases"
	"github.com/saas-ph/api/internal/modules/finance/domain"
	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger          *slog.Logger
	Accounts        domain.ChartOfAccountsRepository
	CostCenters     domain.CostCenterRepository
	BillingAccounts domain.BillingAccountRepository
	Charges         domain.ChargeRepository
	Payments        domain.PaymentRepository
	Allocations     domain.PaymentAllocationRepository
	Reversals       domain.PaymentReversalRepository
	Closures        domain.PeriodClosureRepository
	Webhooks        domain.WebhookIdempotencyRepository
	Outbox          domain.OutboxRepository
	TxRunner        usecases.TxRunner
	Now             func() time.Time
}

func (d *Dependencies) validate() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
}

type handlers struct {
	deps Dependencies
}

func newHandlers(d Dependencies) *handlers {
	d.validate()
	return &handlers{deps: d}
}

// --- Chart of Accounts ---

func (h *handlers) createAccount(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateAccountRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateAccount{Accounts: h.deps.Accounts}
	out, err := uc.Execute(r.Context(), usecases.CreateAccountInput{
		Code:        body.Code,
		Name:        body.Name,
		AccountType: entities.AccountType(body.AccountType),
		ParentID:    body.ParentID,
		ActorID:     strPtr(actorID),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, accountToDTO(out))
}

func (h *handlers) listAccounts(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListAccounts{Accounts: h.deps.Accounts}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListAccountsResponse{
		Items: make([]dto.AccountResponse, 0, len(out)),
		Total: len(out),
	}
	for _, a := range out {
		resp.Items = append(resp.Items, accountToDTO(a))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Cost Centers ---

func (h *handlers) createCostCenter(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateCostCenterRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateCostCenter{CostCenters: h.deps.CostCenters}
	out, err := uc.Execute(r.Context(), usecases.CreateCostCenterInput{
		Code:    body.Code,
		Name:    body.Name,
		ActorID: strPtr(actorID),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, costCenterToDTO(out))
}

func (h *handlers) listCostCenters(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListCostCenters{CostCenters: h.deps.CostCenters}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListCostCentersResponse{
		Items: make([]dto.CostCenterResponse, 0, len(out)),
		Total: len(out),
	}
	for _, cc := range out {
		resp.Items = append(resp.Items, costCenterToDTO(cc))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Charges ---

func (h *handlers) createCharge(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateChargeRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	dueDate, err := time.Parse("2006-01-02", body.DueDate)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("due_date: invalid format, expected YYYY-MM-DD"))
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateCharge{
		Charges:         h.deps.Charges,
		BillingAccounts: h.deps.BillingAccounts,
		Outbox:          h.deps.Outbox,
		TxRunner:        h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateChargeInput{
		BillingAccountID: body.BillingAccountID,
		Concept:          entities.ChargeConcept(body.Concept),
		PeriodYear:       body.PeriodYear,
		PeriodMonth:      body.PeriodMonth,
		Amount:           body.Amount,
		DueDate:          dueDate,
		CostCenterID:     body.CostCenterID,
		AccountID:        body.AccountID,
		IdempotencyKey:   body.IdempotencyKey,
		Description:      body.Description,
		ActorID:          strPtr(actorID),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, chargeToDTO(out))
}

func (h *handlers) listCharges(w http.ResponseWriter, r *http.Request) {
	billingAccountID := r.URL.Query().Get("billing_account_id")
	if billingAccountID == "" {
		h.fail(w, r, apperrors.BadRequest("billing_account_id query parameter is required"))
		return
	}
	uc := usecases.ListCharges{Charges: h.deps.Charges}
	out, err := uc.Execute(r.Context(), billingAccountID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListChargesResponse{
		Items: make([]dto.ChargeResponse, 0, len(out)),
		Total: len(out),
	}
	for _, c := range out {
		resp.Items = append(resp.Items, chargeToDTO(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Payments ---

func (h *handlers) createPayment(w http.ResponseWriter, r *http.Request) {
	var body dto.CreatePaymentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreatePayment{
		Payments:        h.deps.Payments,
		BillingAccounts: h.deps.BillingAccounts,
		Outbox:          h.deps.Outbox,
		TxRunner:        h.deps.TxRunner,
		Now:             h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CreatePaymentInput{
		BillingAccountID: body.BillingAccountID,
		PayerUserID:      body.PayerUserID,
		MethodCode:       body.MethodCode,
		Amount:           body.Amount,
		Currency:         body.Currency,
		IdempotencyKey:   body.IdempotencyKey,
		ActorID:          strPtr(actorID),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, paymentToDTO(out))
}

func (h *handlers) listPayments(w http.ResponseWriter, r *http.Request) {
	billingAccountID := r.URL.Query().Get("billing_account_id")
	if billingAccountID == "" {
		h.fail(w, r, apperrors.BadRequest("billing_account_id query parameter is required"))
		return
	}
	uc := usecases.ListPayments{Payments: h.deps.Payments}
	out, err := uc.Execute(r.Context(), billingAccountID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListPaymentsResponse{
		Items: make([]dto.PaymentResponse, 0, len(out)),
		Total: len(out),
	}
	for _, p := range out {
		resp.Items = append(resp.Items, paymentToDTO(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Payment Allocation ---

func (h *handlers) allocatePayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	var body dto.AllocatePaymentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	lines := make([]usecases.AllocationLine, 0, len(body.Allocations))
	for _, a := range body.Allocations {
		lines = append(lines, usecases.AllocationLine{
			ChargeID: a.ChargeID,
			Amount:   a.Amount,
		})
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AllocatePayment{
		Payments:    h.deps.Payments,
		Charges:     h.deps.Charges,
		Allocations: h.deps.Allocations,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.AllocatePaymentInput{
		PaymentID:   paymentID,
		Allocations: lines,
		ActorID:     actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	allocations := make([]dto.AllocationResponse, 0, len(out.Allocations))
	for _, a := range out.Allocations {
		allocations = append(allocations, allocationToDTO(a))
	}
	writeJSON(w, http.StatusOK, dto.AllocatePaymentResponse{
		Payment:     paymentToDTO(out.Payment),
		Allocations: allocations,
	})
}

// --- Payment Reversals ---

func (h *handlers) requestReversal(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	var body dto.RequestReversalRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.RequestReversal{
		Payments:  h.deps.Payments,
		Reversals: h.deps.Reversals,
	}
	out, err := uc.Execute(r.Context(), usecases.RequestReversalInput{
		PaymentID:   paymentID,
		Reason:      body.Reason,
		RequestedBy: actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, reversalToDTO(out))
}

func (h *handlers) approveReversal(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	reversalID := chi.URLParam(r, "rid")
	actorID := actorIDFromCtx(r)
	uc := usecases.ApproveReversal{
		Reversals: h.deps.Reversals,
		Payments:  h.deps.Payments,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.ApproveReversalInput{
		PaymentID:  paymentID,
		ReversalID: reversalID,
		ApprovedBy: actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reversalToDTO(out))
}

// --- Webhook ---

func (h *handlers) processWebhook(w http.ResponseWriter, r *http.Request) {
	gateway := chi.URLParam(r, "gateway")
	var body dto.WebhookPayload
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.ProcessWebhook{
		Webhooks:        h.deps.Webhooks,
		Payments:        h.deps.Payments,
		BillingAccounts: h.deps.BillingAccounts,
		Outbox:          h.deps.Outbox,
		TxRunner:        h.deps.TxRunner,
		Now:             h.deps.Now,
	}
	err := uc.Execute(r.Context(), usecases.WebhookInput{
		Gateway:        gateway,
		TransactionID:  body.TransactionID,
		Amount:         body.Amount,
		Currency:       body.Currency,
		Status:         body.Status,
		IdempotencyKey: body.IdempotencyKey,
		MerchantRef:    body.MerchantRef,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Period Closure ---

func (h *handlers) closePeriodSoft(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	monthStr := chi.URLParam(r, "month")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("year: invalid integer"))
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("month: invalid integer"))
		return
	}

	var body dto.ClosePeriodSoftRequest
	// Body is optional.
	_ = decodeJSON(r, &body)

	actorID := actorIDFromCtx(r)
	uc := usecases.ClosePeriodSoft{
		Closures: h.deps.Closures,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
		Now:      h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ClosePeriodSoftInput{
		PeriodYear:  int32(year),  //nolint:gosec // year already validated in range 1900-2100
		PeriodMonth: int32(month), //nolint:gosec // month already validated in range 1-12
		ActorID:     actorID,
		Notes:       body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, periodClosureToDTO(out))
}

// --- Statement ---

func (h *handlers) getStatement(w http.ResponseWriter, r *http.Request) {
	billingAccountID := chi.URLParam(r, "id")
	uc := usecases.GetStatement{
		BillingAccounts: h.deps.BillingAccounts,
		Charges:         h.deps.Charges,
		Payments:        h.deps.Payments,
	}
	out, err := uc.Execute(r.Context(), billingAccountID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	charges := make([]dto.ChargeResponse, 0, len(out.Charges))
	for _, c := range out.Charges {
		charges = append(charges, chargeToDTO(c))
	}
	payments := make([]dto.PaymentResponse, 0, len(out.Payments))
	for _, p := range out.Payments {
		payments = append(payments, paymentToDTO(p))
	}
	writeJSON(w, http.StatusOK, dto.StatementResponse{
		BillingAccountID: out.BillingAccountID,
		Charges:          charges,
		Payments:         payments,
		TotalCharged:     out.TotalCharged,
		TotalPaid:        out.TotalPaid,
		TotalBalance:     out.TotalBalance,
	})
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "finance: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "finance: unexpected error",
		slog.String("path", r.URL.Path),
		slog.String("err", err.Error()))
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperrors.BadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}

// --- Entity-to-DTO mapping functions ---

func accountToDTO(a entities.ChartOfAccount) dto.AccountResponse {
	return dto.AccountResponse{
		ID:          a.ID,
		Code:        a.Code,
		Name:        a.Name,
		AccountType: string(a.AccountType),
		ParentID:    a.ParentID,
		Status:      string(a.Status),
		CreatedAt:   dto.FormatTime(a.CreatedAt),
		UpdatedAt:   dto.FormatTime(a.UpdatedAt),
		Version:     a.Version,
	}
}

func costCenterToDTO(cc entities.CostCenter) dto.CostCenterResponse {
	return dto.CostCenterResponse{
		ID:        cc.ID,
		Code:      cc.Code,
		Name:      cc.Name,
		Status:    string(cc.Status),
		CreatedAt: dto.FormatTime(cc.CreatedAt),
		UpdatedAt: dto.FormatTime(cc.UpdatedAt),
		Version:   cc.Version,
	}
}

func chargeToDTO(c entities.Charge) dto.ChargeResponse {
	return dto.ChargeResponse{
		ID:               c.ID,
		BillingAccountID: c.BillingAccountID,
		Concept:          string(c.Concept),
		PeriodYear:       c.PeriodYear,
		PeriodMonth:      c.PeriodMonth,
		Amount:           c.Amount,
		Balance:          c.Balance,
		DueDate:          dto.FormatDate(c.DueDate),
		CostCenterID:     c.CostCenterID,
		AccountID:        c.AccountID,
		Description:      c.Description,
		Status:           string(c.Status),
		CreatedAt:        dto.FormatTime(c.CreatedAt),
		UpdatedAt:        dto.FormatTime(c.UpdatedAt),
		Version:          c.Version,
	}
}

func paymentToDTO(p entities.Payment) dto.PaymentResponse {
	return dto.PaymentResponse{
		ID:                p.ID,
		BillingAccountID:  p.BillingAccountID,
		PayerUserID:       p.PayerUserID,
		MethodCode:        p.MethodCode,
		Gateway:           p.Gateway,
		GatewayTxnID:      p.GatewayTxnID,
		Amount:            p.Amount,
		Currency:          p.Currency,
		UnallocatedAmount: p.UnallocatedAmount,
		CapturedAt:        dto.FormatTimePtr(p.CapturedAt),
		ReceiptNumber:     p.ReceiptNumber,
		Status:            string(p.Status),
		CreatedAt:         dto.FormatTime(p.CreatedAt),
		UpdatedAt:         dto.FormatTime(p.UpdatedAt),
		Version:           p.Version,
	}
}

func allocationToDTO(a entities.PaymentAllocation) dto.AllocationResponse {
	return dto.AllocationResponse{
		ID:        a.ID,
		PaymentID: a.PaymentID,
		ChargeID:  a.ChargeID,
		Amount:    a.Amount,
		Status:    a.Status,
		CreatedAt: dto.FormatTime(a.CreatedAt),
	}
}

func reversalToDTO(rv entities.PaymentReversal) dto.ReversalResponse {
	return dto.ReversalResponse{
		ID:          rv.ID,
		PaymentID:   rv.PaymentID,
		Reason:      rv.Reason,
		RequestedBy: rv.RequestedBy,
		RequestedAt: dto.FormatTime(rv.RequestedAt),
		ApprovedBy:  rv.ApprovedBy,
		ApprovedAt:  dto.FormatTimePtr(rv.ApprovedAt),
		Status:      string(rv.Status),
		CreatedAt:   dto.FormatTime(rv.CreatedAt),
		UpdatedAt:   dto.FormatTime(rv.UpdatedAt),
		Version:     rv.Version,
	}
}

func periodClosureToDTO(pc entities.PeriodClosure) dto.PeriodClosureResponse {
	return dto.PeriodClosureResponse{
		ID:           pc.ID,
		PeriodYear:   pc.PeriodYear,
		PeriodMonth:  pc.PeriodMonth,
		ClosedSoftAt: dto.FormatTimePtr(pc.ClosedSoftAt),
		ClosedHardAt: dto.FormatTimePtr(pc.ClosedHardAt),
		ClosedBy:     pc.ClosedBy,
		Notes:        pc.Notes,
		Status:       string(pc.Status),
		CreatedAt:    dto.FormatTime(pc.CreatedAt),
		UpdatedAt:    dto.FormatTime(pc.UpdatedAt),
		Version:      pc.Version,
	}
}

// actorCtxKey clave de contexto para el actor (user_id) que origina la
// peticion.
type actorCtxKey struct{}

// WithActorID es helper para inyectar el actor desde un middleware
// externo (test o capa auth).
func WithActorID(r *http.Request, actorID string) *http.Request {
	if actorID == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), actorCtxKey{}, actorID))
}

func actorIDFromCtx(r *http.Request) string {
	if v, ok := r.Context().Value(actorCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
