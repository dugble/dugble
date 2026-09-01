package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/platform/idempotency"
	"github.com/dugble/dugble/server/internal/security/authz"
)

func TestRegisteredEmailRoutesRejectRemovedStreamField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "single email", path: "/emails", body: `{"stream":"marketing","to":["recipient@example.com"],"subject":"Subject","text":"Body"}`},
		{name: "batch envelope", path: "/emails/batch", body: `{"messages":[{"stream":"marketing","to":["recipient@example.com"],"subject":"Subject","text":"Body"}]}`},
		{name: "batch array", path: "/emails/batch", body: `[{"stream":"marketing","to":["recipient@example.com"],"subject":"Subject","text":"Body"}]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			router := echo.New()
			RegisterRoutes(router, NewHandler(&Service{}), passthroughTenantAccess)
			request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set(idempotency.Header, "email-1")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"code":"BAD_REQUEST"`) {
				t.Fatalf("body = %s, want BAD_REQUEST error", recorder.Body.String())
			}
		})
	}
}

func passthroughTenantAccess(authz.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

func TestListRejectsMalformedPagination(t *testing.T) {
	t.Parallel()

	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/emails?offset=not-a-number", nil)
	recorder := httptest.NewRecorder()
	context := router.NewContext(request, recorder)

	if err := (&Handler{}).List(context); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestBatchSendRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/emails/batch", strings.NewReader(`[{"to":["recipient@example.com"],"subject":"Subject","text":"Body"}]`))
	recorder := httptest.NewRecorder()
	context := router.NewContext(request, recorder)
	if err := (&Handler{}).BatchSend(context); err != nil {
		t.Fatalf("BatchSend() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
