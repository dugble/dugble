package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestCacheControlDefaultsAndAllowsExplicitOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set  string
		want string
	}{
		{name: "default", want: DefaultCacheControl},
		{name: "explicit public policy", set: "public, max-age=60", want: "public, max-age=60"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			ctx := echo.New().NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), recorder)
			handler := CacheControl(func(c *echo.Context) error {
				if test.set != "" {
					c.Response().Header().Set(echo.HeaderCacheControl, test.set)
				}
				return c.NoContent(http.StatusNoContent)
			})
			if err := handler(ctx); err != nil {
				t.Fatalf("CacheControl() error = %v", err)
			}
			if got := recorder.Header().Get(echo.HeaderCacheControl); got != test.want {
				t.Fatalf("Cache-Control = %q, want %q", got, test.want)
			}
			if got := recorder.Header().Values("Vary"); len(got) != 2 {
				t.Fatalf("Vary = %v", got)
			}
		})
	}
}
