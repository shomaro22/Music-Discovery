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
	questionRepository       repository.IQuestionRepository
	questionAnswerRepository repository.IQuestionAnswerRepository
	userProfileRepository    repository.IUserProfileRepository
	transactionManager       repository.ITransactionManager
	userRepository           repository.IUserRepository
	clock                    IClock
}

func NewQuestionUsecase(
	questionRepository repository.IQuestionRepository,
	questionAnswerRepository repository.IQuestionAnswerRepository,
	userProfileRepository repository.IUserProfileRepository,
	transactionManager repository.ITransactionManager,
	userRepository repository.IUserRepository,
	clock IClock,
) IQuestionUsecase {
	return &questionUsecase{
		questionRepository:       questionRepository,
		questionAnswerRepository: questionAnswerRepository,
		userProfileRepository:    userProfileRepository,
		transactionManager:       transactionManager,
		userRepository:           userRepository,
		clock:                    clock,
	}
}

func (u *questionUsecase) ListQuestions(ctx context.Context) ([]entity.Question, error) {
	questions, err := u.questionRepository.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateQuestionConfiguration(questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func validateQuestionConfiguration(questions []entity.Question) error {
	type expectedQuestion struct {
		code          string
		answerType    entity.QuestionAnswerType
		minSelections int
		maxSelections *int
		sortOrder     int
	}
	one := 1
	three := 3
	expected := []expectedQuestion{
		{entity.QuestionCodeMood, entity.QuestionAnswerTypeSingle, 1, &one, 1},
		{entity.QuestionCodeElement, entity.QuestionAnswerTypeMultiple, 1, nil, 2},
		{entity.QuestionCodeAtmosphere, entity.QuestionAnswerTypeMultiple, 1, nil, 3},
		{entity.QuestionCodeGenre, entity.QuestionAnswerTypeMultiple, 1, &three, 4},
		{entity.QuestionCodeDiscoveryLevel, entity.QuestionAnswerTypeSingle, 1, &one, 5},
	}
	if len(questions) != len(expected) {
		return entity.ErrQuestionConfiguration
	}
	seen := make(map[string]struct{}, len(expected))
	for index, question := range questions {
		definition := expected[index]
		if question.Code != definition.code || question.AnswerType != definition.answerType || question.MinSelections != definition.minSelections || question.SortOrder != definition.sortOrder || !sameOptionalInt(question.MaxSelections, definition.maxSelections) || len(question.Choices) == 0 {
			return entity.ErrQuestionConfiguration
		}
		if _, exists := seen[question.Code]; exists {
			return entity.ErrQuestionConfiguration
		}
		seen[question.Code] = struct{}{}
	}
	return nil
}

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
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
	validated, choicesByID, err := u.validateAnswers(ctx, questions, input.Answers)
	if err != nil {
		return nil, err
	}
	now := u.clock.Now()
	session := buildAnswerSession(userID, questions, validated, now)
	profile, genre, element, atmosphere := buildProfile(userID, questions, choicesByID, now)
	initialized := false
	err = u.transactionManager.WithinTransaction(ctx, func(tx repository.ITransaction) error {
		if err := u.questionAnswerRepository.CreateSession(ctx, tx, session); err != nil {
			return err
		}
		existing, findErr := u.userProfileRepository.FindByUserID(ctx, tx, userID)
		switch {
		case errors.Is(findErr, entity.ErrNotFound):
			initialized = true
			if err := u.userProfileRepository.Create(ctx, tx, profile); err != nil {
				return err
			}
		case findErr != nil:
			return findErr
		default:
			profile.ID = existing.ID
			profile.CreatedAt = existing.CreatedAt
			if err := u.userProfileRepository.UpdateDiscoveryLevel(ctx, tx, existing.ID, profile.DiscoveryLevel, now); err != nil {
				return err
			}
			// Reset every existing question-derived weight, including rows whose
			// choices have since been disabled. Feedback learning is preserved.
			if err := u.userProfileRepository.ResetQuestionWeights(ctx, tx, existing.ID, now); err != nil {
				return err
			}
		}
		assignProfileID(profile.ID, genre, element, atmosphere)
		if err := u.userProfileRepository.UpsertGenreWeights(ctx, tx, genre); err != nil {
			return err
		}
		if err := u.userProfileRepository.UpsertElementWeights(ctx, tx, element); err != nil {
			return err
		}
		if err := u.userProfileRepository.UpsertAtmosphereWeights(ctx, tx, atmosphere); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &SaveQuestionAnswersResult{AnswerSessionID: session.ID, ProfileInitialized: initialized, AnsweredAt: now}, nil
}

func buildAnswerSession(userID uuid.UUID, questions []entity.Question, validated map[uuid.UUID][]uuid.UUID, now time.Time) *entity.QuestionAnswerSession {
	session := &entity.QuestionAnswerSession{ID: uuid.New(), UserID: userID, AnsweredAt: now, CreatedAt: now}
	for _, question := range questions {
		answer := entity.UserAnswer{ID: uuid.New(), AnswerSessionID: session.ID, QuestionID: question.ID, CreatedAt: now}
		for _, choiceID := range validated[question.ID] {
			answer.Choices = append(answer.Choices, entity.UserAnswerChoice{ID: uuid.New(), UserAnswerID: answer.ID, QuestionChoiceID: choiceID, CreatedAt: now})
		}
		session.Answers = append(session.Answers, answer)
	}
	return session
}

func assignProfileID(profileID uuid.UUID, genre []entity.UserGenreWeight, element []entity.UserElementWeight, atmosphere []entity.UserAtmosphereWeight) {
	for index := range genre {
		genre[index].ProfileID = profileID
	}
	for index := range element {
		element[index].ProfileID = profileID
	}
	for index := range atmosphere {
		atmosphere[index].ProfileID = profileID
	}
}

func (u *questionUsecase) validateAnswers(ctx context.Context, questions []entity.Question, answers []QuestionAnswerInput) (map[uuid.UUID][]uuid.UUID, map[uuid.UUID]entity.QuestionChoice, error) {
	if len(answers) != len(questions) {
		return nil, nil, entity.ErrRequiredQuestionMissing
	}
	questionMap := make(map[uuid.UUID]entity.Question, len(questions))
	for _, question := range questions {
		questionMap[question.ID] = question
	}
	validated := make(map[uuid.UUID][]uuid.UUID, len(answers))
	allIDs := make([]uuid.UUID, 0)
	for _, answer := range answers {
		question, exists := questionMap[answer.QuestionID]
		if !exists {
			return nil, nil, entity.ErrQuestionNotFound
		}
		if _, duplicated := validated[answer.QuestionID]; duplicated {
			return nil, nil, entity.ErrDuplicateQuestion
		}
		if len(answer.ChoiceIDs) < question.MinSelections || (question.MaxSelections != nil && len(answer.ChoiceIDs) > *question.MaxSelections) {
			return nil, nil, entity.ErrInvalidSelectionCount
		}
		seen := make(map[uuid.UUID]struct{}, len(answer.ChoiceIDs))
		for _, choiceID := range answer.ChoiceIDs {
			if _, duplicated := seen[choiceID]; duplicated {
				return nil, nil, entity.ErrDuplicateChoice
			}
			seen[choiceID] = struct{}{}
			allIDs = append(allIDs, choiceID)
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
	for questionID, choiceIDs := range validated {
		for _, choiceID := range choiceIDs {
			if choiceMap[choiceID].QuestionID != questionID {
				return nil, nil, entity.ErrQuestionChoiceMismatch
			}
		}
	}
	return validated, choiceMap, nil
}

func buildProfile(userID uuid.UUID, questions []entity.Question, selected map[uuid.UUID]entity.QuestionChoice, now time.Time) (*entity.UserProfile, []entity.UserGenreWeight, []entity.UserElementWeight, []entity.UserAtmosphereWeight) {
	profile := &entity.UserProfile{ID: uuid.New(), UserID: userID, DiscoveryLevel: "DISCOVERY_MEDIUM", CreatedAt: now, UpdatedAt: now}
	genre := make([]entity.UserGenreWeight, 0)
	element := make([]entity.UserElementWeight, 0)
	atmosphere := make([]entity.UserAtmosphereWeight, 0)
	for _, question := range questions {
		for _, choice := range question.Choices {
			questionWeight := 0.0
			if _, exists := selected[choice.ID]; exists {
				questionWeight = 1.0
			}
			switch question.Code {
			case entity.QuestionCodeGenre:
				genre = append(genre, entity.UserGenreWeight{ID: uuid.New(), ChoiceID: choice.ID, QuestionWeight: questionWeight, Weight: questionWeight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeElement:
				element = append(element, entity.UserElementWeight{ID: uuid.New(), ChoiceID: choice.ID, QuestionWeight: questionWeight, Weight: questionWeight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeAtmosphere:
				atmosphere = append(atmosphere, entity.UserAtmosphereWeight{ID: uuid.New(), ChoiceID: choice.ID, QuestionWeight: questionWeight, Weight: questionWeight, CreatedAt: now, UpdatedAt: now})
			case entity.QuestionCodeDiscoveryLevel:
				if questionWeight == 1 {
					profile.DiscoveryLevel = choice.Code
				}
			}
		}
	}
	return profile, genre, element, atmosphere
}

func (u *questionUsecase) GetLatestAnswers(ctx context.Context, userID uuid.UUID) (*LatestQuestionAnswersResult, error) {
	if _, err := u.userRepository.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	session, err := u.questionAnswerRepository.GetLatest(ctx, userID)
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
