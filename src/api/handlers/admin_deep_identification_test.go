package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type fakeDeepObservabilityProvider struct {
	summary *models.DeepIdentificationObservabilitySummary
}

func (f fakeDeepObservabilityProvider) GetObservabilitySummary() (*models.DeepIdentificationObservabilitySummary, error) {
	return f.summary, nil
}

func deepObservabilityToken(t *testing.T, role models.UserRole) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(1),
		"role":   string(role),
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(adminNumistaTestJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestAdminDeepIdentificationObservabilityIsAdminOnlyAndRedacted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	summary := &models.DeepIdentificationObservabilitySummary{
		JobsByTerminalStatus: map[models.DeepJobStatus]int64{models.DeepJobStatusCompleted: 2},
		Providers: map[models.DeepProviderName]models.DeepIdentificationProviderMetrics{
			models.DeepProviderNumista: {
				StatusCounts: map[models.DeepProviderRunStatus]int64{models.DeepProviderRunContributed: 1},
				Latency:      models.DeepIdentificationLatencySummary{P50MS: 25, P95MS: 40},
			},
		},
		QueueDepth: 1,
	}
	handler := NewAdminDeepIdentificationHandler(fakeDeepObservabilityProvider{summary: summary})
	router := gin.New()
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthRequired(adminNumistaTestJWTSecret, nil))
	admin.Use(AdminRequired())
	admin.GET("/deep-identification/observability", handler.Observability)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/deep-identification/observability", nil)
	request.Header.Set("Authorization", "Bearer "+deepObservabilityToken(t, models.RoleAdmin))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("admin status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, forbidden := range []string{"notes", "query", "claims", "report", "token", "proposal"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("observability response exposed prohibited field %q: %s", forbidden, body)
		}
	}

	for _, test := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "unauthenticated", want: http.StatusUnauthorized},
		{name: "non-admin", token: deepObservabilityToken(t, models.RoleUser), want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/deep-identification/observability", nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, test.want, rec.Body.String())
			}
		})
	}
}
