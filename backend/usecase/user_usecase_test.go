package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
	"music-discovery/backend/validator"

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

type fakeRefreshRepository struct{}

func (f *fakeRefreshRepository) Create(context.Context, *entity.RefreshToken) error { return nil }
func (f *fakeRefreshRepository) FindByTokenHash(context.Context, string) (*entity.RefreshToken, error) {
	return nil, entity.ErrInvalidRefreshToken
}
func (f *fakeRefreshRepository) Rotate(context.Context, uuid.UUID, *entity.RefreshToken, time.Time) error {
	return nil
}
func (f *fakeRefreshRepository) RevokeFamily(context.Context, uuid.UUID, time.Time) error { return nil }

type fakeAuthTransactionManager struct{ repo *fakeUserRepository }

func (f *fakeAuthTransactionManager) CreateUserAndRefreshToken(ctx context.Context, user *entity.User, _ *entity.RefreshToken) error {
	_, err := f.repo.Create(ctx, user)
	return err
}
func (f *fakeAuthTransactionManager) RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

type fakeTokenService struct{}

func (f *fakeTokenService) ComparePassword(string, string) error { return nil }
func (f *fakeTokenService) GenerateAccessToken(*entity.User, time.Time) (string, int64, error) {
	return "", 0, nil
}
func (f *fakeTokenService) GenerateRefreshToken() (string, string, error) { return "", "", nil }
func (f *fakeTokenService) HashRefreshToken(string) string                { return "" }
func (f *fakeTokenService) GenerateCSRFToken() (string, error)            { return "", nil }
func (f *fakeTokenService) GenerateFamilyID() uuid.UUID                   { return uuid.New() }
func (f *fakeTokenService) RefreshTokenTTL() time.Duration                { return time.Hour }
func (f *fakeTokenService) VerifyAccessToken(string, time.Time) (uuid.UUID, uint64, error) {
	return uuid.Nil, 0, nil
}

func newUsecaseForTest(repo *fakeUserRepository, hasher *fakeHasher) IUserUsecase {
	return NewUserUsecase(repo, validator.NewUserValidator(), &fakeRefreshRepository{}, &fakeAuthTransactionManager{repo: repo}, hasher, &fakeTokenService{}, &fakeClock{now: time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)})
}

var _ repository.IUserRepository = (*fakeUserRepository)(nil)

func TestSignUpSuccessSetsInternalFields(t *testing.T) {
	repo := &fakeUserRepository{findErr: entity.ErrUserNotFound}
	hasher := &fakeHasher{hash: "hashed"}
	result, err := newUsecaseForTest(repo, hasher).SignUp(context.Background(), SignUpInput{Name: "Taro", Email: "taro@example.com", Password: " password "})
	if err != nil {
		t.Fatal(err)
	}
	user := result.User
	if user.ID == uuid.Nil || user.Role != entity.UserRoleUser || user.Status != entity.UserStatusActive || user.TokenVersion != 0 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if repo.createInputID == uuid.Nil {
		t.Fatal("usecase must generate user ID before the transaction")
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

func TestValidateTokenVersion(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive, TokenVersion: 4}}
	u := newUsecaseForTest(repo, &fakeHasher{})
	user, err := u.ValidateTokenVersion(context.Background(), userID, 4)
	if err != nil || user == nil {
		t.Fatalf("user=%v err=%v", user, err)
	}
}

func TestValidateTokenVersionRejectsMismatch(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive, TokenVersion: 5}}
	u := newUsecaseForTest(repo, &fakeHasher{})
	if _, err := u.ValidateTokenVersion(context.Background(), userID, 4); !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("err=%v", err)
	}
}

type refreshRepoStub struct {
	current      *entity.RefreshToken
	findErr      error
	rotateErr    error
	findHash     string
	findCalls    int
	rotateCalls  int
	revokeCalls  int
	revokeFamily uuid.UUID
	revokeAt     time.Time
	revokeErr    error
	next         *entity.RefreshToken
}

func (f *refreshRepoStub) Create(context.Context, *entity.RefreshToken) error { return nil }
func (f *refreshRepoStub) FindByTokenHash(_ context.Context, hash string) (*entity.RefreshToken, error) {
	f.findCalls++
	f.findHash = hash
	return f.current, f.findErr
}
func (f *refreshRepoStub) Rotate(_ context.Context, _ uuid.UUID, next *entity.RefreshToken, _ time.Time) error {
	f.rotateCalls++
	f.next = next
	return f.rotateErr
}
func (f *refreshRepoStub) RevokeFamily(_ context.Context, familyID uuid.UUID, now time.Time) error {
	f.revokeCalls++
	f.revokeFamily = familyID
	f.revokeAt = now
	return f.revokeErr
}

