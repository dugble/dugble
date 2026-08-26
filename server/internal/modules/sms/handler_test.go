package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/platform/idempotency"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

func TestHandlersRejectNonStrictJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		body   string
		handle func(*Handler, *echo.Context) error
	}{
		{name: "send unknown field", body: `{"to":"+233200000000","from":"Dugble","body":"hello","unknown":true}`, handle: (*Handler).Send},
		{name: "send removed tags field", body: `{"to":"+233200000000","from":"Dugble","body":"hello","tags":[{"name":"campaign","value":"launch"}]}`, handle: (*Handler).Send},
		{name: "batch trailing value", body: `[{"to":"+233200000000","from":"Dugble","body":"hello"}] {}`, handle: (*Handler).BatchSend},
		{name: "update unknown field", body: `{"scheduled_at":"2026-08-09T10:30:00Z","unknown":true}`, handle: (*Handler).Update},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			context, recorder := newHandlerTestContext(test.body)
			context.Request().Header.Set(idempotency.Header, "request-key")

			if err := test.handle(&Handler{}, context); err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
			if !strings.Contains(recorder.Body.String(), apperrors.CodeBadRequest) {
				t.Fatalf("body = %s, want %s", recorder.Body.String(), apperrors.CodeBadRequest)
			}
		})
	}
}

func TestBatchSendRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	body := strings.Repeat(" ", int(httputil.DefaultMaxRequestBodyBytes)+1)
	context, recorder := newHandlerTestContext(body)
	context.Request().Header.Set(idempotency.Header, "batch-key")

	if err := (&Handler{}).BatchSend(context); err != nil {
		t.Fatalf("BatchSend() error = %v", err)
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(recorder.Body.String(), apperrors.CodePayloadTooLarge) {
		t.Fatalf("body = %s, want %s", recorder.Body.String(), apperrors.CodePayloadTooLarge)
	}
}

func TestBatchSendRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()
	context, recorder := newHandlerTestContext(`[{"to":"+233200000000","from":"Dugble","body":"hello"}]`)
	if err := (&Handler{}).BatchSend(context); err != nil {
		t.Fatalf("BatchSend() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestListRejectsMalformedPagination(t *testing.T) {
	t.Parallel()

	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sms?limit=not-a-number", nil)
	recorder := httptest.NewRecorder()
	context := router.NewContext(request, recorder)

	if err := (&Handler{}).List(context); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestListRequestParsesFilters(t *testing.T) {
	t.Parallel()

	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sms?limit=25&offset=50&status=DELIVERED&sender=%20Dugble%20&start_date=2026-08-01T00:00:00Z&end_date=2026-08-31T23:59:59Z&search=%20receipt%20", nil)
	recorder := httptest.NewRecorder()

	req, err := listRequest(router.NewContext(request, recorder))
	if err != nil {
		t.Fatalf("listRequest() error = %v", err)
	}
	if req.Limit != 25 || req.Offset != 50 || req.Status != StatusDelivered || req.Sender != "Dugble" || req.Search != "receipt" {
		t.Fatalf("listRequest() = %+v, want parsed filters", req)
	}
	if req.StartDate == nil || req.StartDate.Format(time.RFC3339) != "2026-08-01T00:00:00Z" {
		t.Fatalf("start date = %v, want 2026-08-01T00:00:00Z", req.StartDate)
	}
	if req.EndDate == nil || req.EndDate.Format(time.RFC3339) != "2026-08-31T23:59:59Z" {
		t.Fatalf("end date = %v, want 2026-08-31T23:59:59Z", req.EndDate)
	}
}

func TestListRequestRejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	tests := []string{
		"/sms?status=invalid",
		"/sms?start_date=not-a-date",
		"/sms?start_date=2026-08-02T00:00:00Z&end_date=2026-08-01T00:00:00Z",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			router := echo.New()
			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			if _, err := listRequest(router.NewContext(request, recorder)); !apperrors.IsCode(err, apperrors.CodeBadRequest) {
				t.Fatalf("listRequest() error = %v, want bad request", err)
			}
		})
	}
}

func newHandlerTestContext(body string) (*echo.Context, *httptest.ResponseRecorder) {
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/sms", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	return router.NewContext(request, recorder), recorder
}
