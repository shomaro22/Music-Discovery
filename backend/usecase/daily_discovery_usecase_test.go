package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

type dailyUserRepositoryStub struct {
	user *entity.User
	err  error
}

func (s *dailyUserRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.User, error) {
	return s.user, s.err
}
func (*dailyUserRepositoryStub) FindByEmail(context.Context, string) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}
func (*dailyUserRepositoryStub) Create(context.Context, *entity.User) (*entity.User, error) {
	return nil, entity.ErrInternal
}
func (*dailyUserRepositoryStub) SoftDelete(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return nil
}
func (*dailyUserRepositoryStub) IncrementTokenVersion(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return nil
}
func (*dailyUserRepositoryStub) UpdateProfile(context.Context, uuid.UUID, string, string, time.Time) (*entity.User, error) {
	return nil, entity.ErrInternal
}

type dailyProfileRepositoryStub struct {
	profile *entity.UserProfile
	err     error
}

func (s *dailyProfileRepositoryStub) FindByUserID(context.Context, repository.ITransaction, uuid.UUID) (*entity.UserProfile, error) {
	return s.profile, s.err
}
func (*dailyProfileRepositoryStub) Create(context.Context, repository.ITransaction, *entity.UserProfile) error {
	return nil
}
func (*dailyProfileRepositoryStub) UpdateDiscoveryLevel(context.Context, repository.ITransaction, uuid.UUID, string, time.Time) error {
	return nil
}
func (*dailyProfileRepositoryStub) ResetQuestionWeights(context.Context, repository.ITransaction, uuid.UUID, time.Time) error {
	return nil
}
func (*dailyProfileRepositoryStub) UpsertGenreWeights(context.Context, repository.ITransaction, []entity.UserGenreWeight) error {
	return nil
}
func (*dailyProfileRepositoryStub) UpsertElementWeights(context.Context, repository.ITransaction, []entity.UserElementWeight) error {
	return nil
}
func (*dailyProfileRepositoryStub) UpsertAtmosphereWeights(context.Context, repository.ITransaction, []entity.UserAtmosphereWeight) error {
	return nil
}

type dailyTransactionStub struct{ err error }

func (s *dailyTransactionStub) WithinTransaction(ctx context.Context, fn func(repository.ITransaction) error) error {
	if s.err != nil {
		return s.err
	}
	return fn(nil)
}

type dailyClockStub struct{ now time.Time }

func (c dailyClockStub) Now() time.Time { return c.now }

type dailyConditionBuilderStub struct {
	condition RecommendationCondition
	err       error
}

func (s *dailyConditionBuilderStub) Build(context.Context, uuid.UUID) (RecommendationCondition, error) {
	return s.condition, s.err
}

type dailyRecommendationRepositoryStub struct {
	findDailyValues []*entity.Recommendation
	findDailyErrs   []error
	findDailyDates  []time.Time
	findByIDValue   *entity.Recommendation
	findByIDErr     error
	excluded        []uuid.UUID
	excludedErr     error
	candidates      []entity.RecommendationSongCandidate
	candidatesErr   error
	artistCounts    map[uuid.UUID]int64
	artistCountsErr error
	songCounts      map[uuid.UUID]int64
	songCountsErr   error
	recency         repository.RecommendationRecencyCounts
	recencyErr      error
	createErr       error
	created         *entity.Recommendation
}