type authTxStub struct {
	calls int
	err   error
}

func (f *authTxStub) CreateUserAndRefreshToken(context.Context, *entity.User, *entity.RefreshToken) error {
	return nil
}
func (f *authTxStub) RevokeFamilyAndIncrementTokenVersion(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	f.calls++
	return f.err
}

type tokenServiceStub struct {
	hash         string
	access       string
	expires      int64
	accessErr    error
	refreshPlain string
	refreshHash  string
	refreshErr   error
	csrf         string
	csrfErr      error
	ttl          time.Duration
}

func (f *tokenServiceStub) ComparePassword(string, string) error { return nil }
func (f *tokenServiceStub) GenerateAccessToken(*entity.User, time.Time) (string, int64, error) {
	return f.access, f.expires, f.accessErr
}
func (f *tokenServiceStub) GenerateRefreshToken() (string, string, error) {
	return f.refreshPlain, f.refreshHash, f.refreshErr
}
func (f *tokenServiceStub) HashRefreshToken(string) string     { return f.hash }
func (f *tokenServiceStub) GenerateCSRFToken() (string, error) { return f.csrf, f.csrfErr }
func (f *tokenServiceStub) GenerateFamilyID() uuid.UUID        { return uuid.New() }
func (f *tokenServiceStub) RefreshTokenTTL() time.Duration     { return f.ttl }
func (f *tokenServiceStub) VerifyAccessToken(string, time.Time) (uuid.UUID, uint64, error) {
	return uuid.Nil, 0, nil
}

func newRefreshUsecaseForTest(repo *fakeUserRepository, refreshRepo *refreshRepoStub, tx *authTxStub, token *tokenServiceStub, now time.Time) IUserUsecase {
	return NewUserUsecase(repo, validator.NewUserValidator(), refreshRepo, tx, &fakeHasher{}, token, &fakeClock{now: now})
}

func TestRefreshSuccess(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	userID := uuid.New()
	familyID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: familyID, TokenHash: "old-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive}}
	refreshRepo := &refreshRepoStub{current: current}
	tx := &authTxStub{}
	token := &tokenServiceStub{hash: "old-hash", access: "access-new", expires: 300, refreshPlain: "refresh-new", refreshHash: "refresh-new-hash", csrf: "csrf-new", ttl: 24 * time.Hour}

	result, err := newRefreshUsecaseForTest(repo, refreshRepo, tx, token, now).Refresh(context.Background(), "refresh-old")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "access-new" || result.RefreshToken != "refresh-new" || result.CSRFToken != "csrf-new" || result.ExpiresIn != 300 {
		t.Fatalf("result=%+v", result)
	}
	if refreshRepo.findHash != "old-hash" || refreshRepo.rotateCalls != 1 {
		t.Fatalf("findHash=%q rotateCalls=%d", refreshRepo.findHash, refreshRepo.rotateCalls)
	}
	if refreshRepo.next == nil || refreshRepo.next.FamilyID != familyID || refreshRepo.next.UserID != userID {
		t.Fatalf("next=%+v", refreshRepo.next)
	}
	if tx.calls != 0 {
		t.Fatalf("tx.calls=%d", tx.calls)
	}
}

func TestRefreshExpiredToken(t *testing.T) {
	now := time.Now().UTC()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: uuid.New(), FamilyID: uuid.New(), ExpiresAt: now}
	refreshRepo := &refreshRepoStub{current: current}
	_, err := newRefreshUsecaseForTest(&fakeUserRepository{}, refreshRepo, &authTxStub{}, &tokenServiceStub{hash: "hash", ttl: time.Hour}, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInvalidRefreshToken) || refreshRepo.rotateCalls != 0 {
		t.Fatalf("err=%v rotate=%d", err, refreshRepo.rotateCalls)
	}
}

func TestRefreshRevokedToken(t *testing.T) {
	now := time.Now().UTC()
	revokedAt := now.Add(-time.Minute)
	usedAt := now.Add(-time.Minute)
	current := &entity.RefreshToken{ID: uuid.New(), UserID: uuid.New(), FamilyID: uuid.New(), UsedAt: &usedAt, RevokedAt: &revokedAt, ExpiresAt: now.Add(time.Hour)}
	tx := &authTxStub{}
	_, err := newRefreshUsecaseForTest(&fakeUserRepository{}, &refreshRepoStub{current: current}, tx, &tokenServiceStub{hash: "hash", ttl: time.Hour}, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInvalidRefreshToken) || tx.calls != 0 {
		t.Fatalf("err=%v tx.calls=%d", err, tx.calls)
	}
}

