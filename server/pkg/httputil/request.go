package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const DefaultMaxRequestBodyBytes int64 = 1 << 20

// ReadBody reads and restores the request body while enforcing a maximum size.
func ReadBody(c *echo.Context, maxBytes int64) ([]byte, error) {
	if c == nil || c.Request() == nil {
		return nil, apperrors.NewBadRequest("Request is not available")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxRequestBodyBytes
	}
	request := c.Request()
	if request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maxBytes+1))
	if err != nil {
		return nil, apperrors.NewBadRequest("Unable to read request body")
	}
	if int64(len(body)) > maxBytes {
		return nil, apperrors.NewPayloadTooLarge("Request body is too large")
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// DecodeJSON decodes one strict JSON value from the request body.
func DecodeJSON(c *echo.Context, destination any, maxBytes int64) error {
	if destination == nil {
		return apperrors.NewInternal("JSON destination is not configured", nil)
	}
	body, err := ReadBody(c, maxBytes)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return apperrors.NewBadRequest("Request body is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return apperrors.NewBadRequest("Invalid JSON request body")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return apperrors.NewBadRequest("Request body must contain one JSON value")
	}
	return nil
}

// QueryInt32 parses an optional int32 query parameter. Invalid values return zero.
func QueryInt32(c *echo.Context, name string) int32 {
	if c == nil {
		return 0
	}
	value, err := strconv.ParseInt(c.QueryParam(name), 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

// Pagination parses optional limit and offset parameters, rejecting malformed
// numbers instead of silently treating them as omitted.
func Pagination(c *echo.Context) (limit int32, offset int32, err error) {
	limit, err = strictOptionalInt32(c, "limit")
	if err != nil {
		return 0, 0, err
	}
	offset, err = strictOptionalInt32(c, "offset")
	if err != nil {
		return 0, 0, err
	}
	return limit, offset, nil
}

func strictOptionalInt32(c *echo.Context, name string) (int32, error) {
	if c == nil {
		return 0, apperrors.NewBadRequest("Request is not available")
	}
	value := c.QueryParam(name)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, apperrors.NewBadRequest(name + " must be a valid integer")
	}
	return int32(parsed), nil
}