func (s *dailyRecommendationRepositoryStub) FindCandidates(context.Context, repository.RecommendationCandidateCondition) ([]entity.RecommendationSongCandidate, error) {
	return s.candidates, s.candidatesErr
}
func (s *dailyRecommendationRepositoryStub) FindRecommendedSongIDs(context.Context, uuid.UUID, time.Time, time.Time) ([]uuid.UUID, error) {
	return s.excluded, s.excludedErr
}
func (s *dailyRecommendationRepositoryStub) Create(_ context.Context, _ repository.ITransaction, value *entity.Recommendation) error {
	s.created = value
	return s.createErr
}
func (*dailyRecommendationRepositoryStub) FindLatest(context.Context, uuid.UUID, entity.RecommendationSource) (*entity.Recommendation, error) {
	return nil, entity.ErrRecommendationNotFound
}
func (s *dailyRecommendationRepositoryStub) FindByID(context.Context, uuid.UUID) (*entity.Recommendation, error) {
	return s.findByIDValue, s.findByIDErr
}
func (s *dailyRecommendationRepositoryStub) FindDailyByDate(_ context.Context, _ uuid.UUID, date time.Time) (*entity.Recommendation, error) {
	s.findDailyDates = append(s.findDailyDates, date)
	index := len(s.findDailyDates) - 1
	if index < len(s.findDailyErrs) && s.findDailyErrs[index] != nil {
		return nil, s.findDailyErrs[index]
	}
	if index < len(s.findDailyValues) && s.findDailyValues[index] != nil {
		return s.findDailyValues[index], nil
	}
	return nil, entity.ErrDailyDiscoveryNotFound
}
func (*dailyRecommendationRepositoryStub) ListDaily(context.Context, uuid.UUID, int, int) ([]entity.Recommendation, int64, error) {
	return nil, 0, nil
}
func (*dailyRecommendationRepositoryStub) HasRecentNormal(context.Context, uuid.UUID, time.Time) (bool, error) {
	return false, nil
}
func (s *dailyRecommendationRepositoryStub) CountByArtist(context.Context, uuid.UUID) (map[uuid.UUID]int64, error) {
	return s.artistCounts, s.artistCountsErr
}
func (s *dailyRecommendationRepositoryStub) CountBySong(context.Context, uuid.UUID) (map[uuid.UUID]int64, error) {
	return s.songCounts, s.songCountsErr
}
func (*dailyRecommendationRepositoryStub) FindArtistWeights(context.Context, uuid.UUID) (map[uuid.UUID]float64, error) {
	return map[uuid.UUID]float64{}, nil
}
func (*dailyRecommendationRepositoryStub) FindSongWeights(context.Context, uuid.UUID) (map[uuid.UUID]float64, error) {
	return map[uuid.UUID]float64{}, nil
}
func (*dailyRecommendationRepositoryStub) FindRecommendedGenres(context.Context, uuid.UUID) ([]string, error) {
	return nil, nil
}
func (s *dailyRecommendationRepositoryStub) FindRecentDisplayCounts(context.Context, uuid.UUID, time.Time) (repository.RecommendationRecencyCounts, error) {
	return s.recency, s.recencyErr
}

func newDailyUsecaseForTest(repo *dailyRecommendationRepositoryStub, profile *dailyProfileRepositoryStub, builder IRecommendationConditionBuilder, now time.Time) *dailyDiscoveryUsecase {
	userID := uuid.New()
	return NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		profile,
		repo,
		&dailyTransactionStub{},
		dailyClockStub{now: now},
		builder,
		DailyDiscoveryConfig{CandidateLimit: 20, DuplicateDays: 30, RecencyDays: 30, MinimumMatchScore: 2, MinimumContactScore: 1},
	).(*dailyDiscoveryUsecase)
}

func dailyCandidate(artistID uuid.UUID, score RecommendationScore, genres ...string) SongCandidate {
	tags := make([]entity.Tag, 0, len(genres))
	for _, genre := range genres {
		tags = append(tags, entity.Tag{Category: "genre", Slug: genre})
	}
	return SongCandidate{
		Data: entity.RecommendationSongCandidate{
			Song:     entity.Song{ID: uuid.New(), ArtistID: artistID},
			Artist:   entity.Artist{ID: artistID},
			SongTags: tags,
		},
		Score: score,
	}
}

func TestDailyDiscoverySelectMatchesRequiresMinimumScoreAndPrefersDifferentArtist(t *testing.T) {
	u := &dailyDiscoveryUsecase{config: DailyDiscoveryConfig{MinimumMatchScore: 2}}
	artistOne := uuid.New()
	artistTwo := uuid.New()
	low := dailyCandidate(uuid.New(), RecommendationScore{Mood: .5})
	first := dailyCandidate(artistOne, RecommendationScore{Mood: 1, Genre: 1, Total: 10})
	sameArtist := dailyCandidate(artistOne, RecommendationScore{Mood: 1, Genre: 1, Total: 9})
	differentArtist := dailyCandidate(artistTwo, RecommendationScore{Mood: 1, Genre: 1, Total: 8})

	selectedFirst, selectedSecond, ok := u.selectMatches([]SongCandidate{low, first, sameArtist, differentArtist})
	if !ok {
		t.Fatal("expected two match songs")
	}
	if selectedFirst.Data.Song.ID != first.Data.Song.ID {
		t.Fatal("highest eligible song was not selected")
	}
	if selectedSecond.Data.Artist.ID != artistTwo {
		t.Fatal("different artist should be preferred")
	}
}

