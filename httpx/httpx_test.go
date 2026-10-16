package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		w.Header().Set("X-Echo-Type", r.Header.Get("Content-Type"))
		w.Header().Set("X-Echo-Auth", r.Header.Get("Authorization"))
		w.Header().Set("X-Echo-Trace", r.Header.Get("X-Trace"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := New(Options{Header: http.Header{"authorization": {"Bearer client"}, "X-Trace": {"client"}}})

	resp, err := c.Do(context.Background(), Request{
		Method: http.MethodPost,
		URL:    srv.URL,
		Header: http.Header{"x-trace": {"request"}},
		JSON:   map[string]any{"a": 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != http.StatusCreated || string(resp.Body) != `{"a":1}` {
		t.Errorf("Status, Body = %d, %s", resp.Status, resp.Body)
	}

	if got := resp.Header.Get("X-Echo-Type"); got != "application/json" {
		t.Errorf("Content-Type sent = %q, want application/json for a JSON body", got)
	}

	if auth, trace := resp.Header.Get("X-Echo-Auth"), resp.Header.Get("X-Echo-Trace"); auth != "Bearer client" || trace != "request" {
		t.Errorf("Authorization, X-Trace sent = %q, %q; want the client's, with the request's over it key by key", auth, trace)
	}

	if got := resp.RequestHeader.Values("X-Trace"); len(got) != 1 || got[0] != "request" {
		t.Errorf("RequestHeader X-Trace = %v, want only the request's", got)
	}

	var decoded map[string]int
	if err := resp.DecodeJSON(&decoded); err != nil || decoded["a"] != 1 {
		t.Errorf("DecodeJSON = %v, %v", decoded, err)
	}
}

func TestDoRawBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(append([]byte(r.Header.Get("Content-Type")+":"), body...))
	}))
	defer srv.Close()

	resp, err := New(Options{}).Do(context.Background(), Request{Method: http.MethodPut, URL: srv.URL, Body: []byte("raw")})
	if err != nil || string(resp.Body) != ":raw" {
		t.Errorf("Do = %q, %v; want the body as is, with no Content-Type added", resp.Body, err)
	}
}

func TestDoErrRequest(t *testing.T) {
	c := New(Options{})

	for name, r := range map[string]Request{
		"json": {Method: http.MethodPost, URL: "http://localhost", JSON: make(chan int)},
		"url":  {Method: http.MethodGet, URL: "http://[::1"},
	} {
		if _, err := c.Do(context.Background(), r); !errors.Is(err, ErrRequest) {
			t.Errorf("%s: Do = %v, want ErrRequest", name, err)
		}
	}
}

func TestDoTimeout(t *testing.T) {
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	start := time.Now()

	_, err := New(Options{Timeout: 20 * time.Millisecond}).Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL})
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrRequest) || time.Since(start) > 5*time.Second {
		t.Errorf("Do = %v; want the client's timeout for a context without a deadline", err)
	}
}
