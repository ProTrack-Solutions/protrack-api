package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fakeModules libera só os módulos do mapa
type fakeModules struct {
	domain.ServiceInterface
	allowed map[string]bool
}

func (f fakeModules) DepartmentHasModule(_ context.Context, _ uuid.UUID, code string) (bool, error) {
	return f.allowed[code], nil
}

func runAnyModule(t *testing.T, role string, dep uuid.UUID, allowed map[string]bool) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		c.Set("role", role)
		if dep != uuid.Nil {
			c.Set("department_id", dep)
		}
		c.Next()
	}, RequireAnyModule(fakeModules{allowed: allowed}, "inventory", "sales"), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(GrantedModuleKey))
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	return w.Code, w.Body.String()
}

func TestRequireAnyModule(t *testing.T) {
	dep := uuid.New()
	cases := []struct {
		name     string
		role     string
		dep      uuid.UUID
		allowed  map[string]bool
		wantCode int
		wantBody string
	}{
		{"admin passa sem departamento", "ADMIN", uuid.Nil, nil, 200, "ADMIN"},
		{"estoque tem preferência", "USER", dep, map[string]bool{"inventory": true, "sales": true}, 200, "inventory"},
		{"só vendas", "USER", dep, map[string]bool{"sales": true}, 200, "sales"},
		{"nenhum módulo", "USER", dep, map[string]bool{"reports": true}, 403, ""},
		{"sem departamento", "USER", uuid.Nil, map[string]bool{"sales": true}, 403, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body := runAnyModule(t, tc.role, tc.dep, tc.allowed)
			if code != tc.wantCode || (tc.wantCode == 200 && body != tc.wantBody) {
				t.Fatalf("got %d %q, want %d %q", code, body, tc.wantCode, tc.wantBody)
			}
		})
	}
}
