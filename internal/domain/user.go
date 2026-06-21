package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
}

func NewUser(email, rawPassword string) (*User, error) {
	if email == "" {
		return nil, errors.New("[Constructor] Email cannot be empty")
	}

	validateEmail := strings.Contains(email, "@")
	if !validateEmail {
		return nil, errors.New("[Constructor] Invalid email format")
	}

	if len(rawPassword) < 8 {
		return nil, errors.New("[Constructor] Password must be at least 8 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("[Constructor] Failed to hash password: %w", err)
	}

	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	newID := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return &User{
		ID:           newID,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}, nil
}
