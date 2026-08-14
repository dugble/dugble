package httptransport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	httpmiddleware "github.com/dugble/dugble/server/internal/transport/middleware"
)

func TestRouterAppliesGeneralResponseMiddleware(t *testing.T) {
	t.Parallel()

	router, err := NewRouter(RouterConfig{CORSOrigins: []string{"https://example.com"}}, func(router *echo.Echo) error {
		router.GET("/resource", func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) })
		return nil
	})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get(echo.HeaderCacheControl); got != httpmiddleware.DefaultCacheControl {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := recorder.Header().Get(echo.HeaderXRequestID); got == "" {
		t.Fatal("X-Request-ID was not generated")
	}
	if got := recorder.Header().Get(echo.HeaderXCorrelationID); got == "" {
		t.Fatal("X-Correlation-ID was not generated")
	}
}

func TestRouterRejectsOversizedBodyBeforeHandler(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	router, err := NewRouter(RouterConfig{BodyLimit: 4, CORSOrigins: []string{"https://example.com"}}, func(router *echo.Echo) error {
		router.POST("/resource", func(c *echo.Context) error {
			handlerCalled = true
			return c.NoContent(http.StatusNoContent)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/resource", strings.NewReader("12345")))
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	if handlerCalled {
		t.Fatal("oversized request reached handler")
	}
}
