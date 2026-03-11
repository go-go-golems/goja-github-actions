package githubmodule

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestGitHubModuleProvidesContextAndAPIClient(t *testing.T) {
	t.Parallel()

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/actions/permissions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"enabled":              true,
				"allowed_actions":      "selected",
				"github_owned_allowed": true,
			})
		case "/repos/acme/widgets/actions/permissions/selected-actions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"patterns_allowed": []string{"acme/*"},
			})
		case "/repos/acme/widgets/actions/permissions/workflow":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"default_workflow_permissions":     "read",
				"can_approve_pull_request_reviews": false,
			})
		case "/repos/acme/widgets/actions/workflows":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"workflows": []map[string]interface{}{
					{"id": 1001, "name": "CI"},
				},
			})
		case "/items":
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				w.Header().Set("Link", `<`+serverURL+`/items?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": 1},
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 2},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	eventPath := filepath.Join(tempDir, "event.json")

	script := `
const github = require("@actions/github");

module.exports = function () {
  const octokit = github.getOctokit();
  const permissions = octokit.rest.actions.getGithubActionsPermissionsRepository({ owner: "acme", repo: "widgets" });
  const selected = octokit.rest.actions.getAllowedActionsRepository({ owner: "acme", repo: "widgets" });
  const workflowPermissions = octokit.rest.actions.getWorkflowPermissionsRepository({ owner: "acme", repo: "widgets" });
  const workflows = octokit.rest.actions.listRepoWorkflows({ owner: "acme", repo: "widgets" });
  const pages = octokit.paginate("GET /items", { page: 1 });

  return {
    actor: github.context.actor,
    repoOwner: github.context.repo.owner,
    repoName: github.context.repo.repo,
    payloadAction: github.context.payload.action,
    permissions: permissions.data,
    selected: selected.data,
    workflowPermissions: workflowPermissions.data,
    workflowCount: workflows.data.total_count,
    pageIds: pages.map((page) => page.id)
  };
};
`
	if err := os.WriteFile(entryPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}
	if err := os.WriteFile(eventPath, []byte(`{"action":"audit","repository":{"full_name":"acme/widgets","owner":{"login":"acme"},"name":"widgets"}}`), 0o644); err != nil {
		t.Fatalf("write event.json: %v", err)
	}

	settings := &gharuntime.Settings{
		ScriptPath:       entryPath,
		WorkingDirectory: tempDir,
		Workspace:        tempDir,
		EventPath:        eventPath,
		GitHubToken:      "secret-token",
		AmbientEnvironment: map[string]string{
			"GITHUB_ACTOR":      "manuel",
			"GITHUB_EVENT_NAME": "workflow_dispatch",
			"GITHUB_REPOSITORY": "acme/widgets",
			"GITHUB_API_URL":    server.URL,
		},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings}),
	)
	if err != nil {
		t.Fatalf("run script with github module: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result.Export())
	}

	if got, want := exported["actor"], "manuel"; got != want {
		t.Fatalf("actor = %v, want %v", got, want)
	}
	if got, want := exported["repoOwner"], "acme"; got != want {
		t.Fatalf("repoOwner = %v, want %v", got, want)
	}
	if got, want := exported["repoName"], "widgets"; got != want {
		t.Fatalf("repoName = %v, want %v", got, want)
	}
	if got, want := exported["payloadAction"], "audit"; got != want {
		t.Fatalf("payloadAction = %v, want %v", got, want)
	}
	if got, want := exported["workflowCount"], int64(1); got != want {
		t.Fatalf("workflowCount = %v, want %v", got, want)
	}

	pageIDs, ok := exported["pageIds"].([]interface{})
	if !ok || len(pageIDs) != 2 {
		t.Fatalf("pageIds = %#v, want 2 entries", exported["pageIds"])
	}
}
