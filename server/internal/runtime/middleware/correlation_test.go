package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRequestCorrelation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		incoming   string
		requestID  string
		want       string
		wantCustom bool
	}{
		{name: "accepts safe incoming id", incoming: "mobile-request-42", requestID: "request-id", want: "mobile-request-42"},
		{name: "falls back to request id", incoming: strings.Repeat("x", maxCorrelationIDLength+1), requestID: "request-id", want: "request-id"},
		{name: "generates when absent", wantCustom: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Set(echo.HeaderXCorrelationID, test.incoming)
			recorder := httptest.NewRecorder()
			ctx := echo.New().NewContext(request, recorder)
			ctx.Response().Header().Set(echo.HeaderXRequestID, test.requestID)
			handler := RequestCorrelation(func(*echo.Context) error { return nil })
			if err := handler(ctx); err != nil {
				t.Fatalf("RequestCorrelation() error = %v", err)
			}
			got := request.Header.Get(echo.HeaderXCorrelationID)
			if got == "" || recorder.Header().Get(echo.HeaderXCorrelationID) != got {
				t.Fatalf("correlation headers were not synchronized: %q", got)
			}
			if test.wantCustom {
				if got == test.incoming || got == test.requestID {
					t.Fatalf("correlation id was not generated: %q", got)
				}
			} else if got != test.want {
				t.Fatalf("correlation id = %q, want %q", got, test.want)
			}
		})
	}
}
