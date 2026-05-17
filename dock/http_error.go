package dock

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"shanhu.io/std/errcode"
)

func isSuccess(resp *http.Response) bool {
	return resp.StatusCode/100 == 2
}

type httpError struct {
	StatusCode int
	Status     string
	Body       string
}

func (err *httpError) Error() string {
	if err.Body != "" {
		return fmt.Sprintf("%s - %s", err.Status, err.Body)
	}
	return err.Status
}

func httpErrorStatusCode(err error) int {
	herr, ok := err.(*httpError)
	if !ok {
		return 0
	}
	return herr.StatusCode
}

func httpAddErrCode(statusCode int, err error) error {
	switch statusCode {
	case http.StatusNotFound:
		err = errcode.Add(errcode.NotFound, err)
	case http.StatusUnauthorized, http.StatusForbidden:
		err = errcode.Add(errcode.Unauthorized, err)
	case http.StatusBadRequest:
		err = errcode.Add(errcode.InvalidArg, err)
	}
	return err
}

func httpRespError(resp *http.Response) error {
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	herr := &httpError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       strings.TrimSpace(string(bs)),
	}
	return httpAddErrCode(resp.StatusCode, herr)
}
