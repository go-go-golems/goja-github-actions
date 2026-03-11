package githubapi

import (
	"context"
	"net/http"
	"testing"
)

func TestBuildRequestInterpolatesRouteAndQuery(t *testing.T) {
	t.Parallel()

	request, err := BuildRequest(
		context.Background(),
		"https://api.example.test",
		"GET /repos/{owner}/{repo}/actions/workflows",
		map[string]interface{}{
			"owner":    "acme",
			"repo":     "widgets",
			"per_page": 50,
		},
	)
	if err != nil {
		t.Fatalf("BuildRequest() error = %v", err)
	}

	if got, want := request.Method, http.MethodGet; got != want {
		t.Fatalf("method = %q, want %q", got, want)
	}
	if got, want := request.URL.String(), "https://api.example.test/repos/acme/widgets/actions/workflows?per_page=50"; got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestBuildRequestUsesAbsoluteRouteURL(t *testing.T) {
	t.Parallel()

	request, err := BuildRequest(
		context.Background(),
		"https://api.example.test",
		"GET https://api.other.test/repos/acme/widgets/actions/workflows?page=2",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("BuildRequest() error = %v", err)
	}

	if got, want := request.URL.String(), "https://api.other.test/repos/acme/widgets/actions/workflows?page=2"; got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}
