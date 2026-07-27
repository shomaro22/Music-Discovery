package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type questionRepositoryStub struct {
	questions []entity.Question
	choices   []entity.QuestionChoice
	listErr   error
}

func (s *questionRepositoryStub) ListActive(context.Context) ([]entity.Question, error) {
	return s.questions, s.listErr
}
func (s *questionRepositoryStub) FindChoicesByIDs(context.Context, []uuid.UUID) ([]entity.QuestionChoice, error) {
	return s.choices, nil
}

type questionAnswerRepositoryStub struct {
	created   []*entity.QuestionAnswerSession
	latest    *entity.QuestionAnswerSession
	latestErr error
	createErr error
}

func (s *questionAnswerRepositoryStub) CreateSession(_ context.Context, _ repository.ITransaction, session *entity.QuestionAnswerSession) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, session)
	return nil
}
func (s *questionAnswerRepositoryStub) GetLatest(context.Context, uuid.UUID) (*entity.QuestionAnswerSession, error) {
	return s.latest, s.latestErr
}

type userProfileRepositoryStub struct {
	profile    *entity.UserProfile
	findErr    error
	created    int
	updated    int
	reset      int
	genre      []entity.UserGenreWeight
	element    []entity.UserElementWeight
	atmosphere []entity.UserAtmosphereWeight
	weightErr  error
}

func (s *userProfileRepositoryStub) FindByUserID(context.Context, repository.ITransaction, uuid.UUID) (*entity.UserProfile, error) {
	return s.profile, s.findErr
}
func (s *userProfileRepositoryStub) Create(_ context.Context, _ repository.ITransaction, profile *entity.UserProfile) error {
	s.created++
	s.profile = profile
	return nil
}
func (s *userProfileRepositoryStub) UpdateDiscoveryLevel(context.Context, repository.ITransaction, uuid.UUID, string, time.Time) error {
	s.updated++
	return nil
}
func (s *userProfileRepositoryStub) ResetQuestionWeights(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	s.reset++
	return nil
}
func (s *userProfileRepositoryStub) UpsertGenreWeights(_ context.Context, _ repository.ITransaction, values []entity.UserGenreWeight) error {
	s.genre = values
	return s.weightErr
}
func (s *userProfileRepositoryStub) UpsertElementWeights(_ context.Context, _ repository.ITransaction, values []entity.UserElementWeight) error {
	s.element = values
	return s.weightErr
}
func (s *userProfileRepositoryStub) UpsertAtmosphereWeights(_ context.Context, _ repository.ITransaction, values []entity.UserAtmosphereWeight) error {
	s.atmosphere = values
	return s.weightErr
}

type transactionManagerStub struct {
	rolledBack bool
}

func (s *transactionManagerStub) WithinTransaction(_ context.Context, fn func(repository.ITransaction) error) error {
	err := fn(nil)
	s.rolledBack = err != nil
	return err
}

type questionUserRepositoryStub struct {
	user *entity.User
	err  error
}

func (s questionUserRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return s.user, s.err
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
	definitions := []struct {
		code       string
		answerType entity.QuestionAnswerType
		max        *int
	}{
		{entity.QuestionCodeMood, entity.QuestionAnswerTypeSingle, intTestPointer(1)},
		{entity.QuestionCodeElement, entity.QuestionAnswerTypeMultiple, nil},
		{entity.QuestionCodeAtmosphere, entity.QuestionAnswerTypeMultiple, nil},
		{entity.QuestionCodeGenre, entity.QuestionAnswerTypeMultiple, intTestPointer(3)},
		{entity.QuestionCodeDiscoveryLevel, entity.QuestionAnswerTypeSingle, intTestPointer(1)},
	}
	questions := make([]entity.Question, 0, len(definitions))
	choices := make([]entity.QuestionChoice, 0, len(definitions))
	answers := make([]QuestionAnswerInput, 0, len(definitions))
	for index, definition := range definitions {
		questionID := uuid.New()
		choiceID := uuid.New()
		choice := entity.QuestionChoice{ID: choiceID, QuestionID: questionID, Code: definition.code + "_CHOICE", IsActive: true, SortOrder: 1}
		question := entity.Question{ID: questionID, Code: definition.code, AnswerType: definition.answerType, MinSelections: 1, MaxSelections: definition.max, SortOrder: index + 1, Choices: []entity.QuestionChoice{choice}}
		questions = append(questions, question)
		choices = append(choices, choice)
		answers = append(answers, QuestionAnswerInput{QuestionID: questionID, ChoiceIDs: []uuid.UUID{choiceID}})
	}
	return questions, choices, answers
}

func intTestPointer(value int) *int { return &value }

func newQuestionUsecaseForTest(questionRepo *questionRepositoryStub, answerRepo *questionAnswerRepositoryStub, profileRepo *userProfileRepositoryStub, tx *transactionManagerStub, userID uuid.UUID) IQuestionUsecase {
	return NewQuestionUsecase(questionRepo, answerRepo, profileRepo, tx, questionUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}}, questionClock{now: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)})
}

func TestQuestionUsecaseListQuestionsValidatesExactConfiguration(t *testing.T) {
	questions, _, _ := buildQuestionFixture()
	u := newQuestionUsecaseForTest(&questionRepositoryStub{questions: questions}, &questionAnswerRepositoryStub{}, &userProfileRepositoryStub{}, &transactionManagerStub{}, uuid.New())
	if _, err := u.ListQuestions(context.Background()); err != nil {
		t.Fatalf("ListQuestions error: %v", err)
	}
	questions[0].Code = "OTHER"
	if _, err := u.ListQuestions(context.Background()); !errors.Is(err, entity.ErrQuestionConfiguration) {
		t.Fatalf("expected configuration error, got %v", err)
	}
}

