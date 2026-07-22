package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"music-discovery/backend/usecase"

	"github.com/labstack/echo/v4"
)

type IRateLimitMiddleware interface {
	SignUp(echo.HandlerFunc) echo.HandlerFunc
	Login(echo.HandlerFunc) echo.HandlerFunc
	Refresh(echo.HandlerFunc) echo.HandlerFunc
	Withdraw(echo.HandlerFunc) echo.HandlerFunc
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

func (m *rateLimitMiddleware) apply(scope string, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := ""
		if (scope == "login" || scope == "signup") && c.Request().Body != nil {
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "Rate Limit service is unavailable").SetInternal(err)
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			var input struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(body, &input); err == nil {
				email = input.Email
			}
		}
		decision, err := m.usecase.Allow(c.Request().Context(), usecase.RateLimitInput{Scope: scope, IP: c.RealIP(), Email: email})
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
