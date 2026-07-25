package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type QuestionAnswerInput struct {
	QuestionID uuid.UUID
	ChoiceIDs  []uuid.UUID
}
type SaveQuestionAnswersInput struct{ Answers []QuestionAnswerInput }
type SaveQuestionAnswersResult struct {
	AnswerSessionID    uuid.UUID
	ProfileInitialized bool
	AnsweredAt         time.Time
}
type LatestQuestionAnswer struct {
	QuestionID uuid.UUID
	ChoiceIDs  []uuid.UUID
}
type LatestQuestionAnswersResult struct {
	Answers    []LatestQuestionAnswer
	AnsweredAt *time.Time
}

type IQuestionUsecase interface {
	ListQuestions(ctx context.Context) ([]entity.Question, error)
	SaveAnswers(ctx context.Context, userID uuid.UUID, input SaveQuestionAnswersInput) (*SaveQuestionAnswersResult, error)
	GetLatestAnswers(ctx context.Context, userID uuid.UUID) (*LatestQuestionAnswersResult, error)
}

type questionUsecase struct {
	questionRepository repository.IQuestionRepository
	userRepository     repository.IUserRepository
	clock              IClock
}

func NewQuestionUsecase(questionRepository repository.IQuestionRepository, userRepository repository.IUserRepository, clock IClock) IQuestionUsecase {
	return &questionUsecase{questionRepository: questionRepository, userRepository: userRepository, clock: clock}
}

func (u *questionUsecase) ListQuestions(ctx context.Context) ([]entity.Question, error) {
	questions, err := u.questionRepository.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	if len(questions) != 5 {
		return nil, entity.ErrQuestionConfiguration
	}
	return questions, nil
}

func (u *questionUsecase) SaveAnswers(ctx context.Context, userID uuid.UUID, input SaveQuestionAnswersInput) (*SaveQuestionAnswersResult, error) {
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	questions, err := u.ListQuestions(ctx)
	if err != nil {
		return nil, err
	}
	validated, choicesByQuestion, err := u.validateAnswers(ctx, questions, input.Answers)
	if err != nil {
		return nil, err
	}
	now := u.clock.Now()
	session := &entity.QuestionAnswerSession{ID: uuid.New(), UserID: userID, AnsweredAt: now, CreatedAt: now}
	for _, question := range questions {
		answer := entity.UserAnswer{ID: uuid.New(), AnswerSessionID: session.ID, QuestionID: question.ID, CreatedAt: now}
		for _, choiceID := range validated[question.ID] {
			answer.Choices = append(answer.Choices, entity.UserAnswerChoice{ID: uuid.New(), UserAnswerID: answer.ID, QuestionChoiceID: choiceID, CreatedAt: now})
		}
		session.Answers = append(session.Answers, answer)
	}
	profile, genre, element, atmosphere := buildProfile(userID, questions, choicesByQuestion, now)
	initialized, err := u.questionRepository.SaveAnswersAndProfile(ctx, session, profile, genre, element, atmosphere)
	if err != nil {
		return nil, err
	}
	return &SaveQuestionAnswersResult{AnswerSessionID: session.ID, ProfileInitialized: initialized, AnsweredAt: now}, nil
}

func (u *questionUsecase) validateAnswers(ctx context.Context, questions []entity.Question, answers []QuestionAnswerInput) (map[uuid.UUID][]uuid.UUID, map[uuid.UUID]entity.QuestionChoice, error) {
	if len(answers) != len(questions) {
		return nil, nil, entity.ErrRequiredQuestionMissing
	}
	questionMap := make(map[uuid.UUID]entity.Question, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}
	validated := make(map[uuid.UUID][]uuid.UUID, len(answers))
	allIDs := make([]uuid.UUID, 0)
	for _, answer := range answers {
		question, ok := questionMap[answer.QuestionID]
		if !ok {
			return nil, nil, entity.ErrQuestionNotFound
		}
		if _, exists := validated[answer.QuestionID]; exists {
			return nil, nil, entity.ErrDuplicateQuestion
		}
		if len(answer.ChoiceIDs) < question.MinSelections || (question.MaxSelections != nil && len(answer.ChoiceIDs) > *question.MaxSelections) {
			return nil, nil, entity.ErrInvalidSelectionCount
		}
		seen := map[uuid.UUID]struct{}{}
		for _, id := range answer.ChoiceIDs {
			if _, ok := seen[id]; ok {
				return nil, nil, entity.ErrDuplicateChoice
			}
			seen[id] = struct{}{}
			allIDs = append(allIDs, id)
		}
		validated[answer.QuestionID] = append([]uuid.UUID(nil), answer.ChoiceIDs...)
	}
	choices, err := u.questionRepository.FindChoicesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(choices) != len(allIDs) {
		return nil, nil, entity.ErrChoiceNotFound
	}
	choiceMap := make(map[uuid.UUID]entity.QuestionChoice, len(choices))
	for _, choice := range choices {
		if !choice.IsActive {
			return nil, nil, entity.ErrChoiceNotFound
		}
		choiceMap[choice.ID] = choice
	}
	for questionID, ids := range validated {
		for _, id := range ids {
			if choiceMap[id].QuestionID != questionID {
				return nil, nil, entity.ErrQuestionChoiceMismatch
			}
		}
	}
	return validated, choiceMap, nil
}

func buildProfile(userID uuid.UUID, questions []entity.Question, selected map[uuid.UUID]entity.QuestionChoice, now time.Time) (*entity.UserProfile, []entity.UserGenreWeight, []entity.UserElementWeight, []entity.UserAtmosphereWeight) {
	profile := &entity.UserProfile{ID: uuid.New(), UserID: userID, DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
	genre := []entity.UserGenreWeight{}
	element := []entity.UserElementWeight{}
	atmosphere := []entity.UserAtmosphereWeight{}
	for _, question := range questions {
		for _, choice := range question.Choices {
			weight := 0.0
			if _, ok := selected[choice.ID]; ok {
				weight = 1.0
			}
			switch question.Code {
			case entity.QuestionCodeGenre:
				genre = append(genre, entity.UserGenreWeight{ID: uuid.New(), ChoiceID: choice.ID, Weight: weight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeElement:
				element = append(element, entity.UserElementWeight{ID: uuid.New(), ChoiceID: choice.ID, Weight: weight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeAtmosphere:
				atmosphere = append(atmosphere, entity.UserAtmosphereWeight{ID: uuid.New(), ChoiceID: choice.ID, Weight: weight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeDiscoveryLevel:
				if weight == 1 {
					profile.DiscoveryLevel = choice.Code
				}
			}
		}
	}
	return profile, genre, element, atmosphere
}

func (u *questionUsecase) GetLatestAnswers(ctx context.Context, userID uuid.UUID) (*LatestQuestionAnswersResult, error) {
	_, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	session, err := u.questionRepository.GetLatestAnswers(ctx, userID)
	if errors.Is(err, entity.ErrNotFound) {
		return &LatestQuestionAnswersResult{Answers: []LatestQuestionAnswer{}}, nil
	}
	if err != nil {
		return nil, err
	}
	result := &LatestQuestionAnswersResult{AnsweredAt: &session.AnsweredAt}
	for _, answer := range session.Answers {
		item := LatestQuestionAnswer{QuestionID: answer.QuestionID}
		for _, choice := range answer.Choices {
			item.ChoiceIDs = append(item.ChoiceIDs, choice.QuestionChoiceID)
		}
		result.Answers = append(result.Answers, item)
	}
	return result, nil
}
