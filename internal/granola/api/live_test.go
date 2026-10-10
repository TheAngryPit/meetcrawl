package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveClientAuthFailsClosed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	client := &LiveClient{
		http:    srv.Client(),
		token:   "synthetic-token",
		baseURL: srv.URL,
	}
	_, err := client.listNotes(context.Background())
	var auth *AuthError
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !AsAuthError(err, &auth) {
		t.Fatalf("err = %T %v, want AuthError", err, err)
	}
}
