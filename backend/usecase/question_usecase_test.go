package usecase

import (
	"context"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type questionRepositoryStub struct {
	questions []entity.Question
	choices   []entity.QuestionChoice
	saved     *entity.QuestionAnswerSession
}

func (s *questionRepositoryStub) ListActive(context.Context) ([]entity.Question, error) {
	return s.questions, nil
}
func (s *questionRepositoryStub) FindChoicesByIDs(context.Context, []uuid.UUID) ([]entity.QuestionChoice, error) {
	return s.choices, nil
}
func (s *questionRepositoryStub) GetLatestAnswers(context.Context, uuid.UUID) (*entity.QuestionAnswerSession, error) {
	return nil, entity.ErrNotFound
}
func (s *questionRepositoryStub) SaveAnswersAndProfile(_ context.Context, session *entity.QuestionAnswerSession, _ *entity.UserProfile, _ []entity.UserGenreWeight, _ []entity.UserElementWeight, _ []entity.UserAtmosphereWeight) (bool, error) {
	s.saved = session
	return true, nil
}

type questionUserRepositoryStub struct{ user *entity.User }

func (s questionUserRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return s.user, nil
}
func (questionUserRepositoryStub) FindByEmail(context.Context, string) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}
func (questionUserRepositoryStub) Create(context.Context, *entity.User) (*entity.User, error) {
	return nil, entity.ErrInternal
}
func (questionUserRepositoryStub) SoftDelete(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return nil
}
func (questionUserRepositoryStub) IncrementTokenVersion(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return nil
}

type questionClock struct{ now time.Time }

func (c questionClock) Now() time.Time { return c.now }

func buildQuestionFixture() ([]entity.Question, []entity.QuestionChoice, []QuestionAnswerInput) {
	codes := []string{entity.QuestionCodeMood, entity.QuestionCodeElement, entity.QuestionCodeAtmosphere, entity.QuestionCodeGenre, entity.QuestionCodeDiscoveryLevel}
	questions := make([]entity.Question, 0, 5)
	choices := make([]entity.QuestionChoice, 0, 5)
	answers := make([]QuestionAnswerInput, 0, 5)
	for index, code := range codes {
		qID := uuid.New()
		cID := uuid.New()
		max := 1
		if index == 1 || index == 2 {
			max = 5
		}
		if index == 3 {
			max = 3
		}
		choice := entity.QuestionChoice{ID: cID, QuestionID: qID, Code: code + "_CHOICE", IsActive: true}
		question := entity.Question{ID: qID, Code: code, MinSelections: 1, MaxSelections: &max, SortOrder: index + 1, Choices: []entity.QuestionChoice{choice}}
		questions = append(questions, question)
		choices = append(choices, choice)
		answers = append(answers, QuestionAnswerInput{QuestionID: qID, ChoiceIDs: []uuid.UUID{cID}})
	}
	return questions, choices, answers
}

func TestQuestionUsecaseSaveAnswersCreatesSingleSession(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	repo := &questionRepositoryStub{questions: questions, choices: choices}
	userID := uuid.New()
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	u := NewQuestionUsecase(repo, questionUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}}, questionClock{now: now})
	result, err := u.SaveAnswers(context.Background(), userID, SaveQuestionAnswersInput{Answers: answers})
	if err != nil {
		t.Fatalf("SaveAnswers error: %v", err)
	}
	if repo.saved == nil || len(repo.saved.Answers) != 5 {
		t.Fatalf("saved session=%+v", repo.saved)
	}
	if result.AnsweredAt != now || !result.ProfileInitialized {
		t.Fatalf("result=%+v", result)
	}
}

func TestQuestionUsecaseRejectsDuplicateQuestion(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	answers[1].QuestionID = answers[0].QuestionID
	repo := &questionRepositoryStub{questions: questions, choices: choices}
	userID := uuid.New()
	u := NewQuestionUsecase(repo, questionUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}}, questionClock{now: time.Now()})
	_, err := u.SaveAnswers(context.Background(), userID, SaveQuestionAnswersInput{Answers: answers})
	if err != entity.ErrDuplicateQuestion {
		t.Fatalf("error=%v", err)
	}
}
