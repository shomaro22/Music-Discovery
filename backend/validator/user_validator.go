package validator

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

type SignUpValues struct {
	Name     string
	Email    string
	Password string
}

type ValidationErrors map[string]string

func ValidateSignUp(
	name string,
	email string,
	password string,
) (SignUpValues, ValidationErrors) {
	normalizedName := strings.TrimSpace(name)
	normalizedEmail := strings.ToLower(
		strings.TrimSpace(email),
	)

	validationErrors := ValidationErrors{}

	nameLength := utf8.RuneCountInString(normalizedName)
	if nameLength < 1 || nameLength > 50 {
		validationErrors["name"] =
			"名前は1文字以上50文字以内で入力してください"
	}

	if normalizedEmail == "" || len(normalizedEmail) > 254 {
		validationErrors["email"] =
			"メールアドレスを254文字以内で入力してください"
	} else {
		address, err := mail.ParseAddress(normalizedEmail)
		if err != nil || address.Address != normalizedEmail {
			validationErrors["email"] =
				"有効なメールアドレスを入力してください"
		}
	}

	passwordCharacterLength := utf8.RuneCountInString(password)
	passwordByteLength := len([]byte(password))

	if passwordCharacterLength < 8 {
		validationErrors["password"] =
			"パスワードは8文字以上で入力してください"
	} else if passwordByteLength > 72 {
		validationErrors["password"] =
			"パスワードが長すぎます"
	}

	return SignUpValues{
		Name:     normalizedName,
		Email:    normalizedEmail,
		Password: password,
	}, validationErrors
}
