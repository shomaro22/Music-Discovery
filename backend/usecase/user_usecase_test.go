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
