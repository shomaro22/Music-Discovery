package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

type fakeUserRepository struct {
	findUser      *entity.User
	findErr       error
	createUser    *entity.User
	createErr     error
	createInputID uuid.UUID
	findCalls     int
	createCalls   int
}

func (f *fakeUserRepository) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return f.findUser, f.findErr
}
func (f *fakeUserRepository) FindByEmail(context.Context, string) (*entity.User, error) {
	f.findCalls++
	return f.findUser, f.findErr
}
func (f *fakeUserRepository) Create(_ context.Context, user *entity.User) (*entity.User, error) {
	f.createCalls++
	f.createInputID = user.ID
	f.createUser = user
	if f.createErr != nil {
		return nil, f.createErr
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return user, nil
}

type fakeHasher struct {
	hash  string
	err   error
	input string
	calls int
}

func (f *fakeHasher) Hash(value string) (string, error) {
	f.calls++
	f.input = value
	return f.hash, f.err
}

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time { return f.now }

func newUsecaseForTest(repo *fakeUserRepository, hasher *fakeHasher) IUserUsecase {
	return NewUserUsecase(repo, hasher, &fakeClock{now: time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)})
}

func TestSignUpSuccessSetsInternalFields(t *testing.T) {
	repo := &fakeUserRepository{findErr: entity.ErrUserNotFound}
	hasher := &fakeHasher{hash: "hashed"}
	user, err := newUsecaseForTest(repo, hasher).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: " password "})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == uuid.Nil || user.Role != entity.UserRoleUser || user.Status != entity.UserStatusActive || user.TokenVersion != 0 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if repo.createInputID != uuid.Nil {
		t.Fatalf("usecase generated user ID: %s", repo.createInputID)
	}
	if hasher.input != " password " {
		t.Fatalf("password was trimmed: %q", hasher.input)
	}
	if repo.createCalls != 1 || user.PasswordHash != "hashed" {
		t.Fatalf("unexpected calls/user: %#v", user)
	}
}

func TestSignUpEmailAlreadyExistsDoesNotHashOrSave(t *testing.T) {
	repo := &fakeUserRepository{findUser: &entity.User{ID: uuid.New()}}
	hasher := &fakeHasher{hash: "hashed"}
	_, err := newUsecaseForTest(repo, hasher).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: "password"})
	if !errors.Is(err, entity.ErrEmailAlreadyExists) {
		t.Fatalf("err=%v", err)
	}
	if hasher.calls != 0 || repo.createCalls != 0 {
		t.Fatalf("unexpected calls: hash=%d create=%d", hasher.calls, repo.createCalls)
	}
}

func TestSignUpRepositoryFindError(t *testing.T) {
	repo := &fakeUserRepository{findErr: errors.New("db down")}
	_, err := newUsecaseForTest(repo, &fakeHasher{}).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: "password"})
	if !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
}

func TestSignUpHashFailureDoesNotSave(t *testing.T) {
	repo := &fakeUserRepository{findErr: entity.ErrUserNotFound}
	hasher := &fakeHasher{err: errors.New("hash failed")}
	_, err := newUsecaseForTest(repo, hasher).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: "password"})
	if !errors.Is(err, entity.ErrInternal) || repo.createCalls != 0 {
		t.Fatalf("err=%v create=%d", err, repo.createCalls)
	}
}

func TestSignUpSaveFailure(t *testing.T) {
	for _, saveErr := range []error{entity.ErrInternal, entity.ErrEmailAlreadyExists} {
		t.Run(saveErr.Error(), func(t *testing.T) {
			repo := &fakeUserRepository{findErr: entity.ErrUserNotFound, createErr: saveErr}
			_, err := newUsecaseForTest(repo, &fakeHasher{hash: "hashed"}).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: "password"})
			if !errors.Is(err, saveErr) {
				t.Fatalf("err=%v want=%v", err, saveErr)
			}
		})
	}
}

type fakeRefreshRepository struct {
	input *entity.RefreshToken
	err   error
	calls int
}

func (f *fakeRefreshRepository) FindByTokenHash(context.Context, string) (*entity.RefreshToken, error) {
	return f.input, f.err
}
func (f *fakeRefreshRepository) Rotate(context.Context, string, *entity.RefreshToken, time.Time) error {
	return f.err
}
func (f *fakeRefreshRepository) RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return f.err
}
func (f *fakeRefreshRepository) Create(_ context.Context, token *entity.RefreshToken) error {
	f.calls++
	f.input = token
	return f.err
}

type fakeTokenService struct {
	compareErr      error
	accessErr       error
	refreshErr      error
	csrfErr         error
	compareHash     string
	comparePassword string
	compareCalls    int
	accessToken     string
	expiresIn       int64
	refreshPlain    string
	refreshHash     string
	csrf            string
	familyID        uuid.UUID
	refreshTTL      time.Duration
	accessUser      *entity.User
}

