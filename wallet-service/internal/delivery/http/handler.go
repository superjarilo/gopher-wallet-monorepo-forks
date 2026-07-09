package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Описываем контракт бизнес-логики (Use Case) для хендлера
type WalletUseCase interface {
	Deposit(ctx context.Context, userID string, amount int64) error
	Debit(ctx context.Context, userID string, amount int64) error
}

type Handler struct {
	uc WalletUseCase
}

func NewHandler(uc WalletUseCase) *Handler {
	return &Handler{uc: uc}
}

// Routes регистрирует эндпоинты в роутере Chi
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/deposit", h.HandleDeposit)
	r.Post("/debit", h.HandleDebit)
	return r
}

type TxRequest struct {
	UserID string `json:"user_id"`
	Amount int64  `json:"amount"` // Передаем строго в копейках
}

func (h *Handler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	var req TxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.uc.Deposit(r.Context(), req.UserID, req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"deposit processed"}`))
}

func (h *Handler) HandleDebit(w http.ResponseWriter, r *http.Request) {
	var req TxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.uc.Debit(r.Context(), req.UserID, req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"debit processed"}`))
}
