package domain

import (
	"errors"
	"strings"
)

type User struct {
	ID string
	Email string
	PasswordHash string
}

func NewUser (email, rawPassord string) (*User, error) {
	if email == "" {
		return nil, errors.New("[Constructor] Email cannot be empty")
	}
	
	validateEmail := strings.Contains(email, "@")
	if !validateEmail {
		return nil, errors.New("[Constructor] Invalid email format")
	}

	if len(rawPassord) < 8 {
		return nil, errors.New("[Constructor] Password must be at least 8 characters long")
	}
	
	return &User{
		Email: email,
		PasswordHash: rawPassord,
	}, nil
}