func TestDailyDiscoverySelectChallengeKeepsMinimumContactAndNoSongDuplication(t *testing.T) {
	u := &dailyDiscoveryUsecase{config: DailyDiscoveryConfig{MinimumContactScore: 1}}
	selectedSong := uuid.New()
	selectedArtist := uuid.New()
	noContact := dailyCandidate(uuid.New(), RecommendationScore{Discovery: 10}, "rock")
	valid := dailyCandidate(uuid.New(), RecommendationScore{Mood: 1, Discovery: 5}, "rock")
	condition := RecommendationCondition{Genres: []string{"rock"}, DiscoveryLevel: "low"}

	candidate, ok := u.selectChallenge(
		[]SongCandidate{noContact, valid}, condition, map[uuid.UUID]int64{},
		map[uuid.UUID]bool{selectedSong: true}, map[uuid.UUID]bool{selectedArtist: true},
	)
	if !ok {
		t.Fatal("expected challenge candidate")
	}
	if candidate.Data.Song.ID != valid.Data.Song.ID {
		t.Fatal("candidate without contact must be excluded")
	}
}

func TestDailyDiscoveryGetTodayReturnsExistingForUserTimezone(t *testing.T) {
	userID := uuid.New()
	existing := &entity.Recommendation{ID: uuid.New(), UserID: userID, Source: entity.RecommendationSourceDailyDiscovery}
	repo := &dailyRecommendationRepositoryStub{findDailyValues: []*entity.Recommendation{existing}}
	now := time.Date(2026, 8, 1, 15, 30, 0, 0, time.UTC) // Asia/Tokyo is 2026-08-02 00:30
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo,
		&dailyTransactionStub{}, dailyClockStub{now: now},
		&dailyConditionBuilderStub{}, DailyDiscoveryConfig{},
	)
	value, generated, err := u.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday error: %v", err)
	}
	if generated {
		t.Fatal("existing recommendation must return generated=false")
	}
	if value.ID != existing.ID {
		t.Fatal("existing recommendation was not returned")
	}
	if len(repo.findDailyDates) != 1 || repo.findDailyDates[0].Format("2006-01-02") != "2026-08-02" {
		t.Fatalf("target date=%v, want 2026-08-02", repo.findDailyDates)
	}
}

