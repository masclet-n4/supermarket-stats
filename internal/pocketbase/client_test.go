package pocketbase

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/collections/_superusers/auth-with-password" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"jwt"}`))
	}))
	defer server.Close()
	c := New(server.URL, "admin@example.com", "secret", server.Client())
	if err := c.Authenticate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.token != "jwt" {
		t.Fatal("unexpected token")
	}
}

func TestRequestRefreshesTokenOnForbidden(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"new-token"}`))
		case "/api/collections/products/records":
			requests++
			if requests == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if got := r.Header.Get("Authorization"); got != "new-token" {
				http.Error(w, fmt.Sprintf("unexpected token: %s", got), http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := New(server.URL, "admin@example.com", "secret", server.Client())
	if err := c.Authenticate(context.Background()); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Items []any `json:"items"`
	}
	if err := c.List(context.Background(), "products", nil, &response); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("expected two requests, got %d", requests)
	}
}
