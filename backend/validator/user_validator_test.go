package validator

import (
	"strings"
	"testing"
)

func TestValidateSignUpNameBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", true},
		{"one", "a", false},
		{"fifty", strings.Repeat("a", 50), false},
		{"fifty-one", strings.Repeat("a", 51), true},
		{"trimmed-one", " a ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := ValidateSignUp(tt.value, "user@example.com", "password")
			_, exists := errs["name"]
			if exists != tt.wantErr {
				t.Fatalf("name error = %v, want %v: %#v", exists, tt.wantErr, errs)
			}
		})
	}
}

func TestValidateSignUpEmailNormalizationAndBoundaries(t *testing.T) {
	values, errs := ValidateSignUp("Taro", "  TARO@EXAMPLE.COM  ", "password")
	if len(errs) != 0 || values.Email != "taro@example.com" {
		t.Fatalf("unexpected result: values=%#v errors=%#v", values, errs)
	}

	for _, email := range []string{"", "not-an-email", strings.Repeat("a", 245) + "@example.com"} {
		_, errs := ValidateSignUp("Taro", email, "password")
		if _, ok := errs["email"]; !ok {
			t.Fatalf("expected email error for length %d", len(email))
		}
	}
}

func TestValidateSignUpPasswordBoundariesAndNoTrim(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"seven", "1234567", true},
		{"eight", "12345678", false},
		{"seventy-two-bytes", strings.Repeat("a", 72), false},
		{"seventy-three-bytes", strings.Repeat("a", 73), true},
		{"spaces-are-data", " 123456 ", false},
		{"multibyte-within-limit", strings.Repeat("あ", 24), false},
		{"multibyte-over-limit", strings.Repeat("あ", 25), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, errs := ValidateSignUp("Taro", "user@example.com", tt.value)
			_, exists := errs["password"]
			if exists != tt.wantErr {
				t.Fatalf("password error = %v, want %v: %#v", exists, tt.wantErr, errs)
			}
			if values.Password != tt.value {
				t.Fatalf("password was normalized: got %q want %q", values.Password, tt.value)
			}
		})
	}
}