func TestDailyDiscoveryGetTodayMapsConditionBuilderFailure(t *testing.T) {
	userID := uuid.New()
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		&dailyRecommendationRepositoryStub{}, &dailyTransactionStub{}, dailyClockStub{now: time.Now()},
		&dailyConditionBuilderStub{err: entity.ErrRecommendationGenerateFailed}, DailyDiscoveryConfig{},
	)
	_, _, err := u.GetToday(context.Background(), userID)
	if !errors.Is(err, entity.ErrDailyDiscoveryGenerateFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestDailyDiscoveryGetTodayDoesNotIgnoreRepositoryError(t *testing.T) {
	userID := uuid.New()
	repo := &dailyRecommendationRepositoryStub{excludedErr: entity.ErrInternal}
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo, &dailyTransactionStub{}, dailyClockStub{now: time.Now()},
		&dailyConditionBuilderStub{condition: RecommendationCondition{}}, DailyDiscoveryConfig{DuplicateDays: 30},
	)
	_, _, err := u.GetToday(context.Background(), userID)
	if !errors.Is(err, entity.ErrDailyDiscoveryGenerateFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestDailyDiscoveryGetByIDDistinguishesNotFoundAndRepositoryFailure(t *testing.T) {
	userID := uuid.New()
	tests := []struct {
		name    string
		repoErr error
		want    error
	}{
		{"not found", entity.ErrRecommendationNotFound, entity.ErrDailyDiscoveryNotFound},
		{"database failure", entity.ErrInternal, entity.ErrDailyDiscoveryFetchFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &dailyDiscoveryUsecase{recommendations: &dailyRecommendationRepositoryStub{findByIDErr: test.repoErr}}
			_, err := u.GetByID(context.Background(), userID, uuid.New())
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
		})
	}
}

func dailyEntityCandidate(artistID uuid.UUID, mood, genre string) entity.RecommendationSongCandidate {
	return entity.RecommendationSongCandidate{
		Song:   entity.Song{ID: uuid.New(), ArtistID: artistID},
		Artist: entity.Artist{ID: artistID},
		SongTags: []entity.Tag{
			{Category: "mood", Slug: mood},
			{Category: "genre", Slug: genre},
		},
	}
}

func dailyGenerationConfig() DailyDiscoveryConfig {
	return DailyDiscoveryConfig{
		CandidateLimit:      20,
		DuplicateDays:       30,
		RecencyDays:         30,
		MinimumMatchScore:   1,
		MinimumContactScore: 1,
		AlgorithmVersion:    "daily-v1-test",
		MatchWeights: RecommendationTypeScoreWeights{
			Mood:  1,
			Genre: 1,
		},
		ChallengeWeights: RecommendationTypeScoreWeights{
			Mood:      1,
			Discovery: 1,
		},
	}
}

func TestDailyDiscoveryGetTodayCreatesThreeSongs(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC)
	saved := &entity.Recommendation{ID: uuid.New(), UserID: userID, Source: entity.RecommendationSourceDailyDiscovery}
	repo := &dailyRecommendationRepositoryStub{
		findDailyValues: []*entity.Recommendation{nil, nil, saved},
		candidates: []entity.RecommendationSongCandidate{
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
		},
		artistCounts: map[uuid.UUID]int64{},
		songCounts:   map[uuid.UUID]int64{},
		recency: repository.RecommendationRecencyCounts{
			ArtistCounts: map[uuid.UUID]int64{},
			GenreCounts:  map[string]int64{},
		},
	}
	condition := RecommendationCondition{
		Mood:              []string{"calm"},
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		GenreWeights:      map[string]float64{},
		ElementWeights:    map[string]float64{},
		AtmosphereWeights: map[string]float64{},
		ArtistWeights:     map[uuid.UUID]float64{},
		SongWeights:       map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{},
	}
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo,
		&dailyTransactionStub{},
		dailyClockStub{now: now},
		&dailyConditionBuilderStub{condition: condition},
		dailyGenerationConfig(),
	)

	value, generated, err := u.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday error: %v", err)
	}
	if !generated {
		t.Fatal("new recommendation must return generated=true")
	}
	if value.ID != saved.ID {
		t.Fatalf("returned recommendation=%s, want %s", value.ID, saved.ID)
	}
	if repo.created == nil {
		t.Fatal("recommendation was not created")
	}
	if repo.created.Source != entity.RecommendationSourceDailyDiscovery {
		t.Fatalf("source=%s", repo.created.Source)
	}
	if repo.created.TargetDate == nil || repo.created.TargetDate.Format("2006-01-02") != "2026-08-02" {
		t.Fatalf("target date=%v", repo.created.TargetDate)
	}
	if repo.created.AlgorithmVersion != "daily-v1-test" {
		t.Fatalf("algorithm version=%s", repo.created.AlgorithmVersion)
	}
	if len(repo.created.Details) != 3 {
		t.Fatalf("details=%d, want 3", len(repo.created.Details))
	}
	wantTypes := []entity.RecommendationType{
		entity.RecommendationTypeDailyMatchOne,
		entity.RecommendationTypeDailyMatchTwo,
		entity.RecommendationTypeDailyChallenge,
	}
	seenSongs := map[uuid.UUID]bool{}
	for index, detail := range repo.created.Details {
		if detail.Type != wantTypes[index] {
			t.Fatalf("detail[%d].type=%s, want %s", index, detail.Type, wantTypes[index])
		}
		if detail.Position != index+1 {
			t.Fatalf("detail[%d].position=%d, want %d", index, detail.Position, index+1)
		}
		if seenSongs[detail.SongID] {
			t.Fatalf("duplicate song ID: %s", detail.SongID)
		}
		seenSongs[detail.SongID] = true
	}
}

