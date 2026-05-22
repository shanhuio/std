package docker

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"shanhu.io/std/errcode"
)

func TestHTTPAddErrCode(t *testing.T) {
	base := errors.New("oops")
	for _, c := range []struct {
		name   string
		status int
		check  func(error) bool
	}{
		{"InvalidArg/400", http.StatusBadRequest, errcode.IsInvalidArg},
		{"Unauthorized/401", http.StatusUnauthorized, errcode.IsUnauthorized},
		{"Unauthorized/403", http.StatusForbidden, errcode.IsUnauthorized},
		{"NotFound/404", http.StatusNotFound, errcode.IsNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := httpAddErrCode(c.status, base)
			if !c.check(err) {
				t.Errorf("status %d: check failed, got %v", c.status, err)
			}
		})
	}

	t.Run("500 unmapped", func(t *testing.T) {
		err := httpAddErrCode(http.StatusInternalServerError, base)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errcode.IsInvalidArg(err) || errcode.IsUnauthorized(err) ||
			errcode.IsNotFound(err) {
			t.Errorf("500 should not match any specific errcode, got %v", err)
		}
	})
}

func TestHTTPError(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		e := &httpError{StatusCode: 404, Status: "404 Not Found", Body: "missing"}
		s := e.Error()
		if !strings.Contains(s, "404 Not Found") || !strings.Contains(s, "missing") {
			t.Errorf("got %q", s)
		}
	})

	t.Run("without body", func(t *testing.T) {
		e := &httpError{StatusCode: 500, Status: "500 Internal Server Error"}
		if got := e.Error(); got != "500 Internal Server Error" {
			t.Errorf("got %q, want %q", got, "500 Internal Server Error")
		}
	})
}

func TestHTTPErrorStatusCode(t *testing.T) {
	if got := httpErrorStatusCode(&httpError{StatusCode: 304}); got != 304 {
		t.Errorf("got %d, want 304", got)
	}
	if got := httpErrorStatusCode(errors.New("plain")); got != 0 {
		t.Errorf("plain error: got %d, want 0", got)
	}
	if got := httpErrorStatusCode(nil); got != 0 {
		t.Errorf("nil: got %d, want 0", got)
	}
}

func TestHTTPRespError(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		body   string
		check  func(error) bool
	}{
		{"400", http.StatusBadRequest, "bad request body", errcode.IsInvalidArg},
		{"401", http.StatusUnauthorized, "no token", errcode.IsUnauthorized},
		{"500 no body", http.StatusInternalServerError, "", func(err error) bool {
			// 500 has no errcode mapping; the returned *httpError is bare.
			return httpErrorStatusCode(err) == http.StatusInternalServerError
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: c.status,
				Status:     http.StatusText(c.status),
				Body:       io.NopCloser(strings.NewReader(c.body)),
			}
			err := httpRespError(resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if c.body != "" && !strings.Contains(err.Error(), c.body) {
				t.Errorf("expected body %q in error %q", c.body, err.Error())
			}
			if !c.check(err) {
				t.Errorf("check failed for %d, got %v", c.status, err)
			}
		})
	}
}