func TestRefreshUsedTokenRevokesFamilyOnce(t *testing.T) {
	now := time.Now().UTC()
	usedAt := now.Add(-time.Minute)
	current := &entity.RefreshToken{ID: uuid.New(), UserID: uuid.New(), FamilyID: uuid.New(), UsedAt: &usedAt, ExpiresAt: now.Add(time.Hour)}
	tx := &authTxStub{}
	_, err := newRefreshUsecaseForTest(&fakeUserRepository{}, &refreshRepoStub{current: current}, tx, &tokenServiceStub{hash: "hash", ttl: time.Hour}, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrRefreshTokenReused) || tx.calls != 1 {
		t.Fatalf("err=%v tx.calls=%d", err, tx.calls)
	}
}

func TestRefreshRotationConflictRevokesFamily(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive}}
	refreshRepo := &refreshRepoStub{current: current, rotateErr: entity.ErrRefreshTokenReused}
	tx := &authTxStub{}
	token := &tokenServiceStub{hash: "hash", refreshPlain: "new", refreshHash: "new-hash", ttl: time.Hour}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, tx, token, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrRefreshTokenReused) || tx.calls != 1 {
		t.Fatalf("err=%v tx.calls=%d", err, tx.calls)
	}
}

func TestRefreshRotationConflictTransactionFailure(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive}}
	refreshRepo := &refreshRepoStub{current: current, rotateErr: entity.ErrRefreshTokenReused}
	tx := &authTxStub{err: errors.New("tx failed")}
	token := &tokenServiceStub{hash: "hash", refreshPlain: "new", refreshHash: "new-hash", ttl: time.Hour}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, tx, token, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
}

func TestRefreshEmptyToken(t *testing.T) {
	now := time.Now().UTC()
	refreshRepo := &refreshRepoStub{}
	_, err := newRefreshUsecaseForTest(&fakeUserRepository{}, refreshRepo, &authTxStub{}, &tokenServiceStub{}, now).Refresh(context.Background(), "")
	if !errors.Is(err, entity.ErrInvalidRefreshToken) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.findCalls != 0 || refreshRepo.rotateCalls != 0 {
		t.Fatalf("findCalls=%d rotateCalls=%d", refreshRepo.findCalls, refreshRepo.rotateCalls)
	}
}

func TestRefreshTokenNotFound(t *testing.T) {
	now := time.Now().UTC()
	refreshRepo := &refreshRepoStub{findErr: entity.ErrInvalidRefreshToken}
	_, err := newRefreshUsecaseForTest(&fakeUserRepository{}, refreshRepo, &authTxStub{}, &tokenServiceStub{hash: "missing-hash"}, now).Refresh(context.Background(), "missing-token")
	if !errors.Is(err, entity.ErrInvalidRefreshToken) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.findCalls != 1 || refreshRepo.rotateCalls != 0 {
		t.Fatalf("findCalls=%d rotateCalls=%d", refreshRepo.findCalls, refreshRepo.rotateCalls)
	}
}

func TestRefreshUserNotFound(t *testing.T) {
	now := time.Now().UTC()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: uuid.New(), FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findErr: entity.ErrUserNotFound}
	refreshRepo := &refreshRepoStub{current: current}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, &authTxStub{}, &tokenServiceStub{hash: "hash"}, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInvalidRefreshToken) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.rotateCalls != 0 {
		t.Fatalf("rotateCalls=%d", refreshRepo.rotateCalls)
	}
}

func TestRefreshSuspendedUser(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusSuspended}}
	refreshRepo := &refreshRepoStub{current: current}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, &authTxStub{}, &tokenServiceStub{hash: "hash"}, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrUserSuspended) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.rotateCalls != 0 {
		t.Fatalf("rotateCalls=%d", refreshRepo.rotateCalls)
	}
}

func TestRefreshRotationFailure(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive}}
	refreshRepo := &refreshRepoStub{current: current, rotateErr: errors.New("database error")}
	tx := &authTxStub{}
	token := &tokenServiceStub{hash: "hash", refreshPlain: "new-token", refreshHash: "new-hash", ttl: time.Hour}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, tx, token, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.rotateCalls != 1 || tx.calls != 0 {
		t.Fatalf("rotateCalls=%d tx.calls=%d", refreshRepo.rotateCalls, tx.calls)
	}
}

