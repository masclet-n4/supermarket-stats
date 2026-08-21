package pocketbase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/collections/_superusers/auth-with-password" { t.Fatalf("unexpected path: %s", r.URL.Path) }
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"jwt"}`))
	}))
	defer server.Close()
	c := New(server.URL, "admin@example.com", "secret", server.Client())
	if err := c.Authenticate(context.Background()); err != nil { t.Fatal(err) }
	if c.token != "jwt" { t.Fatal("unexpected token") }
}