func (f *fakeTokenService) ComparePassword(hash, password string) error {
	f.compareCalls++
	f.compareHash = hash
	f.comparePassword = password
	return f.compareErr
}
func (f *fakeTokenService) GenerateAccessToken(user *entity.User, _ time.Time) (string, int64, error) {
	f.accessUser = user
	return f.accessToken, f.expiresIn, f.accessErr
}
func (f *fakeTokenService) HashRefreshToken(plain string) (string, error) {
	if plain == "" {
		return "", entity.ErrInvalidRefreshToken
	}
	return f.refreshHash, nil
}
func (f *fakeTokenService) GenerateRefreshToken() (string, string, error) {
	return f.refreshPlain, f.refreshHash, f.refreshErr
}
func (f *fakeTokenService) GenerateCSRFToken() (string, error) { return f.csrf, f.csrfErr }
func (f *fakeTokenService) GenerateFamilyID() uuid.UUID        { return f.familyID }
func (f *fakeTokenService) RefreshTokenTTL() time.Duration     { return f.refreshTTL }

func newLoginUsecase(repo *fakeUserRepository, refresh *fakeRefreshRepository, tokens *fakeTokenService) IUserUsecase {
	return NewUserUsecase(repo, &fakeHasher{}, &fakeClock{now: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)}, refresh, tokens)
}
func activeLoginUser() *entity.User {
	return &entity.User{ID: uuid.New(), Name: "Taro", Email: "taro@example.com", PasswordHash: "hash", Role: entity.UserRoleUser, Status: entity.UserStatusActive, TokenVersion: 9}
}
func validFakeTokens() *fakeTokenService {
	return &fakeTokenService{accessToken: "access", expiresIn: 300, refreshPlain: "plain", refreshHash: "hash2", csrf: "csrf", familyID: uuid.New(), refreshTTL: 24 * time.Hour}
}

func TestLoginSuccessAndTokenVersion(t *testing.T) {
	user := activeLoginUser()
	repo := &fakeUserRepository{findUser: user}
	refresh := &fakeRefreshRepository{}
	tokens := validFakeTokens()
	result, err := newLoginUsecase(repo, refresh, tokens).Login(context.Background(), LoginInput{Email: user.Email, Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "access" || result.RefreshToken != "plain" || result.CSRFToken != "csrf" {
		t.Fatalf("result=%#v", result)
	}
	if tokens.accessUser.TokenVersion != 9 {
		t.Fatalf("token version=%d", tokens.accessUser.TokenVersion)
	}
	if refresh.calls != 1 || refresh.input.UserID != user.ID || refresh.input.TokenHash != "hash2" || refresh.input.FamilyID != tokens.familyID {
		t.Fatalf("refresh=%#v", refresh.input)
	}
}

func TestLoginFailures(t *testing.T) {
	tests := []struct {
		name      string
		repo      *fakeUserRepository
		configure func(*fakeTokenService, *fakeRefreshRepository)
		want      error
	}{
		{"missing user", &fakeUserRepository{findErr: entity.ErrUserNotFound}, nil, entity.ErrInvalidCredentials},
		{"repository", &fakeUserRepository{findErr: errors.New("db")}, nil, entity.ErrInternal},
		{"password", &fakeUserRepository{findUser: activeLoginUser()}, func(t *fakeTokenService, _ *fakeRefreshRepository) { t.compareErr = entity.ErrInvalidCredentials }, entity.ErrInvalidCredentials},
		{"suspended", &fakeUserRepository{findUser: func() *entity.User { u := activeLoginUser(); u.Status = entity.UserStatusSuspended; return u }()}, nil, entity.ErrUserSuspended},
		{"access", &fakeUserRepository{findUser: activeLoginUser()}, func(t *fakeTokenService, _ *fakeRefreshRepository) { t.accessErr = errors.New("sign") }, entity.ErrInternal},
		{"refresh generation", &fakeUserRepository{findUser: activeLoginUser()}, func(t *fakeTokenService, _ *fakeRefreshRepository) { t.refreshErr = errors.New("random") }, entity.ErrInternal},
		{"csrf generation", &fakeUserRepository{findUser: activeLoginUser()}, func(t *fakeTokenService, _ *fakeRefreshRepository) { t.csrfErr = errors.New("random") }, entity.ErrInternal},
		{"refresh save", &fakeUserRepository{findUser: activeLoginUser()}, func(_ *fakeTokenService, r *fakeRefreshRepository) { r.err = errors.New("db") }, entity.ErrInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := validFakeTokens()
			refresh := &fakeRefreshRepository{}
			if tt.configure != nil {
				tt.configure(tokens, refresh)
			}
			_, err := newLoginUsecase(tt.repo, refresh, tokens).Login(context.Background(), LoginInput{Email: "taro@example.com", Password: "password"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestLoginMissingUserUsesDummyHash(t *testing.T) {
	tokens := validFakeTokens()
	_, _ = newLoginUsecase(&fakeUserRepository{findErr: entity.ErrUserNotFound}, &fakeRefreshRepository{}, tokens).Login(context.Background(), LoginInput{Email: "missing@example.com", Password: "password"})
	if tokens.compareCalls != 1 || tokens.compareHash != dummyPasswordHash || tokens.comparePassword != "password" {
		t.Fatalf("dummy compare not performed: %#v", tokens)
	}
}