func TestQuestionUsecaseSaveAnswersCreatesSingleSessionAndProfile(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	answerRepo := &questionAnswerRepositoryStub{}
	profileRepo := &userProfileRepositoryStub{findErr: entity.ErrNotFound}
	userID := uuid.New()
	u := newQuestionUsecaseForTest(&questionRepositoryStub{questions: questions, choices: choices}, answerRepo, profileRepo, &transactionManagerStub{}, userID)
	result, err := u.SaveAnswers(context.Background(), userID, SaveQuestionAnswersInput{Answers: answers})
	if err != nil {
		t.Fatalf("SaveAnswers error: %v", err)
	}
	if len(answerRepo.created) != 1 || len(answerRepo.created[0].Answers) != 5 {
		t.Fatalf("session was not saved as one unit")
	}
	if !result.ProfileInitialized || profileRepo.created != 1 {
		t.Fatalf("profile was not initialized")
	}
}

func TestQuestionUsecaseReanswerCreatesNewSessionWithoutDuplicateProfile(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	answerRepo := &questionAnswerRepositoryStub{}
	profileRepo := &userProfileRepositoryStub{profile: &entity.UserProfile{ID: uuid.New(), UserID: uuid.New()}}
	userID := uuid.New()
	u := newQuestionUsecaseForTest(&questionRepositoryStub{questions: questions, choices: choices}, answerRepo, profileRepo, &transactionManagerStub{}, userID)
	for range 2 {
		if _, err := u.SaveAnswers(context.Background(), userID, SaveQuestionAnswersInput{Answers: answers}); err != nil {
			t.Fatalf("SaveAnswers error: %v", err)
		}
	}
	if len(answerRepo.created) != 2 || answerRepo.created[0].ID == answerRepo.created[1].ID {
		t.Fatalf("reanswer must create a new session")
	}
	if profileRepo.created != 0 || profileRepo.updated != 2 {
		t.Fatalf("profile must be updated, not duplicated")
	}
	if profileRepo.reset != 2 {
		t.Fatalf("question-derived weights must be reset before each reanswer: reset=%d", profileRepo.reset)
	}
}

func TestQuestionUsecaseRejectsInvalidAnswers(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	tests := []struct {
		name   string
		mutate func([]QuestionAnswerInput)
		want   error
	}{
		{"missing", func(values []QuestionAnswerInput) { values[4] = QuestionAnswerInput{} }, entity.ErrQuestionNotFound},
		{"duplicate question", func(values []QuestionAnswerInput) { values[1].QuestionID = values[0].QuestionID }, entity.ErrDuplicateQuestion},
		{"duplicate choice", func(values []QuestionAnswerInput) {
			values[1].ChoiceIDs = []uuid.UUID{values[1].ChoiceIDs[0], values[1].ChoiceIDs[0]}
		}, entity.ErrDuplicateChoice},
		{"empty", func(values []QuestionAnswerInput) { values[1].ChoiceIDs = nil }, entity.ErrInvalidSelectionCount},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copied := append([]QuestionAnswerInput(nil), answers...)
			test.mutate(copied)
			u := newQuestionUsecaseForTest(&questionRepositoryStub{questions: questions, choices: choices}, &questionAnswerRepositoryStub{}, &userProfileRepositoryStub{findErr: entity.ErrNotFound}, &transactionManagerStub{}, uuid.New())
			_, err := u.SaveAnswers(context.Background(), uuid.New(), SaveQuestionAnswersInput{Answers: copied})
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestQuestionUsecaseRollsBackWhenProfileSaveFails(t *testing.T) {
	questions, choices, answers := buildQuestionFixture()
	tx := &transactionManagerStub{}
	u := newQuestionUsecaseForTest(&questionRepositoryStub{questions: questions, choices: choices}, &questionAnswerRepositoryStub{}, &userProfileRepositoryStub{findErr: entity.ErrNotFound, weightErr: entity.ErrProfileCreateFailed}, tx, uuid.New())
	_, err := u.SaveAnswers(context.Background(), uuid.New(), SaveQuestionAnswersInput{Answers: answers})
	if !errors.Is(err, entity.ErrProfileCreateFailed) || !tx.rolledBack {
		t.Fatalf("transaction was not rolled back: err=%v", err)
	}
}

func TestQuestionUsecaseGetLatestAnswers(t *testing.T) {
	userID := uuid.New()
	questionID := uuid.New()
	choiceID := uuid.New()
	now := time.Now().UTC()
	answerRepo := &questionAnswerRepositoryStub{latest: &entity.QuestionAnswerSession{AnsweredAt: now, Answers: []entity.UserAnswer{{QuestionID: questionID, Choices: []entity.UserAnswerChoice{{QuestionChoiceID: choiceID}}}}}}
	u := newQuestionUsecaseForTest(&questionRepositoryStub{}, answerRepo, &userProfileRepositoryStub{}, &transactionManagerStub{}, userID)
	result, err := u.GetLatestAnswers(context.Background(), userID)
	if err != nil || len(result.Answers) != 1 || result.Answers[0].ChoiceIDs[0] != choiceID {
		t.Fatalf("latest result=%+v err=%v", result, err)
	}
}

func TestBuildProfileSeparatesQuestionWeightFromFeedbackWeight(t *testing.T) {
	questions, choices, _ := buildQuestionFixture()
	selected := map[uuid.UUID]entity.QuestionChoice{choices[3].ID: choices[3]}
	_, genre, _, _ := buildProfile(uuid.New(), questions, selected, time.Now().UTC())
	if len(genre) != 1 || genre[0].QuestionWeight != 1 || genre[0].FeedbackWeight != 0 || genre[0].Weight != 1 {
		t.Fatalf("genre weight=%+v", genre)
	}
}
