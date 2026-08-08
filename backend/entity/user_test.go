package entity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserSignupDefaultsAndJSON(t *testing.T) {
	user := User{
		ID:           uuid.New(),
		Name:         "Taro",
		Email:        "taro@example.com",
		PasswordHash: "secret-hash",
		Role:         UserRoleUser,
		Status:       UserStatusActive,
		TokenVersion: 0,
		CreatedAt:    time.Now().UTC(),
	}

	if user.ID == uuid.Nil || user.Role != UserRoleUser || user.Status != UserStatusActive || user.TokenVersion != 0 {
		t.Fatalf("unexpected signup defaults: %#v", user)
	}

	body, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "secret-hash") || strings.Contains(string(body), "password_hash") {
		t.Fatalf("password hash leaked in JSON: %s", body)
	}
}
