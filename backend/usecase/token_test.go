package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("random failed") }

func testTokenService() *tokenService {
	return &tokenService{secret: []byte("01234567890123456789012345678901"), issuer: "issuer", audience: "audience", accessTTL: 5 * time.Minute, refreshTTL: 24 * time.Hour, random: strings.NewReader(strings.Repeat("x", 256))}
}

func TestGenerateAccessTokenClaimsAndSignature(t *testing.T) {
	service := testTokenService()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	user := &entity.User{ID: uuid.New(), Role: entity.UserRoleContentAdmin, TokenVersion: 7}
	token, ttl, err := service.GenerateAccessToken(user, now)
	if err != nil || ttl != 300 {
		t.Fatalf("token=%q ttl=%d err=%v", token, ttl, err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("parts=%d", len(parts))
	}
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatal(err)
	}
	checks := map[string]any{"iss": "issuer", "aud": "audience", "sub": user.ID.String(), "role": string(user.Role), "tv": float64(7), "iat": float64(now.Unix()), "exp": float64(now.Add(5 * time.Minute).Unix())}
	for key, want := range checks {
		if claims[key] != want {
			t.Fatalf("%s=%v want=%v", key, claims[key], want)
		}
	}
	mac := hmac.New(sha256.New, service.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if parts[2] != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatal("invalid signature")
	}
}

func TestSecureTokensAndHash(t *testing.T) {
	service := testTokenService()
	plain, hash, err := service.GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if plain == hash {
		t.Fatal("plain token equals hash")
	}
	sum := sha256.Sum256([]byte(plain))
	if hash != hex.EncodeToString(sum[:]) {
		t.Fatal("unexpected hash")
	}
	csrf, err := service.GenerateCSRFToken()
	if err != nil || csrf == "" {
		t.Fatalf("csrf=%q err=%v", csrf, err)
	}
	if service.GenerateFamilyID() == uuid.Nil {
		t.Fatal("nil family id")
	}
}

func TestRandomFailure(t *testing.T) {
	service := testTokenService()
	service.random = failingReader{}
	if _, _, err := service.GenerateRefreshToken(); err == nil {
		t.Fatal("expected refresh error")
	}
	if _, err := service.GenerateCSRFToken(); err == nil {
		t.Fatal("expected csrf error")
	}
}

func TestTokenServiceEnvironmentValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "short")
	if _, err := NewTokenServiceFromEnv(); err == nil {
		t.Fatal("expected short secret error")
	}
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ACCESS_TOKEN_TTL_SECONDS", "0")
	if _, err := NewTokenServiceFromEnv(); err == nil {
		t.Fatal("expected ttl error")
	}
	t.Setenv("ACCESS_TOKEN_TTL_SECONDS", "3600")
	t.Setenv("REFRESH_TOKEN_TTL_HOURS", "1")
	if _, err := NewTokenServiceFromEnv(); err == nil {
		t.Fatal("expected refresh ttl error")
	}
}

func TestComparePassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	service := testTokenService()
	if err := service.ComparePassword(string(hash), "password"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(service.ComparePassword(string(hash), "wrongpass"), entity.ErrInvalidCredentials) {
		t.Fatal("expected invalid credentials")
	}
}
