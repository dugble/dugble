package smscampaign

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coffeyvidzro/dugble/server/internal/authz"
	"github.com/labstack/echo/v5"
)

func TestCampaignCreateRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	router := echo.New()
	RegisterRoutes(router, NewHandler(&Service{}), passthroughAccess)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/campaigns", strings.NewReader(`{
		"name":"Launch","segment_id":"11111111-1111-4111-8111-111111111111",
		"sender_id":"11111111-1111-4111-8111-111111111111","body":"Hello","template":"not-supported"
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func passthroughAccess(authz.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

func TestCampaignListRejectsMalformedPagination(t *testing.T) {
	t.Parallel()
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/campaigns?limit=invalid", nil)
	recorder := httptest.NewRecorder()
	context := router.NewContext(request, recorder)
	if err := (&Handler{}).List(context); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestRecordOptOutRejectsRemovedKeywordField(t *testing.T) {
	t.Parallel()
	router := echo.New()
	RegisterRoutes(router, NewHandler(&Service{}), passthroughAccess)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/sms/opt-outs", strings.NewReader(`{"phone":"+233200000000","keyword":"STOP"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}
