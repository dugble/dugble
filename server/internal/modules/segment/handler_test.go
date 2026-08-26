package segment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestListRejectsMalformedPagination(t *testing.T) {
	t.Parallel()

	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/segments?offset=not-a-number", nil)
	recorder := httptest.NewRecorder()

	if err := (&Handler{}).List(router.NewContext(request, recorder)); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestListContactsRejectsMalformedPagination(t *testing.T) {
	t.Parallel()

	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/segments/id/contacts?limit=not-a-number", nil)
	recorder := httptest.NewRecorder()

	if err := (&Handler{}).ListContacts(router.NewContext(request, recorder)); err != nil {
		t.Fatalf("ListContacts() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
