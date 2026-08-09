package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"music-discovery/backend/usecase"

	"github.com/labstack/echo/v4"
)

type IRateLimitMiddleware interface {
	SignUp(echo.HandlerFunc) echo.HandlerFunc
	Login(echo.HandlerFunc) echo.HandlerFunc
	Refresh(echo.HandlerFunc) echo.HandlerFunc
	Withdraw(echo.HandlerFunc) echo.HandlerFunc
	QuestionAnswers(echo.HandlerFunc) echo.HandlerFunc
	Recommendations(echo.HandlerFunc) echo.HandlerFunc
	DailyDiscovery(echo.HandlerFunc) echo.HandlerFunc
	Feedback(echo.HandlerFunc) echo.HandlerFunc
}

type rateLimitMiddleware struct{ usecase usecase.IRateLimitUsecase }

func NewRateLimitMiddleware(u usecase.IRateLimitUsecase) IRateLimitMiddleware {
	return &rateLimitMiddleware{usecase: u}
}
func (m *rateLimitMiddleware) SignUp(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("signup", next)
}
func (m *rateLimitMiddleware) Login(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("login", next)
}
func (m *rateLimitMiddleware) Refresh(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("refresh", next)
}
func (m *rateLimitMiddleware) Withdraw(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("withdraw", next)
}
func (m *rateLimitMiddleware) QuestionAnswers(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("question_answers", next)
}
func (m *rateLimitMiddleware) Recommendations(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("recommendations", next)
}
func (m *rateLimitMiddleware) DailyDiscovery(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("daily_discovery", next)
}
func (m *rateLimitMiddleware) Feedback(next echo.HandlerFunc) echo.HandlerFunc {
	return m.apply("feedback", next)
}

func (m *rateLimitMiddleware) apply(scope string, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := ""
		if value, ok := c.Get(AuthUserIDKey).(uuid.UUID); ok {
			userID = value.String()
		}
		email := ""
		if scope == "signup" || scope == "login" {
			email = readRateLimitEmail(c.Request())
		}
		decision, err := m.usecase.Allow(c.Request().Context(), usecase.RateLimitInput{Scope: scope, IP: c.RealIP(), UserID: userID, Email: email})
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "Rate Limit service is unavailable").SetInternal(err)
		}
		c.Response().Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(decision.Remaining, 'f', -1, 64))
		c.Response().Header().Set("X-RateLimit-Name", decision.LimitName)
		if decision.Allowed {
			return next(c)
		}
		c.Response().Header().Set("Retry-After", strconv.Itoa(int(decision.RetryAfter.Seconds())))
		return echo.NewHTTPError(http.StatusTooManyRequests)
	}
}

const maxRateLimitAuthBodyBytes int64 = 16 * 1024

func readRateLimitEmail(request *http.Request) string {
	if request == nil || request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maxRateLimitAuthBodyBytes+1))
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil || int64(len(body)) > maxRateLimitAuthBodyBytes {
		return ""
	}
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Email
}
