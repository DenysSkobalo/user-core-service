package transport

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"user-core-service/internal/domain"
	"user-core-service/internal/repository"
)

type UserHandler struct {
	repo repository.UserRepository
}

type CreateUserReq struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
	Email string `json:"email"`
}

type UserResponse struct {
	ID string `json:"id"`
	Email string `json:"email"`
}

func NewHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method !=  http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error": "method not allowed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req CreateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_, _ = w.Write([]byte(`{"error":"invalid json body"}`))
		return
	}

	newUser, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		slog.Error("failed to generate cryptographically secure UUID", "error",  err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		return
	}
	newUser.ID = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	if err = h.repo.Save(r.Context(), newUser); err != nil {
		slog.ErrorContext(r.Context(), "database save execution failed", 
			"user_id", newUser.ID, 
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		return
	}

	responseDto := CreateUserResponse{
		ID: newUser.ID,
		Email: newUser.Email,
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(responseDto)
}

func (h *UserHandler) GetByEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method !=  http.MethodGet {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error": "method not allowed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"email query parameter is required"}`))
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), email)
	if err != nil {
		slog.ErrorContext(r.Context(), "database lookup failed", 
			"email", email, 
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		return
	}

	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"user not found"}`))
		return
	}
	
	responseDto := UserResponse{
		ID: user.ID,
		Email: user.Email,
	}

	_ = json.NewEncoder(w).Encode(responseDto) 
}
