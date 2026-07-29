package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-discovery/backend/entity"
	appMiddleware "music-discovery/backend/middleware"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type routerUserControllerStub struct{ calls map[string]int }

func (s *routerUserControllerStub) hit(name string, c echo.Context) error {
	s.calls[name]++
	return c.NoContent(http.StatusNoContent)
}
func (s *routerUserControllerStub) SignUp(c echo.Context) error      { return s.hit("signup", c) }
func (s *routerUserControllerStub) Login(c echo.Context) error       { return s.hit("login", c) }
func (s *routerUserControllerStub) Refresh(c echo.Context) error     { return s.hit("refresh", c) }
func (s *routerUserControllerStub) Logout(c echo.Context) error      { return s.hit("logout", c) }
func (s *routerUserControllerStub) CurrentUser(c echo.Context) error { return s.hit("current", c) }
func (s *routerUserControllerStub) Withdraw(c echo.Context) error    { return s.hit("withdraw", c) }

type routerQuestionControllerStub struct{}

func (routerQuestionControllerStub) ListQuestions(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
func (routerQuestionControllerStub) SaveAnswers(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
func (routerQuestionControllerStub) GetLatestAnswers(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

type routerRecommendationControllerStub struct{}

func (routerRecommendationControllerStub) Generate(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
func (routerRecommendationControllerStub) GetLatest(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
func (routerRecommendationControllerStub) GetByID(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

type routerAdminManagementControllerStub struct{ placeholder, changeRole int }

func (s *routerAdminManagementControllerStub) hit(c echo.Context) error {
	s.placeholder++
	return c.NoContent(http.StatusNoContent)
}
func (s *routerAdminManagementControllerStub) ListArtists(c echo.Context) error    { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetArtist(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) CreateArtist(c echo.Context) error   { return s.hit(c) }
func (s *routerAdminManagementControllerStub) UpdateArtist(c echo.Context) error   { return s.hit(c) }
func (s *routerAdminManagementControllerStub) DeleteArtist(c echo.Context) error   { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListSongs(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetSong(c echo.Context) error        { return s.hit(c) }
func (s *routerAdminManagementControllerStub) CreateSong(c echo.Context) error     { return s.hit(c) }
func (s *routerAdminManagementControllerStub) UpdateSong(c echo.Context) error     { return s.hit(c) }
func (s *routerAdminManagementControllerStub) DeleteSong(c echo.Context) error     { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListTags(c echo.Context) error       { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListTagOptions(c echo.Context) error { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetTag(c echo.Context) error         { return s.hit(c) }
func (s *routerAdminManagementControllerStub) CreateTag(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) UpdateTag(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) DeleteTag(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListUsers(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetUser(c echo.Context) error        { return s.hit(c) }
func (s *routerAdminManagementControllerStub) SetUserStatus(c echo.Context) error  { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListRoles(c echo.Context) error      { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ChangeRole(c echo.Context) error {
	s.changeRole++
	return c.NoContent(http.StatusNoContent)
}
func (s *routerAdminManagementControllerStub) ListAuditLogs(c echo.Context) error  { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetAuditLog(c echo.Context) error    { return s.hit(c) }
func (s *routerAdminManagementControllerStub) ListSystemLogs(c echo.Context) error { return s.hit(c) }
func (s *routerAdminManagementControllerStub) GetSystemLog(c echo.Context) error   { return s.hit(c) }

type passRateLimit struct{}

func (passRateLimit) SignUp(next echo.HandlerFunc) echo.HandlerFunc          { return next }
func (passRateLimit) Login(next echo.HandlerFunc) echo.HandlerFunc           { return next }
func (passRateLimit) Refresh(next echo.HandlerFunc) echo.HandlerFunc         { return next }
func (passRateLimit) Withdraw(next echo.HandlerFunc) echo.HandlerFunc        { return next }
func (passRateLimit) QuestionAnswers(next echo.HandlerFunc) echo.HandlerFunc { return next }
func (passRateLimit) Recommendations(next echo.HandlerFunc) echo.HandlerFunc { return next }

type testAuthMiddleware struct{ calls int }

func (m *testAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		m.calls++
		role := entity.Role(c.Request().Header.Get("X-Test-Role"))
		if !role.IsPersistable() {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		c.Set(appMiddleware.AuthUserIDKey, uuid.New())
		c.Set(appMiddleware.AuthUserRoleKey, role)
		return next(c)
	}
}

func newRoleTestRouter() (*echo.Echo, *routerUserControllerStub, *routerAdminManagementControllerStub, *testAuthMiddleware) {
	users := &routerUserControllerStub{calls: map[string]int{}}
	admins := &routerAdminManagementControllerStub{}
	auth := &testAuthMiddleware{}
	e := NewRouter(
		slog.New(slog.NewTextHandler(nilWriter{}, nil)),
		users,
		routerQuestionControllerStub{},
		routerRecommendationControllerStub{},
		admins,
		admins,
		admins,
		admins,
		admins,
		admins,
		passRateLimit{},
		auth,
	)
	return e, users, admins, auth
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }

func performRoleRequest(e *echo.Echo, method, path string, role entity.Role) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if role != "" {
		req.Header.Set("X-Test-Role", string(role))
	}
	if method == http.MethodPatch || method == http.MethodPost {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRouterPublicRoutesDoNotUseAuthMiddleware(t *testing.T) {
	e, users, _, auth := newRoleTestRouter()
	rec := performRoleRequest(e, http.MethodPost, "/api/auth/signup", "")
	if rec.Code != http.StatusNoContent || users.calls["signup"] != 1 || auth.calls != 0 {
		t.Fatalf("status=%d signup=%d auth=%d", rec.Code, users.calls["signup"], auth.calls)
	}
}

func TestRouterRoleProtectionAndMiddlewareOrder(t *testing.T) {
	tests := []struct {
		name, method, path string
		role               entity.Role
		wantStatus         int
		wantAdminCalls     int
		wantRoleCalls      int
	}{
		{name: "content allowed", method: http.MethodGet, path: "/api/admin/artists", role: entity.RoleContentAdmin, wantStatus: http.StatusNoContent, wantAdminCalls: 1},
		{name: "system denied content", method: http.MethodGet, path: "/api/admin/artists", role: entity.RoleSystemAdmin, wantStatus: http.StatusForbidden},
		{name: "user denied content", method: http.MethodGet, path: "/api/admin/artists", role: entity.RoleUser, wantStatus: http.StatusForbidden},
		{name: "system allowed system", method: http.MethodGet, path: "/api/system/users", role: entity.RoleSystemAdmin, wantStatus: http.StatusNoContent, wantAdminCalls: 1},
		{name: "content denied system", method: http.MethodGet, path: "/api/system/users", role: entity.RoleContentAdmin, wantStatus: http.StatusForbidden},
		{name: "system role change", method: http.MethodPatch, path: "/api/system/users/" + uuid.NewString() + "/role", role: entity.RoleSystemAdmin, wantStatus: http.StatusNoContent, wantRoleCalls: 1},
		{name: "unauthenticated stops before role", method: http.MethodGet, path: "/api/system/users", role: "", wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _, admins, auth := newRoleTestRouter()
			rec := performRoleRequest(e, tt.method, tt.path, tt.role)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if admins.placeholder != tt.wantAdminCalls || admins.changeRole != tt.wantRoleCalls {
				t.Fatalf("placeholder=%d changeRole=%d", admins.placeholder, admins.changeRole)
			}
			if auth.calls != 1 {
				t.Fatalf("auth calls=%d", auth.calls)
			}
		})
	}
}

func TestRouterDoesNotExposeLegacyAdminRoutes(t *testing.T) {
	e, _, _, _ := newRoleTestRouter()
	legacyRoutes := map[string]struct{}{
		http.MethodPatch + " /api/admin/protected":      {},
		http.MethodPatch + " /api/admin/users/:id/role": {},
	}

	for _, route := range e.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := legacyRoutes[key]; exists {
			t.Fatalf("legacy route is exposed: %s", key)
		}
	}
}

func TestQuestionRoutesAllowAllAuthenticatedRoles(t *testing.T) {
	roles := []entity.Role{entity.RoleUser, entity.RoleContentAdmin, entity.RoleSystemAdmin}
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/questions"},
		{http.MethodGet, "/api/user-answers/latest"},
		{http.MethodPost, "/api/user-answers"},
	}
	for _, role := range roles {
		for _, route := range paths {
			t.Run(string(role)+"_"+route.method+"_"+route.path, func(t *testing.T) {
				e, _, _, _ := newRoleTestRouter()
				rec := performRoleRequest(e, route.method, route.path, role)
				if rec.Code != http.StatusNoContent {
					t.Fatalf("role=%s method=%s path=%s status=%d body=%s", role, route.method, route.path, rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestRecommendationRoutesAllowAllAuthenticatedRoles(t *testing.T) {
	roles := []entity.Role{entity.RoleUser, entity.RoleContentAdmin, entity.RoleSystemAdmin}
	paths := []struct{ method, path string }{
		{http.MethodPost, "/api/recommendations"},
		{http.MethodGet, "/api/recommendations/latest"},
		{http.MethodGet, "/api/recommendations/" + uuid.NewString()},
	}
	for _, role := range roles {
		for _, route := range paths {
			t.Run(string(role)+"_"+route.method+"_"+route.path, func(t *testing.T) {
				e, _, _, _ := newRoleTestRouter()
				rec := performRoleRequest(e, route.method, route.path, role)
				if rec.Code != http.StatusNoContent {
					t.Fatalf("role=%s method=%s path=%s status=%d body=%s", role, route.method, route.path, rec.Code, rec.Body.String())
				}
			})
		}
	}
}