func TestRefreshAccessTokenGenerationFailure(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	current := &entity.RefreshToken{ID: uuid.New(), UserID: userID, FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}
	repo := &fakeUserRepository{findUser: &entity.User{ID: userID, Status: entity.UserStatusActive}}
	refreshRepo := &refreshRepoStub{current: current}
	token := &tokenServiceStub{
		hash:         "hash",
		refreshPlain: "new-token",
		refreshHash:  "new-hash",
		ttl:          time.Hour,
		accessErr:    errors.New("signing failed"),
	}
	_, err := newRefreshUsecaseForTest(repo, refreshRepo, &authTxStub{}, token, now).Refresh(context.Background(), "token")
	if !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
	if refreshRepo.rotateCalls != 1 {
		t.Fatalf("rotateCalls=%d", refreshRepo.rotateCalls)
	}
}

func TestLogoutRevokesFamilyForAnyStoredTokenState(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	usedAt := now.Add(-2 * time.Hour)
	revokedAt := now.Add(-time.Hour)

	tests := []struct {
		name  string
		token *entity.RefreshToken
	}{
		{name: "active token", token: &entity.RefreshToken{FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour)}},
		{name: "expired token", token: &entity.RefreshToken{FamilyID: uuid.New(), ExpiresAt: now.Add(-time.Minute)}},
		{name: "used token", token: &entity.RefreshToken{FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour), UsedAt: &usedAt}},
		{name: "revoked token", token: &entity.RefreshToken{FamilyID: uuid.New(), ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refreshRepo := &refreshRepoStub{current: tc.token}
			token := &tokenServiceStub{hash: "hashed-token"}
			u := newRefreshUsecaseForTest(&fakeUserRepository{}, refreshRepo, &authTxStub{}, token, now)
			if err := u.Logout(context.Background(), "plain-token"); err != nil {
				t.Fatal(err)
			}
			if refreshRepo.findHash != "hashed-token" || refreshRepo.revokeCalls != 1 || refreshRepo.revokeFamily != tc.token.FamilyID || !refreshRepo.revokeAt.Equal(now) {
				t.Fatalf("repo=%+v", refreshRepo)
			}
		})
	}
}

func TestLogoutIsIdempotentForMissingToken(t *testing.T) {
	for _, tc := range []struct {
		name    string
		plain   string
		findErr error
	}{
		{name: "empty cookie", plain: ""},
		{name: "unknown token", plain: "plain", findErr: entity.ErrInvalidRefreshToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &refreshRepoStub{findErr: tc.findErr}
			u := newRefreshUsecaseForTest(&fakeUserRepository{}, repo, &authTxStub{}, &tokenServiceStub{hash: "hash"}, time.Now().UTC())
			if err := u.Logout(context.Background(), tc.plain); err != nil {
				t.Fatal(err)
			}
			if repo.revokeCalls != 0 {
				t.Fatalf("revokeCalls=%d", repo.revokeCalls)
			}
		})
	}
}

func TestLogoutRevokeFailureReturnsInternalError(t *testing.T) {
	repo := &refreshRepoStub{current: &entity.RefreshToken{FamilyID: uuid.New()}, revokeErr: errors.New("db down")}
	u := newRefreshUsecaseForTest(&fakeUserRepository{}, repo, &authTxStub{}, &tokenServiceStub{hash: "hash"}, time.Now().UTC())
	if err := u.Logout(context.Background(), "plain"); !errors.Is(err, entity.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
}

func TestCurrentUserReturnsPublicUserInformation(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserRepository{findUser: &entity.User{
		ID: userID, Name: "Taro", Email: "taro@example.com",
		Role: entity.UserRoleUser, Status: entity.UserStatusActive,
		PasswordHash: "must-not-leak", TokenVersion: 9,
	}}

	result, err := newUsecaseForTest(repo, &fakeHasher{}).CurrentUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.ID != userID || result.Email != "taro@example.com" || result.Status != entity.UserStatusActive {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCurrentUserRejectsInvalidOrUnavailableUser(t *testing.T) {
	tests := []struct {
		name    string
		userID  uuid.UUID
		repo    *fakeUserRepository
		wantErr error
	}{
		{name: "nil user id", userID: uuid.Nil, repo: &fakeUserRepository{}, wantErr: entity.ErrInvalidCredentials},
		{name: "not found", userID: uuid.New(), repo: &fakeUserRepository{findErr: entity.ErrUserNotFound}, wantErr: entity.ErrInvalidCredentials},
		{name: "suspended", userID: uuid.New(), repo: &fakeUserRepository{findUser: &entity.User{Status: entity.UserStatusSuspended}}, wantErr: entity.ErrUserSuspended},
		{name: "repository error", userID: uuid.New(), repo: &fakeUserRepository{findErr: errors.New("db error")}, wantErr: entity.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newUsecaseForTest(tt.repo, &fakeHasher{}).CurrentUser(context.Background(), tt.userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want=%v", err, tt.wantErr)
			}
		})
	}
}
