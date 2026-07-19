package usecase

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ITokenService interface {
	ComparePassword(passwordHash, password string) error
	GenerateAccessToken(user *entity.User, now time.Time) (string, int64, error)
	GenerateRefreshToken() (plain string, hash string, error error)
	GenerateCSRFToken() (string, error)
	GenerateFamilyID() uuid.UUID
	RefreshTokenTTL() time.Duration
}

type tokenService struct {
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	random     io.Reader
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
	accessTTL := time.Duration(accessSeconds) * time.Second
	refreshTTL := time.Duration(refreshHours) * time.Hour
	if refreshTTL <= accessTTL {
		return nil, errors.New("REFRESH_TOKEN_TTL_HOURS must be longer than ACCESS_TOKEN_TTL_SECONDS")
	}

	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "music-discovery"
	}
	audience := os.Getenv("JWT_AUDIENCE")
	if audience == "" {
		audience = "music-discovery-frontend"
	}
	return &tokenService{secret: secret, issuer: issuer, audience: audience, accessTTL: accessTTL, refreshTTL: refreshTTL, random: rand.Reader}, nil
}

func (s *tokenService) ComparePassword(hash, password string) error {
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return entity.ErrInvalidCredentials
	}
	return nil
}

func (s *tokenService) GenerateAccessToken(user *entity.User, now time.Time) (string, int64, error) {
	if user == nil {
		return "", 0, entity.ErrInternal
	}
	exp := now.Add(s.accessTTL)
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(map[string]any{
		"iss": s.issuer, "aud": s.audience, "sub": user.ID.String(), "role": user.Role,
		"tv": user.TokenVersion, "iat": now.Unix(), "exp": exp.Unix(),
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

func secureToken(reader io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(reader, b); err != nil {
		return "", entity.ErrInternal
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (s *tokenService) GenerateRefreshToken() (string, string, error) {
	plain, err := secureToken(s.random)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}
func (s *tokenService) GenerateCSRFToken() (string, error) { return secureToken(s.random) }
func (s *tokenService) GenerateFamilyID() uuid.UUID        { return uuid.New() }
func (s *tokenService) RefreshTokenTTL() time.Duration     { return s.refreshTTL }
