package transport

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"user-core-service/internal/domain"
	"user-core-service/internal/repository"
)

type UserHandler struct {
	repo repository.UserRepository
}

type CreateUserReq struct {
	Email string `json: "email"`
	Password string `json: "password"`
}

func NewHander(repo repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid json body"}`, http.StatusBadRequest)
		return
	}

	newUser, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()),  http.StatusBadRequest)
		return
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, `{"error": "failed to generate secure ID"}`, http.StatusInternalServerError)
		return
	}
	newUser.ID = fmt.Sprintf(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))


	if err = h.repo.Save(r.Context(), newUser); err != nil {
		http.Error(w, `{"error": "failed to save user"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newUser)
}
