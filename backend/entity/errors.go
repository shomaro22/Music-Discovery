package entity

import "errors"

var (
	ErrInvalidInput             = errors.New("invalid input")
	ErrInvalidUserForCreate     = errors.New("invalid user for create")
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrUserNotFound             = errors.New("user not found")
	ErrUserAlreadyWithdrawn     = errors.New("user already withdrawn")
	ErrInternal                 = errors.New("internal error")
	ErrProfileFetchFailed       = errors.New("profile fetch failed")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrUserSuspended            = errors.New("user suspended")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrRefreshTokenReused       = errors.New("refresh token reused")
	ErrCurrentPasswordMismatch  = errors.New("current password mismatch")
	ErrInvalidRoleConfiguration = errors.New("invalid role configuration")
	ErrForbidden                = errors.New("forbidden")
	ErrNotFound                 = errors.New("not found")
	ErrConflict                 = errors.New("conflict")
	ErrArtistAlreadyExists      = errors.New("artist already exists")
	ErrAlreadyActive            = errors.New("user already active")
	ErrAlreadySuspended         = errors.New("user already suspended")
	ErrLastSystemAdmin          = errors.New("last active system admin")
	ErrReasonRequired           = errors.New("reason required")
	ErrQuestionConfiguration    = errors.New("question configuration invalid")
	ErrRequiredQuestionMissing  = errors.New("required question missing")
	ErrDuplicateQuestion        = errors.New("duplicate question")
	ErrInvalidSelectionCount    = errors.New("invalid selection count")
	ErrDuplicateChoice          = errors.New("duplicate choice")
	ErrQuestionNotFound         = errors.New("question not found")
	ErrChoiceNotFound           = errors.New("choice not found")
	ErrQuestionChoiceMismatch   = errors.New("question choice mismatch")
	ErrAnswerSaveFailed         = errors.New("answer save failed")
	ErrProfileCreateFailed      = errors.New("profile create failed")
)

type APIErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

var (
	ErrQuestionAnswerRequired                        = errors.New("question answer required")
	ErrUserProfileRequired                           = errors.New("user profile required")
	ErrInvalidRecommendationID                       = errors.New("invalid recommendation id")
	ErrRecommendationAccessDenied                    = errors.New("recommendation access denied")
	ErrRecommendationNotFound                        = errors.New("recommendation not found")
	ErrRecommendationAlreadyGenerated                = errors.New("recommendation already generated")
	ErrRecommendationCandidatesNotFound              = errors.New("recommendation candidates not found")
	ErrRecommendationCandidatesImportedPendingReview = errors.New("recommendation candidates imported pending review")
	ErrSpotifyCandidateImportFailed                  = errors.New("spotify candidate import failed")
	ErrCuratedCatalogImportFailed                    = errors.New("curated catalog import failed")
	ErrSpotifyUnavailable                            = errors.New("spotify unavailable")
	ErrRecommendationGenerateFailed                  = errors.New("recommendation generate failed")
	ErrRecommendationSaveFailed                      = errors.New("recommendation save failed")
	ErrInvalidWeightConfig                           = errors.New("invalid recommendation weight config")
	ErrInvalidGenreRelation                          = errors.New("invalid genre relation")
	ErrInvalidRecommendationType                     = errors.New("invalid recommendation type")
	ErrScoreCalculationFailed                        = errors.New("score calculation failed")
	ErrRecommendationSelectionFailed                 = errors.New("recommendation selection failed")
	ErrDuplicateRecommendationSong                   = errors.New("duplicate recommendation song")
	ErrRecommendationReasonGenerateFailed            = errors.New("recommendation reason generate failed")
	ErrDailyDiscoveryAccessDenied                    = errors.New("daily discovery access denied")
	ErrDailyDiscoveryNotFound                        = errors.New("daily discovery not found")
	ErrDailyDiscoveryCandidatesNotFound              = errors.New("daily discovery candidates not found")
	ErrDailyDiscoveryGenerationConflict              = errors.New("daily discovery generation conflict")
	ErrDailyDiscoveryGenerateFailed                  = errors.New("daily discovery generate failed")
	ErrDailyDiscoverySaveFailed                      = errors.New("daily discovery save failed")
	ErrDailyDiscoveryFetchFailed                     = errors.New("daily discovery fetch failed")
)

var (
	ErrInvalidRecommendationDetailID = errors.New("invalid recommendation detail id")
	ErrInvalidFeedbackType           = errors.New("invalid feedback type")
	ErrFeedbackAccessDenied          = errors.New("feedback access denied")
	ErrRecommendationDetailNotFound  = errors.New("recommendation detail not found")
	ErrSongNotFound                  = errors.New("song not found")
	ErrArtistNotFound                = errors.New("artist not found")
	ErrFeedbackNotChanged            = errors.New("feedback not changed")
	ErrFeedbackSaveFailed            = errors.New("feedback save failed")
	ErrFeedbackUpdateFailed          = errors.New("feedback update failed")
)

var (
	ErrInvalidPage                       = errors.New("invalid page")
	ErrInvalidLimit                      = errors.New("invalid limit")
	ErrInvalidRecommendationSource       = errors.New("invalid recommendation source")
	ErrInvalidDateRange                  = errors.New("invalid date range")
	ErrRecommendationHistoryAccessDenied = errors.New("recommendation history access denied")
	ErrRecommendationHistoryNotFound     = errors.New("recommendation history not found")
	ErrRecommendationHistoryFetchFailed  = errors.New("recommendation history fetch failed")
)

var ErrInvalidPasswordResetToken = errors.New("invalid password reset token")
