package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"user-core-service/internal/domain"
	"user-core-service/internal/repository"
)

func RegisterUserRoutes(mux *http.ServeMux, h *UserHandler) {
	mux.HandleFunc("GET /health", h.DeepHealthCheck)
	mux.HandleFunc("POST /v1/users", h.CreateUser)
	mux.HandleFunc("GET /v1/users", h.GetByEmail)
}

type UserHandler struct {
	repo repository.UserRepository
}

type CreateUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type HealthRes struct {
	Status   string `json:"status"`
	DBShards string `json:"dbshards"`
	Cache    string `json:"cache"`
}

func NewHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()

	var req CreateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	newUser, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = h.repo.Save(r.Context(), newUser); err != nil {
		if strings.Contains(err.Error(), "23505") {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.ErrorContext(
			r.Context(), "database save execution failed",
			"user_id", newUser.ID,
			"error", err,
		)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, UserRes{ID: newUser.ID, Email: newUser.Email})
}

func (h *UserHandler) GetByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		respondError(w, http.StatusBadRequest, "email query parameter is required")
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), email)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "database lookup failed",
			"email", email,
			"error", err,
		)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if user == nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, UserRes{ID: user.ID, Email: user.Email})
}

func (h *UserHandler) DeepHealthCheck(w http.ResponseWriter, r *http.Request) {
	healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	isHealthy := true
	res := HealthRes{
		Status:   "UP",
		DBShards: "OK",
		Cache:    "OK",
	}

	if err := h.repo.PingDB(healthCtx); err != nil {
		isHealthy = false
		res.DBShards = err.Error()
		slog.ErrorContext(healthCtx, "healthcheck failure: postgres shard down", "error", err)
	}

	if err := h.repo.PingCache(healthCtx); err != nil {
		isHealthy = false
		res.Cache = err.Error()
		slog.ErrorContext(healthCtx, "healthcheck failure: redis manager down", "error", err)
	}

	if !isHealthy {
		res.Status = "DOWN"
		respondJSON(w, http.StatusServiceUnavailable, res)
		return
	}

	respondJSON(w, http.StatusOK, res)
}
