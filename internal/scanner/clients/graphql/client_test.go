package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/wallarm/gotestwaf/internal/config"
)

type seenRequest struct {
	path    string
	host    string
	headers http.Header
}

func TestCheckAvailabilitySendsConfiguredHeaders(t *testing.T) {
	var (
		mu   sync.Mutex
		seen []seenRequest
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, seenRequest{path: r.URL.Path, host: r.Host, headers: r.Header.Clone()})
		mu.Unlock()

		if r.URL.Path != "/api/graphql" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"__typename":"Query"}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		URL:             srv.URL,
		GraphQLURL:      srv.URL,
		HTTPHeaders:     map[string]string{"X-From-Config": "config", "Host": "configured.example"},
		AddHeader:       "X-Added: cli",
		IdleConnTimeout: 1,
		MaxIdleConns:    1,
	}

	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	available, err := client.CheckAvailability(context.Background())
	if err != nil {
		t.Fatalf("CheckAvailability: %v", err)
	}
	if !available {
		t.Fatal("expected GraphQL to be detected as available")
	}
	if got, want := client.graphqlUrl, srv.URL+"/api/graphql"; got != want {
		t.Fatalf("graphqlUrl = %q, want %q", got, want)
	}

	// /graphql, /_graphql and /api/graphql are probed; the check stops at the first hit.
	if len(seen) != 3 {
		t.Fatalf("expected 3 probe requests, got %d", len(seen))
	}
	for _, req := range seen {
		if req.headers.Get("X-From-Config") != "config" {
			t.Errorf("%s: header from config file missing, got %v", req.path, req.headers)
		}
		if req.headers.Get("X-Added") != "cli" {
			t.Errorf("%s: header from --addHeader missing, got %v", req.path, req.headers)
		}
		if req.host != "configured.example" {
			t.Errorf("%s: Host = %q, want configured.example", req.path, req.host)
		}
	}
}

func TestCheckAvailabilityNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client, err := NewClient(&config.Config{URL: srv.URL, GraphQLURL: srv.URL, IdleConnTimeout: 1, MaxIdleConns: 1}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	available, err := client.CheckAvailability(context.Background())
	if err != nil {
		t.Fatalf("CheckAvailability: %v", err)
	}
	if available || client.IsAvailable() {
		t.Fatal("expected GraphQL to be reported as not available")
	}
}
