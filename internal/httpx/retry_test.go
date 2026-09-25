package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReadAllRetries503Then200(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(503)
			return
		}
		io.WriteString(w, "hello")
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := ReadAll(srv.Client(), req, nil)
	if err != nil || string(resp.Body) != "hello" || calls != 3 {
		t.Fatalf("resp=%+v err=%v calls=%d", resp, err, calls)
	}
}
func TestReadAllDoesNotRetry404(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(404) }))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := ReadAll(srv.Client(), req, nil)
	if err != nil || resp.StatusCode != 404 || calls != 1 {
		t.Fatalf("resp=%+v err=%v calls=%d", resp, err, calls)
	}
}
func TestDownloadToFileRetriesTruncatedBody(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Length", "10")
		if calls == 1 {
			io.WriteString(w, "abc")
			return
		}
		io.WriteString(w, "0123456789")
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	dir := t.TempDir()
	dest := filepath.Join(dir, "001.jpg")
	if err := DownloadToFile(srv.Client(), req, dest, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "0123456789" || calls != 2 {
		t.Fatalf("body=%q err=%v calls=%d", b, err, calls)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("partial files remained: %+v, %v", entries, err)
	}
}
func TestDoKeepsResponseOpenForCaller(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := Do(srv.Client(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil || string(b) != "ok" {
		t.Fatalf("%q %v", b, err)
	}
}
func TestReadAllRetriesTruncatedBody(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Length", "10")
		if calls == 1 {
			io.WriteString(w, "abc")
			return
		}
		io.WriteString(w, "0123456789")
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	got, err := ReadAll(srv.Client(), req, nil)
	if err != nil || string(got.Body) != "0123456789" || calls != 2 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls)
	}
}
func TestNoRetryLongRetryAfter(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(429)
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	got, err := ReadAll(srv.Client(), req, nil)
	if err != nil || got.StatusCode != 429 || calls != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls)
	}
}