func TestDailyDiscoveryGetTodayReturnsSavedValueAfterConflict(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC)
	saved := &entity.Recommendation{ID: uuid.New(), UserID: userID, Source: entity.RecommendationSourceDailyDiscovery}
	repo := &dailyRecommendationRepositoryStub{
		findDailyValues: []*entity.Recommendation{nil, nil, saved},
		candidates: []entity.RecommendationSongCandidate{
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
		},
		artistCounts: map[uuid.UUID]int64{},
		songCounts:   map[uuid.UUID]int64{},
		recency: repository.RecommendationRecencyCounts{
			ArtistCounts: map[uuid.UUID]int64{},
			GenreCounts:  map[string]int64{},
		},
		createErr: entity.ErrConflict,
	}
	condition := RecommendationCondition{
		Mood:              []string{"calm"},
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		GenreWeights:      map[string]float64{},
		ElementWeights:    map[string]float64{},
		AtmosphereWeights: map[string]float64{},
		ArtistWeights:     map[uuid.UUID]float64{},
		SongWeights:       map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{},
	}
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo,
		&dailyTransactionStub{},
		dailyClockStub{now: now},
		&dailyConditionBuilderStub{condition: condition},
		dailyGenerationConfig(),
	)

	value, generated, err := u.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday error: %v", err)
	}
	if generated {
		t.Fatal("conflict recovery must return generated=false")
	}
	if value.ID != saved.ID {
		t.Fatalf("returned recommendation=%s, want %s", value.ID, saved.ID)
	}
}

func TestDailyDiscoveryGetTodayReturnsConflictWhenSavedValueCannotBeRecovered(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC)
	repo := &dailyRecommendationRepositoryStub{
		candidates: []entity.RecommendationSongCandidate{
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
		},
		artistCounts: map[uuid.UUID]int64{},
		songCounts:   map[uuid.UUID]int64{},
		recency: repository.RecommendationRecencyCounts{
			ArtistCounts: map[uuid.UUID]int64{},
			GenreCounts:  map[string]int64{},
		},
		createErr: entity.ErrConflict,
	}
	condition := RecommendationCondition{
		Mood:              []string{"calm"},
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		GenreWeights:      map[string]float64{},
		ElementWeights:    map[string]float64{},
		AtmosphereWeights: map[string]float64{},
		ArtistWeights:     map[uuid.UUID]float64{},
		SongWeights:       map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{},
	}
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo,
		&dailyTransactionStub{},
		dailyClockStub{now: now},
		&dailyConditionBuilderStub{condition: condition},
		dailyGenerationConfig(),
	)

	_, _, err := u.GetToday(context.Background(), userID)
	if !errors.Is(err, entity.ErrDailyDiscoveryGenerationConflict) {
		t.Fatalf("got %v, want %v", err, entity.ErrDailyDiscoveryGenerationConflict)
	}
}

func TestDailyDiscoveryGetTodayMapsPostSaveFetchFailure(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC)
	repo := &dailyRecommendationRepositoryStub{
		findDailyErrs: []error{nil, nil, entity.ErrInternal},
		candidates: []entity.RecommendationSongCandidate{
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
			dailyEntityCandidate(uuid.New(), "calm", "rock"),
		},
		artistCounts: map[uuid.UUID]int64{},
		songCounts:   map[uuid.UUID]int64{},
		recency: repository.RecommendationRecencyCounts{
			ArtistCounts: map[uuid.UUID]int64{},
			GenreCounts:  map[string]int64{},
		},
	}
	condition := RecommendationCondition{
		Mood:              []string{"calm"},
		Genres:            []string{"rock"},
		DiscoveryLevel:    "low",
		GenreWeights:      map[string]float64{},
		ElementWeights:    map[string]float64{},
		AtmosphereWeights: map[string]float64{},
		ArtistWeights:     map[uuid.UUID]float64{},
		SongWeights:       map[uuid.UUID]float64{},
		RecommendedGenres: map[string]bool{},
	}
	u := NewDailyDiscoveryUsecase(
		&dailyUserRepositoryStub{user: &entity.User{ID: userID, Status: entity.UserStatusActive}},
		&dailyProfileRepositoryStub{profile: &entity.UserProfile{Timezone: "Asia/Tokyo"}},
		repo,
		&dailyTransactionStub{},
		dailyClockStub{now: now},
		&dailyConditionBuilderStub{condition: condition},
		dailyGenerationConfig(),
	)

	_, _, err := u.GetToday(context.Background(), userID)
	if !errors.Is(err, entity.ErrDailyDiscoveryFetchFailed) {
		t.Fatalf("got %v, want %v", err, entity.ErrDailyDiscoveryFetchFailed)
	}
}
