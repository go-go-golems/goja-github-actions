package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDoRouteSetsHeadersAndDecodesJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer secret-token"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Accept"), "application/vnd.github+json"; got != want {
			t.Fatalf("accept = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/repos/acme/widgets/actions/permissions"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled": true,
		})
	}))
	defer server.Close()

	client := NewClient("secret-token", server.URL)
	result, err := client.DoRoute(context.Background(), "GET /repos/{owner}/{repo}/actions/permissions", map[string]interface{}{
		"owner": "acme",
		"repo":  "widgets",
	})
	if err != nil {
		t.Fatalf("DoRoute() error = %v", err)
	}

	payload, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", result.Data)
	}
	if got, want := payload["enabled"], true; got != want {
		t.Fatalf("enabled = %v, want %v", got, want)
	}
}

func TestClientDoRouteReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "bad credentials",
		})
	}))
	defer server.Close()

	client := NewClient("bad-token", server.URL)
	_, err := client.DoRoute(context.Background(), "GET /user", map[string]interface{}{})
	if err == nil {
		t.Fatal("DoRoute() error = nil, want APIError")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if got, want := apiErr.Status, http.StatusUnauthorized; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := apiErr.Message, "bad credentials"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}
