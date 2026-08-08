package validator

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

type SignUpValues struct{ Name, Email, Password string }
type LoginValues struct{ Email, Password string }
type ValidationErrors map[string]string

type IUserValidator interface {
	ValidateSignUp(name, email, password string) (SignUpValues, ValidationErrors)
	ValidateLogin(email, password string) (LoginValues, ValidationErrors)
}

type userValidator struct{}

func NewUserValidator() IUserValidator { return &userValidator{} }

func (v *userValidator) ValidateSignUp(name, email, password string) (SignUpValues, ValidationErrors) {
	return ValidateSignUp(name, email, password)
}
func (v *userValidator) ValidateLogin(email, password string) (LoginValues, ValidationErrors) {
	return ValidateLogin(email, password)
}

func ValidateSignUp(name, email, password string) (SignUpValues, ValidationErrors) {
	normalizedName := strings.TrimSpace(name)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	validationErrors := ValidationErrors{}
	if n := utf8.RuneCountInString(normalizedName); n < 1 || n > 50 {
		validationErrors["name"] = "名前は1文字以上50文字以内で入力してください"
	}
	if normalizedEmail == "" || len(normalizedEmail) > 254 {
		validationErrors["email"] = "メールアドレスを254文字以内で入力してください"
	} else if address, err := mail.ParseAddress(normalizedEmail); err != nil || address.Address != normalizedEmail {
		validationErrors["email"] = "有効なメールアドレスを入力してください"
	}
	if utf8.RuneCountInString(password) < 8 {
		validationErrors["password"] = "パスワードは8文字以上で入力してください"
	} else if len([]byte(password)) > 72 {
		validationErrors["password"] = "パスワードが長すぎます"
	}
	return SignUpValues{Name: normalizedName, Email: normalizedEmail, Password: password}, validationErrors
}

func ValidateLogin(email, password string) (LoginValues, ValidationErrors) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	validationErrors := ValidationErrors{}
	if normalizedEmail == "" || len(normalizedEmail) > 254 {
		validationErrors["email"] = "メールアドレスを254文字以内で入力してください"
	} else if address, err := mail.ParseAddress(normalizedEmail); err != nil || address.Address != normalizedEmail {
		validationErrors["email"] = "有効なメールアドレスを入力してください"
	}
	if utf8.RuneCountInString(password) < 8 {
		validationErrors["password"] = "パスワードは8文字以上で入力してください"
	} else if len([]byte(password)) > 72 {
		validationErrors["password"] = "パスワードが長すぎます"
	}
	return LoginValues{Email: normalizedEmail, Password: password}, validationErrors
}
