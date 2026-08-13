package httputil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

func TestReadBodyRestoresRequestBody(t *testing.T) {
	c := newTestContext(http.MethodPost, "/", `{"value":"test"}`)

	body, err := ReadBody(c, 1024)
	if err != nil {
		t.Fatalf("ReadBody() error = %v", err)
	}
	if got := string(body); got != `{"value":"test"}` {
		t.Fatalf("body = %q", got)
	}
	restored, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if got := string(restored); got != string(body) {
		t.Fatalf("restored body = %q, want %q", got, body)
	}
}

func TestReadBodyRejectsOversizedBody(t *testing.T) {
	c := newTestContext(http.MethodPost, "/", "1234")

	err := func() error {
		_, readErr := ReadBody(c, 3)
		return readErr
	}()
	if !apperrors.IsCode(err, apperrors.CodePayloadTooLarge) {
		t.Fatalf("ReadBody() error = %v, want payload too large", err)
	}
}

func TestDecodeJSONIsStrict(t *testing.T) {
	type request struct {
		Value string `json:"value"`
	}

	for name, body := range map[string]string{
		"unknown field":  `{"value":"test","extra":true}`,
		"trailing value": `{"value":"test"} {"value":"again"}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestContext(http.MethodPost, "/", body)
			var decoded request
			if err := DecodeJSON(c, &decoded, 1024); err == nil {
				t.Fatal("DecodeJSON() error = nil")
			}
		})
	}
}

func TestQueryInt32(t *testing.T) {
	c := newTestContext(http.MethodGet, "/?valid=42&invalid=nope&overflow=2147483648", "")

	if got := QueryInt32(c, "valid"); got != 42 {
		t.Fatalf("valid = %d", got)
	}
	if got := QueryInt32(c, "invalid"); got != 0 {
		t.Fatalf("invalid = %d", got)
	}
	if got := QueryInt32(c, "overflow"); got != 0 {
		t.Fatalf("overflow = %d", got)
	}
}

func TestPaginationRejectsMalformedValues(t *testing.T) {
	for name, target := range map[string]string{
		"invalid limit":   "/?limit=nope",
		"overflow offset": "/?offset=2147483648",
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestContext(http.MethodGet, target, "")
			if _, _, err := Pagination(c); !apperrors.IsCode(err, apperrors.CodeBadRequest) {
				t.Fatalf("Pagination() error = %v, want bad request", err)
			}
		})
	}
}

func TestPaginationAcceptsOmittedAndValidValues(t *testing.T) {
	for name, target := range map[string]string{
		"omitted": "/",
		"valid":   "/?limit=25&offset=50",
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestContext(http.MethodGet, target, "")
			limit, offset, err := Pagination(c)
			if err != nil {
				t.Fatalf("Pagination() error = %v", err)
			}
			if name == "valid" && (limit != 25 || offset != 50) {
				t.Fatalf("Pagination() = (%d, %d), want (25, 50)", limit, offset)
			}
		})
	}
}

func newTestContext(method, target, body string) *echo.Context {
	router := echo.New()
	request := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	response := httptest.NewRecorder()
	return router.NewContext(request, response)
}
