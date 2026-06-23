package fakeweb

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSinkDial(t *testing.T) {
	now := time.Now()
	counter := 0
	h := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(
				w, "only get method supported",
				http.StatusMethodNotAllowed,
			)
			return
		}
		counter++
		http.ServeContent(w, req, "index.html", now, strings.NewReader("fake"))
	}

	s := httptest.NewServer(http.HandlerFunc(h))
	defer s.Close()

	serverAddr := s.Listener.Addr().String()
	client := sinkClient(serverAddr)

	req := &http.Request{
		URL: &url.URL{Scheme: "http", Host: "fake.example.com"},
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("issue request: ", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("read body: ", err)
	}
	if string(body) != "fake" {
		t.Errorf("got body %q, want `fake`", body)
	}

	if counter != 1 {
		t.Errorf("got counter %d, want 1", counter)
	}
}
