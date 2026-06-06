package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerMiddlewareAllowsValidToken(t *testing.T) {
	called := false
	handler := BearerMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/routes", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestBearerMiddlewareRejectsInvalidToken(t *testing.T) {
	handler := BearerMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/routes", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}
