package fakeweb

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	now := time.Now()
	h := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "only supports get", http.StatusMethodNotAllowed)
			return
		}

		r := strings.NewReader("hello")
		http.ServeContent(w, req, "index.html", now, r)
	}

	s, err := NewServer("shanhu.io", http.HandlerFunc(h))
	if err != nil {
		t.Fatal("create server:", err)
	}
	defer s.Close()

	client := s.Client()

	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "shanhu.io",
			Path:   "/",
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("request error: ", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want ok", resp.StatusCode)
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal("read body: ", err)
		}

		if string(body) != "hello" {
			t.Errorf("body got %q, want `hello`", body)
		}
	}
}
