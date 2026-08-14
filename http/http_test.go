package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("a"); got != "1" {
			t.Errorf("query a = %q, want 1", got)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cli := NewHTTPClient()
	resp, err := cli.HTTPGet(srv.URL, nil, map[string]string{"a": "1"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := ResponseToString(resp, err)
	if err != nil {
		t.Fatal(err)
	}
	if body != "ok" {
		t.Fatalf("body = %q, want ok", body)
	}
}

func TestHTTPGetSendsHeadersAndUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "yes" {
			t.Errorf("X-Custom = %q, want yes", r.Header.Get("X-Custom"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header should be set")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cli := NewHTTPClient()
	resp, err := cli.HTTPGet(srv.URL, map[string]string{"X-Custom": "yes"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestHTTPFormPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("k"); got != "v" {
			t.Errorf("form k = %q, want v", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cli := NewHTTPClient()
	resp, err := cli.HTTPFormPost(srv.URL, nil, map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestHTTPRequestCustomMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cli := NewHTTPClient()
	resp, err := cli.HTTPRequest("DELETE", srv.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestResponseToStringWithNilResponse(t *testing.T) {
	body, err := ResponseToString(nil, nil)
	if body != "" || err != nil {
		t.Fatalf("ResponseToString(nil, nil) = (%q, %v), want (\"\", nil)", body, err)
	}
}
