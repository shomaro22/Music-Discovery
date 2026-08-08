package usecase

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ITokenService interface {
	ComparePassword(passwordHash, password string) error
	GenerateAccessToken(user *entity.User, now time.Time) (string, int64, error)
	GenerateRefreshToken() (plain string, hash string, error error)
	HashRefreshToken(token string) string
	GenerateCSRFToken() (string, error)
	GenerateFamilyID() uuid.UUID
	RefreshTokenTTL() time.Duration
	VerifyAccessToken(token string, now time.Time) (uuid.UUID, entity.Role, uint64, error)
}

type accessTokenClaims struct {
	Issuer       string          `json:"iss"`
	Audience     string          `json:"aud"`
	Subject      string          `json:"sub"`
	Role         entity.UserRole `json:"role"`
	TokenVersion uint64          `json:"tv"`
	IssuedAt     int64           `json:"iat"`
	ExpiresAt    int64           `json:"exp"`
}

type tokenService struct {
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenServiceFromEnv() (ITokenService, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	accessSeconds := int64(300)
	if raw := os.Getenv("ACCESS_TOKEN_TTL_SECONDS"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, errors.New("invalid ACCESS_TOKEN_TTL_SECONDS")
		}
		accessSeconds = value
	}
	refreshHours := int64(24 * 30)
	if raw := os.Getenv("REFRESH_TOKEN_TTL_HOURS"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, errors.New("invalid REFRESH_TOKEN_TTL_HOURS")
		}
		refreshHours = value
	}
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "music-discovery"
	}
	audience := os.Getenv("JWT_AUDIENCE")
	if audience == "" {
		audience = "music-discovery-frontend"
	}
	return &tokenService{secret: secret, issuer: issuer, audience: audience, accessTTL: time.Duration(accessSeconds) * time.Second, refreshTTL: time.Duration(refreshHours) * time.Hour}, nil
}

func (s *tokenService) ComparePassword(hash, password string) error {
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return entity.ErrInvalidCredentials
	}
	return nil
}

func (s *tokenService) GenerateAccessToken(user *entity.User, now time.Time) (string, int64, error) {
	if user == nil || !user.Role.IsPersistable() {
		return "", 0, entity.ErrInternal
	}
	exp := now.Add(s.accessTTL)
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(accessTokenClaims{
		Issuer:       s.issuer,
		Audience:     s.audience,
		Subject:      user.ID.String(),
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		IssuedAt:     now.Unix(),
		ExpiresAt:    exp.Unix(),
	})
	if err != nil {
		return "", 0, entity.ErrInternal
	}
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString(header) + "." + enc.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil)), int64(s.accessTTL.Seconds()), nil
}

func secureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", entity.ErrInternal
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (s *tokenService) GenerateRefreshToken() (string, string, error) {
	plain, err := secureToken()
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}
func (s *tokenService) HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *tokenService) GenerateCSRFToken() (string, error) { return secureToken() }
func (s *tokenService) GenerateFamilyID() uuid.UUID        { return uuid.New() }
func (s *tokenService) RefreshTokenTTL() time.Duration     { return s.refreshTTL }

func (s *tokenService) VerifyAccessToken(token string, now time.Time) (uuid.UUID, entity.Role, uint64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, "", 0, entity.ErrInvalidCredentials
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return uuid.Nil, "", 0, entity.ErrInvalidCredentials
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, "", 0, entity.ErrInvalidCredentials
	}
	var payload struct {
		Subject      string      `json:"sub"`
		Role         entity.Role `json:"role"`
		TokenVersion *uint64     `json:"tv"`
		ExpiresAt    int64       `json:"exp"`
		Issuer       string      `json:"iss"`
		Audience     string      `json:"aud"`
	}
	if json.Unmarshal(payloadBytes, &payload) != nil || payload.TokenVersion == nil || payload.Issuer != s.issuer || payload.Audience != s.audience || !payload.Role.IsPersistable() || now.Unix() >= payload.ExpiresAt {
		return uuid.Nil, "", 0, entity.ErrInvalidCredentials
	}
	userID, err := uuid.Parse(payload.Subject)
	if err != nil {
		return uuid.Nil, "", 0, entity.ErrInvalidCredentials
	}
	return userID, payload.Role, *payload.TokenVersion, nil
}
