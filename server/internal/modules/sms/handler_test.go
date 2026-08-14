package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func newHandlerTestContext(body string) (*echo.Context, *httptest.ResponseRecorder) {
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/sms", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	return router.NewContext(request, recorder), recorder
}
