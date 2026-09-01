package messagetemplate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

func TestPaginationDefaultsAndBounds(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantLimit  int32
		wantOffset int32
	}{
		{name: "defaults", target: "/templates", wantLimit: 50, wantOffset: 0},
		{name: "valid", target: "/templates?limit=25&offset=50", wantLimit: 25, wantOffset: 50},
		{name: "caps limit", target: "/templates?limit=101", wantLimit: 100, wantOffset: 0},
		{name: "negative limit defaults", target: "/templates?limit=-1", wantLimit: 50, wantOffset: 0},
		{name: "negative offset becomes zero", target: "/templates?offset=-10", wantLimit: 50, wantOffset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newPaginationTestContext(test.target)
			limit, offset, err := pagination(c)
			if err != nil {
				t.Fatalf("pagination() error = %v", err)
			}
			if limit != test.wantLimit || offset != test.wantOffset {
				t.Fatalf("pagination() = (%d, %d), want (%d, %d)", limit, offset, test.wantLimit, test.wantOffset)
			}
		})
	}
}

func TestPaginationRejectsMalformedValues(t *testing.T) {
	for name, target := range map[string]string{
		"invalid limit":  "/templates?limit=nope",
		"invalid offset": "/templates?offset=nope",
	} {
		t.Run(name, func(t *testing.T) {
			c := newPaginationTestContext(target)
			_, _, err := pagination(c)
			if !apperrors.IsCode(err, apperrors.CodeBadRequest) {
				t.Fatalf("pagination() error = %v, want bad request", err)
			}
		})
	}
}

func newPaginationTestContext(target string) *echo.Context {
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	response := httptest.NewRecorder()
	return router.NewContext(request, response)
}
