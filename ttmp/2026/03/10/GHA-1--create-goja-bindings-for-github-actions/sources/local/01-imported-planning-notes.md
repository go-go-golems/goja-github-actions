---
Title: Imported planning notes
Ticket: GHA-1
Status: active
Topics:
    - goja
    - github-actions
    - javascript
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/goja-gha-plan.md
Summary: Imported user-provided planning notes covering GitHub Actions policy APIs and possible implementation directions.
LastUpdated: 2026-03-10T21:28:42.132847306-04:00
WhatFor: Preserve the imported source material that seeded the goja-gha design work.
WhenToUse: Use when tracing the original planning note that informed the first concrete use case and API references.
---

To REDUCE (not eliminate) the risk that one of your packages is the next to be compromised via Github Actions, I recommend you do at least the following. You have to do this for every Github organizations AND each and every repository you control. For organizations (but not user accounts) some of these settings can be changed for all repositories in that account, but NOT ALL! So you still have to check each and every repository that belongs to an organization. Going forward whenever you create a new repository, check these settings. The defaults are unsafe. Github may also add new features and choose unsafe defaults for those. Go to https://github.com/organizations/ORGANIZATION/settings/actions resp. https://github.com/USER/REPO/settings/actions . Note that there are several Save buttons, which have to be clicked individually. Do NOT change all settings and then only click the final Save button. Double check the final result before moving on to the next organization/repo. Action permissions Select Disable actions if you don't actually use actions CLick Save You are not done. Also make the below changes, to avoid surprises should you later enable actions for this repository/organization or For organizations check the value of Selected repositories and limit to repositories that you are certain have a legitimate use and currently actually use actions. Select Allow X, and select non-X, actions and reusable workflows Uncheck Allow actions created by GitHub. This likely will break some workflows; you can later add the required actions to the below list. Uncheck Allow actions by Marketplace verified creators Explicitly populate Allow or block specified actions and reusable workflows CLick Save Approval for running fork pull request workflows from contributors Select Require approval for all external contributors CLick Save Going forward ALWAYS inspect the changes BEFORE allowing the actions to run for a pull-request Workflow permissions Select Read repository contents and packages permissions Uncheck Allow GitHub Actions to create and approve pull requests The attack used against kubernetes-el/kubernetes-el exfiltrated a token (which had too many unneeded permissions) from memory. I don't think the above protects against that (as I understand it, only not EVER running any untrusted code does), but at least it protects against less sophisticated attacks. If you use actions/checkout without persist-credentials: false then a much less sophisticated attack can be used to extract the token (i.e., read from well-known file). So make sure you always use persist-credentials: false. permalink All from: https://old.reddit.com/r/emacs/comments/1rowm5i/first_hacked_emacs_package/ Is this something I can all do from the CLI?

## Imported note segment 1
Yes—almost all of that can be done from the CLI, and the supported path is `gh api` against GitHub’s REST API. GitHub has API endpoints for org- and repo-level Actions permissions, selected-action allowlists, public-fork approval policy, private-fork workflow settings, and default `GITHUB_TOKEN` workflow permissions, and GitHub’s CLI docs explicitly position `gh api` as the command-line way to call those endpoints. ([GitHub CLI][1])

The only big exception in your Reddit checklist is `persist-credentials: false`: that is **not** an account/repo setting. It lives inside workflow YAML, so you have to audit and edit `.github/workflows/*.yml` in each repo. Also, that specific Reddit detail is a bit version-sensitive now: current `actions/checkout` docs say newer releases persist credentials in a separate file under `$RUNNER_TEMP` rather than directly in `.git/config`, but `persist-credentials: false` is still the opt-out knob. ([GitHub][2])

Here’s the rough mapping from UI to CLI/API:

* **Action permissions**: org/repo enable-disable Actions, choose `all` / `local_only` / `selected`, and optionally require full SHA pinning. ([GitHub Docs][3])
* **Allow/block specific actions and reusable workflows**: org/repo selected-actions endpoint with `github_owned_allowed`, `verified_allowed`, and `patterns_allowed`. ([GitHub Docs][3])
* **Approval for running fork pull request workflows from contributors**: org/repo fork-PR contributor approval endpoint; `all_external_contributors` matches the strictest option from the UI. ([GitHub Docs][3])
* **Workflow permissions**: org/repo endpoint for `default_workflow_permissions=read` and `can_approve_pull_request_reviews=false`. ([GitHub Docs][3])
* **Org “selected repositories”**: org endpoint to set `enabled_repositories=selected`, plus endpoints to list/set/enable/disable the selected repo IDs. ([GitHub Docs][3])
* **Private repo fork workflow options**: repo/org endpoint for `run_workflows_from_fork_pull_requests`, `send_write_tokens_to_workflows`, `send_secrets_and_variables`, and `require_approval_for_fork_pr_workflows`. ([GitHub Docs][3])

A basic repo-level example with `gh` looks like this:

```bash
# authenticate once
gh auth login

OWNER=your-owner
REPO=your-repo

# 1) Actions permissions: enabled, but only selected actions/workflows allowed
gh api -X PUT "repos/$OWNER/$REPO/actions/permissions" \
  -F enabled=true \
  -f allowed_actions=selected \
  -F sha_pinning_required=true

# 2) Explicit allowlist: no blanket GitHub-owned actions, no blanket verified creators
#    Add only the exact actions/workflows you want
gh api -X PUT "repos/$OWNER/$REPO/actions/permissions/selected-actions" \
  -F github_owned_allowed=false \
  -F verified_allowed=false \
  -F 'patterns_allowed[]=your-org/*' \
  -F 'patterns_allowed[]=your-org/your-reusable-workflows/.github/workflows/build.yml@*'

# 3) Public-fork approval policy: require approval for all outside contributors
gh api -X PUT "repos/$OWNER/$REPO/actions/permissions/fork-pr-contributor-approval" \
  -f approval_policy=all_external_contributors

# 4) Default workflow token perms: read only; Actions may not approve PRs
gh api -X PUT "repos/$OWNER/$REPO/actions/permissions/workflow" \
  -f default_workflow_permissions=read \
  -F can_approve_pull_request_reviews=false

# 5) Private-repo fork PR workflow policy (private repos only)
gh api -X PUT "repos/$OWNER/$REPO/actions/permissions/fork-pr-workflows-private-repos" \
  -F run_workflows_from_fork_pull_requests=true \
  -F send_write_tokens_to_workflows=false \
  -F send_secrets_and_variables=false \
  -F require_approval_for_fork_pr_workflows=true

# 6) Verify current state
gh api "repos/$OWNER/$REPO/actions/permissions"
gh api "repos/$OWNER/$REPO/actions/permissions/selected-actions"
gh api "repos/$OWNER/$REPO/actions/permissions/fork-pr-contributor-approval"
gh api "repos/$OWNER/$REPO/actions/permissions/workflow"
```

For an organization, the same pattern works with `orgs/$ORG/...`. If you want “selected repositories” at the org level, you first set `enabled_repositories=selected`, then populate the selected repo IDs. `gh repo list` can give you repo IDs and names for batching. ([GitHub CLI][4])

```bash
ORG=your-org

# Org-wide policy
gh api -X PUT "orgs/$ORG/actions/permissions" \
  -f enabled_repositories=selected \
  -f allowed_actions=selected \
  -F sha_pinning_required=true

# Example: set the exact org repos that may use Actions
gh api -X PUT "orgs/$ORG/actions/permissions/repositories" \
  -F 'selected_repository_ids[]=123456789' \
  -F 'selected_repository_ids[]=987654321'

# Org-wide public-fork approval policy
gh api -X PUT "orgs/$ORG/actions/permissions/fork-pr-contributor-approval" \
  -f approval_policy=all_external_contributors

# Org-wide workflow token defaults
gh api -X PUT "orgs/$ORG/actions/permissions/workflow" \
  -f default_workflow_permissions=read \
  -F can_approve_pull_request_reviews=false
```

Two important caveats. First, the fork-approval policy does **not** stop `pull_request_target` workflows; GitHub documents that those always run in the base-branch context regardless of the approval setting, so that setting is helpful but not sufficient. Second, “workflow permissions” are only the default: people with write access can still change permissions in the workflow file itself, and repo-level changes can be blocked by stricter org settings. ([GitHub Docs][5])

Also, the “defaults are unsafe” claim is too broad in 2026. GitHub’s docs currently say new personal-account repos default to read-only `contents` and `packages`, new orgs default to that same restricted token setting, and PR creation/approval by Actions is disabled by default for new personal repos and new orgs. Older orgs or inherited settings can still be looser, so auditing is still smart. ([GitHub Docs][5])

If you want, I can turn this into a single idempotent bash script that audits every repo in one or more orgs, prints the current state, and optionally applies the hardened settings.

[1]: https://cli.github.com/manual/gh_api?utm_source=chatgpt.com "GitHub CLI | Take GitHub to the command line"
[2]: https://github.com/actions/checkout?utm_source=chatgpt.com "actions/checkout: Action for checking out a repo"
[3]: https://docs.github.com/en/rest/actions/permissions "REST API endpoints for GitHub Actions permissions - GitHub Docs"
[4]: https://cli.github.com/manual/gh_repo_list "GitHub CLI | Take GitHub to the command line"
[5]: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository "Managing GitHub Actions settings for a repository - GitHub Docs"


## Imported note segment 2

Yes. For this, I’d build a real policy tool around GitHub’s REST API rather than drive the web UI or shell out to `gh` for everything. The settings you care about live in the REST Actions permissions endpoints, while the workflow-file checks live in the repo contents/tree APIs. For authentication, a GitHub App is the cleanest long-term option for many repos/orgs: apps start with no permissions, you grant only the minimum, and installation access tokens expire after one hour and can be limited to specific repositories. If you just want a quick first version, a fine-grained PAT also works. ([GitHub Docs][1])

In Go, `go-github` is a good fit for client/auth plumbing, but you do not need to wait for typed wrappers for every endpoint. The current upstream README shows `github.com/google/go-github/v84/github`, `WithAuthToken`, and GitHub App auth patterns; the client also exposes `NewRequest`, which builds JSON requests and sets the API version header, so you can call the exact REST paths directly. ([GitHub][2])

The official pages that matter most are these:

* **Actions permissions**: this is the main one. It covers org and repo endpoints for enable/disable Actions, `allowed_actions`, `sha_pinning_required`, selected repositories for orgs, selected action allowlists, fork PR approval policy, private-repo fork workflow settings, and default `GITHUB_TOKEN` workflow permissions. ([GitHub Docs][3])
* **Workflows API**: use this to list workflow objects, check enabled/disabled state, and optionally disable/enable specific workflows by ID or filename. ([GitHub Docs][4])
* **Repository contents API**: use this to read and update `.github/workflows/*.yml`; it supports raw file fetches via `Accept: application/vnd.github.raw+json`. GitHub’s docs also note that modifying files under `.github/workflows` has extra auth considerations. ([GitHub Docs][5])
* **Git Trees API**: useful if you want a generic scanner that walks a repo recursively; watch the documented truncation limits on large trees. ([GitHub Docs][6])
* **Repository listing**: `GET /user/repos` for everything you can access, and `GET /orgs/{org}/repos` for org inventory. ([GitHub Docs][7])

The exact REST paths I’d model in your tool are:

```text
# inventory
GET /user/repos
GET /orgs/{org}/repos
GET /repos/{owner}/{repo}

# repo-level Actions settings
GET /repos/{owner}/{repo}/actions/permissions
PUT /repos/{owner}/{repo}/actions/permissions

GET /repos/{owner}/{repo}/actions/permissions/selected-actions
PUT /repos/{owner}/{repo}/actions/permissions/selected-actions

GET /repos/{owner}/{repo}/actions/permissions/fork-pr-contributor-approval
PUT /repos/{owner}/{repo}/actions/permissions/fork-pr-contributor-approval

GET /repos/{owner}/{repo}/actions/permissions/fork-pr-workflows-private-repos
PUT /repos/{owner}/{repo}/actions/permissions/fork-pr-workflows-private-repos

GET /repos/{owner}/{repo}/actions/permissions/workflow
PUT /repos/{owner}/{repo}/actions/permissions/workflow

# org-level Actions settings
GET /orgs/{org}/actions/permissions
PUT /orgs/{org}/actions/permissions

GET /orgs/{org}/actions/permissions/repositories
PUT /orgs/{org}/actions/permissions/repositories/{repository_id}
DELETE /orgs/{org}/actions/permissions/repositories/{repository_id}

GET /orgs/{org}/actions/permissions/selected-actions
PUT /orgs/{org}/actions/permissions/selected-actions

GET /orgs/{org}/actions/permissions/fork-pr-contributor-approval
PUT /orgs/{org}/actions/permissions/fork-pr-contributor-approval

GET /orgs/{org}/actions/permissions/fork-pr-workflows-private-repos
PUT /orgs/{org}/actions/permissions/fork-pr-workflows-private-repos

GET /orgs/{org}/actions/permissions/workflow
PUT /orgs/{org}/actions/permissions/workflow

# workflow metadata
GET /repos/{owner}/{repo}/actions/workflows
GET /repos/{owner}/{repo}/actions/workflows/{workflow_id}
PUT /repos/{owner}/{repo}/actions/workflows/{workflow_id}/disable
PUT /repos/{owner}/{repo}/actions/workflows/{workflow_id}/enable

# workflow file content
GET /repos/{owner}/{repo}/contents/.github/workflows?ref={branch}
GET /repos/{owner}/{repo}/contents/.github/workflows/{file}?ref={branch}
PUT /repos/{owner}/{repo}/contents/.github/workflows/{file}
```

Those repo/org Actions endpoints are all documented on the single Actions permissions page, including the allowed enum values like `all`, `local_only`, `selected`, `read`, `write`, and the contributor approval policies. GitHub also documents that `patterns_allowed` uses string-matching patterns and only applies to public repositories, and that selected-action endpoints only make sense when `allowed_actions=selected`. ([GitHub Docs][3])

A good validator should treat security as two layers:

1. **Repo/org settings** from the Actions permissions API.
2. **Workflow YAML semantics** from `.github/workflows`.

That distinction matters because GitHub calculates `GITHUB_TOKEN` permissions from the enterprise/org/repo default first, then adjusts them using workflow-level and job-level `permissions`, and for fork PRs may downshift writes to reads unless the “send write tokens” setting is enabled. Also, if you specify any individual permission in workflow YAML, unspecified permissions become `none`. ([GitHub Docs][8])

So the checks I would implement are:

* repo/org `allowed_actions` should not be `all` unless explicitly intended
* `sha_pinning_required` should be `true`
* default workflow permissions should be `read`
* `can_approve_pull_request_reviews` should be `false`
* fork PR contributor approval should usually be `all_external_contributors`
* selected-action allowlists should be explicit if you choose `selected`
* workflow files should avoid broad `permissions`, especially `write-all`
* flag `pull_request_target` and `workflow_run` for manual review, because GitHub explicitly warns that running untrusted code on those triggers can lead to cache poisoning or unintended write/secrets exposure
* flag any `uses:` references that are not pinned to a full commit SHA
* flag `actions/checkout` steps that do not set `persist-credentials: false` if that is your org policy. That one is a workflow lint rule, not an account setting. Current `checkout` docs say v6 stores credentials in a separate file under `$RUNNER_TEMP`, and `persist-credentials: false` is still the opt-out knob. ([GitHub Docs][9])

Here is a straightforward Go skeleton using plain `net/http` for the exact REST paths:

```go
package ghsec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiVersion = "2022-11-28"

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		BaseURL: "https://api.github.com",
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, accept string, out any) error {
	u := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, r)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: status=%d body=%s", method, u, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	switch v := out.(type) {
	case *[]byte:
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		*v = b
		return nil
	default:
		return json.NewDecoder(resp.Body).Decode(out)
	}
}

type Repo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	Disabled      bool   `json:"disabled"`
}

type RepoActionsPermissions struct {
	Enabled            bool   `json:"enabled"`
	AllowedActions     string `json:"allowed_actions"`
	SelectedActionsURL string `json:"selected_actions_url"`
	SHAPinningRequired bool   `json:"sha_pinning_required"`
}

type SelectedActions struct {
	GitHubOwnedAllowed bool     `json:"github_owned_allowed"`
	VerifiedAllowed    bool     `json:"verified_allowed"`
	PatternsAllowed    []string `json:"patterns_allowed"`
}

type WorkflowPermissions struct {
	DefaultWorkflowPermissions   string `json:"default_workflow_permissions"`
	CanApprovePullRequestReviews bool   `json:"can_approve_pull_request_reviews"`
}

type ForkPRContributorApproval struct {
	ApprovalPolicy string `json:"approval_policy"`
}

type PrivateForkPRWorkflows struct {
	RunWorkflowsFromForkPullRequests bool `json:"run_workflows_from_fork_pull_requests"`
	SendWriteTokensToWorkflows       bool `json:"send_write_tokens_to_workflows"`
	SendSecretsAndVariables          bool `json:"send_secrets_and_variables"`
	RequireApprovalForForkPRWorkflows bool `json:"require_approval_for_fork_pr_workflows"`
}

type WorkflowList struct {
	TotalCount int        `json:"total_count"`
	Workflows  []Workflow `json:"workflows"`
}

type Workflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

type ContentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

func (c *Client) ListMyRepos(ctx context.Context, page int) ([]Repo, error) {
	var repos []Repo
	q := url.Values{}
	q.Set("per_page", "100")
	q.Set("page", fmt.Sprintf("%d", page))
	err := c.do(ctx, http.MethodGet, "user/repos", q, nil, "application/vnd.github+json", &repos)
	return repos, err
}

func (c *Client) ListOrgRepos(ctx context.Context, org string, page int) ([]Repo, error) {
	var repos []Repo
	q := url.Values{}
	q.Set("per_page", "100")
	q.Set("page", fmt.Sprintf("%d", page))
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("orgs/%s/repos", org), q, nil, "application/vnd.github+json", &repos)
	return repos, err
}

func (c *Client) GetRepoActionsPermissions(ctx context.Context, owner, repo string) (*RepoActionsPermissions, error) {
	var out RepoActionsPermissions
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/actions/permissions", owner, repo),
		nil, nil, "application/vnd.github+json", &out)
	return &out, err
}

func (c *Client) SetRepoActionsPermissions(ctx context.Context, owner, repo string, in RepoActionsPermissions) error {
	body := map[string]any{
		"enabled":              in.Enabled,
		"allowed_actions":      in.AllowedActions,
		"sha_pinning_required": in.SHAPinningRequired,
	}
	return c.do(ctx, http.MethodPut,
		fmt.Sprintf("repos/%s/%s/actions/permissions", owner, repo),
		nil, body, "application/vnd.github+json", nil)
}

func (c *Client) GetRepoSelectedActions(ctx context.Context, owner, repo string) (*SelectedActions, error) {
	var out SelectedActions
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/actions/permissions/selected-actions", owner, repo),
		nil, nil, "application/vnd.github+json", &out)
	return &out, err
}

func (c *Client) SetRepoSelectedActions(ctx context.Context, owner, repo string, in SelectedActions) error {
	return c.do(ctx, http.MethodPut,
		fmt.Sprintf("repos/%s/%s/actions/permissions/selected-actions", owner, repo),
		nil, in, "application/vnd.github+json", nil)
}

func (c *Client) GetRepoWorkflowPermissions(ctx context.Context, owner, repo string) (*WorkflowPermissions, error) {
	var out WorkflowPermissions
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/actions/permissions/workflow", owner, repo),
		nil, nil, "application/vnd.github+json", &out)
	return &out, err
}

func (c *Client) SetRepoWorkflowPermissions(ctx context.Context, owner, repo string, in WorkflowPermissions) error {
	return c.do(ctx, http.MethodPut,
		fmt.Sprintf("repos/%s/%s/actions/permissions/workflow", owner, repo),
		nil, in, "application/vnd.github+json", nil)
}

func (c *Client) ListWorkflows(ctx context.Context, owner, repo string) (*WorkflowList, error) {
	var out WorkflowList
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/actions/workflows", owner, repo),
		nil, nil, "application/vnd.github+json", &out)
	return &out, err
}

func (c *Client) ListWorkflowFiles(ctx context.Context, owner, repo, ref string) ([]ContentEntry, error) {
	var out []ContentEntry
	q := url.Values{}
	q.Set("ref", ref)
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/contents/.github/workflows", owner, repo),
		q, nil, "application/vnd.github+json", &out)
	return out, err
}

func (c *Client) GetWorkflowFileRaw(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	var out []byte
	q := url.Values{}
	q.Set("ref", ref)
	err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, path),
		q, nil, "application/vnd.github.raw+json", &out)
	return out, err
}
```

With that client, an audit loop is pretty simple:

```go
func AuditRepo(ctx context.Context, gh *Client, owner, repo, defaultBranch string) ([]string, error) {
	var findings []string

	ap, err := gh.GetRepoActionsPermissions(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if ap.AllowedActions == "all" {
		findings = append(findings, "allowed_actions=all")
	}
	if !ap.SHAPinningRequired {
		findings = append(findings, "sha_pinning_required=false")
	}

	wp, err := gh.GetRepoWorkflowPermissions(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if wp.DefaultWorkflowPermissions != "read" {
		findings = append(findings, "default_workflow_permissions != read")
	}
	if wp.CanApprovePullRequestReviews {
		findings = append(findings, "can_approve_pull_request_reviews=true")
	}

	files, err := gh.ListWorkflowFiles(ctx, owner, repo, defaultBranch)
	if err != nil {
		// no workflows directory is not necessarily an error in your policy
		return findings, nil
	}

	for _, f := range files {
		if f.Type != "file" || !(strings.HasSuffix(f.Name, ".yml") || strings.HasSuffix(f.Name, ".yaml")) {
			continue
		}
		raw, err := gh.GetWorkflowFileRaw(ctx, owner, repo, f.Path, defaultBranch)
		if err != nil {
			findings = append(findings, "cannot read "+f.Path+": "+err.Error())
			continue
		}
		s := string(raw)

		if strings.Contains(s, "pull_request_target:") {
			findings = append(findings, f.Path+": uses pull_request_target")
		}
		if strings.Contains(s, "workflow_run:") {
			findings = append(findings, f.Path+": uses workflow_run")
		}
		if strings.Contains(s, "permissions: write-all") {
			findings = append(findings, f.Path+": permissions write-all")
		}
		if strings.Contains(s, "uses: actions/checkout") && !strings.Contains(s, "persist-credentials: false") {
			findings = append(findings, f.Path+": checkout without persist-credentials:false")
		}
	}

	return findings, nil
}
```

For a production version, I’d make three subcommands:

* `audit`: read-only, emits JSON and a human report
* `fix-settings`: applies org/repo Actions settings through the permissions endpoints
* `fix-workflows`: opens `.github/workflows/*`, rewrites obvious cases, and commits changes

A few design notes matter:

* For **org-level selected repositories**, you need repository IDs, not names, so collect those from repo inventory first. ([GitHub Docs][3])
* For **selected action allowlists**, GitHub documents the `patterns_allowed` syntax and notes that it applies only to public repositories, so your validator should not blindly treat that field the same way on private repos. ([GitHub Docs][3])
* For **editing workflow files**, the Contents API is easiest for one-file-at-a-time updates. If you later want batched, single-commit multi-file changes across a repo, move to the Git Data APIs (`trees`, `commits`, `refs`). ([GitHub Docs][5])
* If you’re on **GitHub Enterprise Server**, `go-github` has `WithEnterpriseURLs`, and its docs note the `/api/v3/` URL shape. ([GitHub][10])

The one thing I would not do is make this a pure “settings mirror” tool. The dangerous cases are often in the YAML itself, and GitHub’s own docs explicitly warn about `pull_request_target`, `workflow_run`, broad permissions, and mutable action refs. So the sweet spot is: **inventory + settings audit + YAML lint + optional auto-fix**. ([GitHub Docs][9])

[2]: https://github.com/google/go-github "https://github.com/google/go-github"
[3]: https://docs.github.com/en/rest/actions/permissions "https://docs.github.com/en/rest/actions/permissions"
[4]: https://docs.github.com/rest/actions/workflows "https://docs.github.com/rest/actions/workflows"
[5]: https://docs.github.com/rest/repos/contents "https://docs.github.com/rest/repos/contents"
[6]: https://docs.github.com/en/rest/git/trees "https://docs.github.com/en/rest/git/trees"
[7]: https://docs.github.com/en/rest/repos/repos "https://docs.github.com/en/rest/repos/repos"
[8]: https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions "https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions"
[9]: https://docs.github.com/en/actions/reference/security/secure-use "https://docs.github.com/en/actions/reference/security/secure-use"
[10]: https://github.com/google/go-github/blob/master/github/github.go "https://github.com/google/go-github/blob/master/github/github.go"


## Imported note segment 3

I would make Go the **collector / normalizer / executor**, and Goja the **policy engine**. In other words: Go talks to GitHub, fetches repo settings plus workflow files, parses YAML, computes the security-relevant semantics, and then hands a clean immutable snapshot into small JS validators. That keeps the JS side pleasant and expressive, while the hard parts — auth, rate limits, YAML edge cases, API pagination, caching, and safety limits — stay in Go. That split also fits Goja well, because Goja does **not** provide browser/Node-style timers by default; its host application is expected to provide that environment, and it supports interruption from Go, so a synchronous, bounded validator model is the cleanest design. ([GitHub][1])

The main thing to optimize for is this: your validators should not need to “understand GitHub Actions from scratch.” They should receive a **semantic model** that already answers the annoying questions you actually care about, such as:

* what events can trigger this workflow?
* is this a privileged workflow?
* what are the effective `GITHUB_TOKEN` permissions after repo defaults, workflow-level overrides, job-level overrides, and fork downgrades?
* does this workflow check out untrusted code in a privileged context?
* are any third-party actions unpinned?
* is `actions/checkout` used without `persist-credentials: false`?
* does a job bridge trust boundaries through artifacts, caches, or reusable workflows?

That semantic layer matters because GitHub’s model is subtle. GitHub documents that workflow/job `permissions` override the defaults; if you specify any individual permission, the unspecified ones become `none`; and fork-triggered runs may have write permissions reduced to read unless the relevant setting allows write tokens. GitHub also documents that `pull_request_target` gets read/write repository permission even when triggered from a public fork, and its secure-use guidance warns against using `pull_request_target` or `workflow_run` with untrusted code and recommends pinning third-party actions to a full commit SHA. ([GitHub Docs][2])

## The mental model

I’d structure it like this:

**Layer 1: Raw collectors in Go**

* list orgs/repos
* fetch Actions settings from the REST API
* fetch workflow files from `.github/workflows`
* optionally fetch action metadata, branch protections, rulesets, environments, reusable workflow references, etc.

**Layer 2: Normalized domain model in Go**

* parse YAML into a stable AST
* normalize `on:` syntax into a canonical trigger list
* normalize jobs, steps, `uses`, `run`, `permissions`, `env`, matrices
* compute semantic facts like “job is privileged,” “job checks out PR head,” “uses mutable ref,” “crosses trust boundary”

**Layer 3: JS validator runtime**

* expose a frozen `ctx`
* expose helper libraries like `gha`, `trust`, `perms`, `report`
* run validator scripts with time and memory limits
* collect findings

**Layer 4: Fix / suggest / gate**

* report findings
* optionally generate patches
* optionally auto-fix settings
* optionally fail CI or open PRs

## What the JS API should feel like

Not “here is raw YAML, good luck.”

More like this:

```js
export const meta = {
  id: "checkout-persist-creds",
  title: "actions/checkout should opt out of persisted credentials",
  scope: ["workflow"],
  severity: "high",
};

export function validate(ctx, api) {
  for (const wf of ctx.workflows) {
    for (const job of wf.jobs) {
      for (const step of job.steps) {
        if (api.gha.isCheckout(step) && !api.gha.checkoutHasPersistCredentialsFalse(step)) {
          api.report.finding({
            ruleId: meta.id,
            severity: "high",
            message: "actions/checkout is used without persist-credentials: false",
            location: step.location,
            remediation: {
              kind: "yaml-edit",
              suggestion: "Add `with.persist-credentials: false` to this checkout step.",
            },
            evidence: {
              workflow: wf.path,
              job: job.id,
              step: step.name || step.uses,
            },
          });
        }
      }
    }
  }
}
```

And for something more semantic:

```js
export const meta = {
  id: "privileged-untrusted-checkout",
  title: "Privileged workflow checks out untrusted code",
  severity: "critical",
};

export function validate(ctx, api) {
  for (const wf of ctx.workflows) {
    if (!wf.trust.isPrivilegedTrigger) continue;

    for (const job of wf.jobs) {
      if (job.trust.checksOutUntrustedRef) {
        api.report.finding({
          ruleId: meta.id,
          severity: "critical",
          message: "Privileged workflow appears to check out untrusted code.",
          location: job.location,
          help: [
            "Avoid checking out PR head or fork code in pull_request_target/workflow_run contexts.",
            "Split privileged and unprivileged work into separate workflows.",
          ],
          evidence: {
            trigger: wf.on.normalized,
            job: job.id,
            checkoutRefs: job.trust.checkedOutRefs,
          },
        });
      }
    }
  }
}
```

That second one is the kind of rule that becomes pleasant only if Go has already done the hard inference.

## What `ctx` should contain

I would make `ctx` a read-only snapshot with four main sections.

### 1. Repository / organization settings

```ts
type RepoContext = {
  repo: {
    owner: string;
    name: string;
    private: boolean;
    defaultBranch: string;
    archived: boolean;
  };

  org?: {
    login: string;
  };

  actionsSettings: {
    enabled: boolean;
    allowedActions: "all" | "local_only" | "selected";
    shaPinningRequired: boolean;
    selectedActions?: {
      githubOwnedAllowed: boolean;
      verifiedAllowed: boolean;
      patternsAllowed: string[];
    };
    forkPrApproval?: {
      approvalPolicy: string;
    };
    workflowPermissions: {
      defaultWorkflowPermissions: "read" | "write";
      canApprovePullRequestReviews: boolean;
    };
    privateForkPolicy?: {
      runWorkflowsFromForkPullRequests: boolean;
      sendWriteTokensToWorkflows: boolean;
      sendSecretsAndVariables: boolean;
      requireApprovalForForkPrWorkflows: boolean;
    };
  };

  workflows: WorkflowContext[];
};
```

That directly maps to the settings you were discussing earlier.

### 2. Workflow AST + normalized semantics

```ts
type WorkflowContext = {
  id: string;
  path: string;
  name?: string;

  raw: {
    yaml: string;
  };

  on: {
    normalized: Trigger[];
    has(eventName: string): boolean;
  };

  permissions: {
    workflow?: PermissionMap | "read-all" | "write-all" | "{}";
  };

  trust: {
    isPrivilegedTrigger: boolean;
    usesPullRequestTarget: boolean;
    usesWorkflowRun: boolean;
    mayReceiveUntrustedInput: boolean;
  };

  jobs: JobContext[];

  findingsCache?: {
    usesMutableActions: boolean;
    usesUnpinnedThirdPartyActions: boolean;
  };
};
```

### 3. Job-level effective semantics

This is where a lot of the value lives.

```ts
type JobContext = {
  id: string;
  name?: string;
  needs: string[];
  runsOn: string[];
  environment?: string;

  permissions: {
    declared?: PermissionMap | "read-all" | "write-all" | "{}";
    effective: EffectivePermissionMap;
    source: {
      repoDefault: string;
      workflowOverride?: boolean;
      jobOverride?: boolean;
      downgradedForFork?: boolean;
    };
  };

  trust: {
    checksOutUntrustedRef: boolean;
    checkedOutRefs: string[];
    receivesArtifactsFromUntrustedJobs: boolean;
    publishesArtifacts: boolean;
    consumesArtifacts: boolean;
    usesCaches: boolean;
    crossesTrustBoundary: boolean;
  };

  steps: StepContext[];
  location: SourceLocation;
};
```

This is exactly where you encode GitHub’s documented permission-calculation rules, instead of forcing every script author to rediscover them. GitHub’s docs say defaults come from enterprise/org/repo, then workflow-level and job-level `permissions` adjust them, and forked PR runs can have writes reduced to reads unless the repo setting sends write tokens. ([GitHub Docs][2])

### 4. Step-level helpers

```ts
type StepContext = {
  kind: "uses" | "run";
  id?: string;
  name?: string;

  uses?: {
    owner?: string;
    repo?: string;
    subpath?: string;
    ref?: string;
    isAction: boolean;
    isReusableWorkflow: boolean;
    isDocker: boolean;
    isLocal: boolean;
    isThirdParty: boolean;
    isPinnedToFullSha: boolean;
    isMutableRef: boolean; // tag, branch, major tag, etc.
  };

  run?: {
    shell?: string;
    script: string;
  };

  with: Record<string, unknown>;
  env: Record<string, string>;

  location: SourceLocation;
};
```

## The helper API I’d expose to JS

The JS side should feel like a small standard library.

### `api.gha`

Workflow-specific helpers.

```js
api.gha.isCheckout(step)
api.gha.checkoutHasPersistCredentialsFalse(step)
api.gha.isThirdPartyAction(step)
api.gha.isPinnedToFullSha(step)
api.gha.isReusableWorkflow(step)
api.gha.isPrivilegedWorkflow(workflow)
api.gha.isPrivilegedJob(job)
api.gha.eventIncludes(workflow, "pull_request_target")
api.gha.eventIncludes(workflow, "workflow_run")
```

This matters because GitHub explicitly recommends pinning third-party actions to a full commit SHA, and the `actions/checkout` README still documents `persist-credentials: false` as the opt-out. ([GitHub Docs][3])

### `api.perms`

Permission reasoning.

```js
api.perms.hasWrite(job, "contents")
api.perms.hasAnyWrite(job)
api.perms.isReadOnly(job)
api.perms.explain(job)
```

The last one should return a structured explanation like:

```json
{
  "repoDefault": "read",
  "workflowPermissions": "read-all",
  "jobPermissions": { "contents": "write" },
  "forkDowngradeApplied": true,
  "effective": { "contents": "read", "packages": "read", ... }
}
```

That makes debugging validator output much easier.

### `api.trust`

Trust-boundary reasoning.

```js
api.trust.receivesUntrustedCode(workflow)
api.trust.isPrivilegedContext(workflow)
api.trust.jobChecksOutPrHead(job)
api.trust.jobConsumesArtifactsFromUntrustedJob(job)
api.trust.describeBoundary(job)
```

This is how you address the actual class of problems rather than just linting text.

### `api.query`

A small query DSL can be surprisingly nice.

```js
api.query.steps("uses[action=checkout]")
api.query.jobs("permissions.contents=write")
api.query.workflows("on includes pull_request_target")
```

Under the hood this can just be filtering Go-side arrays, but it gives you a compact authoring style.

### `api.report`

Findings emission.

```js
api.report.finding({
  ruleId: "unpinned-third-party-action",
  severity: "high",
  message: "Third-party action is not pinned to a full commit SHA",
  location: step.location,
  evidence: {...},
  remediation: {...}
})
```

### `api.patch`

Optional autofix builders.

```js
api.patch.setYaml(step.location, ["with", "persist-credentials"], false)
api.patch.replaceUsesRef(step.location, fullSha)
api.patch.setWorkflowPermission(workflow.location, "read-all")
```

I would keep patch application in Go, not JS. JS can propose edits; Go validates and applies them.

## The kinds of rules you want

There are two broad classes.

### 1. Structural rules

Easy and reliable.

* repo default workflow permissions must be `read`
* PR approval requirement must be strict
* `canApprovePullRequestReviews` must be false
* `allowed_actions` should not be `all`
* `sha_pinning_required` should be true
* no `write-all`
* no broad workflow-level write unless explicitly justified
* no third-party action without full SHA pin

These are the baseline hardening rules. GitHub’s docs cover restricted defaults, repo/org settings for PR approval creation, permission calculation, and SHA pinning guidance. ([GitHub Docs][2])

### 2. Semantic trust-boundary rules

These are the real prize.

* `pull_request_target` or `workflow_run` plus checkout of PR/fork code
* privileged job downloads artifacts from an untrusted job
* reusable workflow inherits too much authority
* token permissions are broader than the job’s actual actions need
* secret-bearing job runs after a job that can poison the workspace/cache
* shell step interpolates untrusted context into command strings
* checkout of another repo with writable token
* external action plus write token plus secrets in same job

GitHub’s secure-use guidance is particularly relevant here: it explicitly warns against `pull_request_target` or `workflow_run` with untrusted code and calls out the significance of a single compromised action in a workflow. ([GitHub Docs][3])

## A rule authoring style I think would work well

I would support three levels:

### Simple rule

Just JS.

```js
export function validate(ctx, api) {
  for (const wf of ctx.workflows) {
    if (wf.permissions.workflow === "write-all") {
      api.report.finding({
        ruleId: "no-write-all",
        severity: "high",
        message: "Workflow uses write-all permissions",
        location: wf.location
      });
    }
  }
}
```

### Query rule

Declarative.

```js
export const query = `
  workflow where on includes "pull_request_target"
  |> jobs where trust.checksOutUntrustedRef = true
`;
```

### Semantic rule

Uses the trust model.

```js
export function validate(ctx, api) {
  for (const wf of ctx.workflows) {
    if (!api.trust.isPrivilegedContext(wf)) continue;
    for (const job of wf.jobs) {
      if (api.trust.jobChecksOutPrHead(job) && api.perms.hasAnyWrite(job)) {
        api.report.finding({
          ruleId: "pwn-request-shape",
          severity: "critical",
          message: "Privileged job checks out PR head with writable token",
          location: job.location,
        });
      }
    }
  }
}
```

## How I would keep the JS side safe

This part matters a lot, because you do not want your validator system to recreate the same mess you are trying to detect.

I would make validators:

* synchronous by default
* pure and side-effect free
* unable to make network calls
* unable to read arbitrary files
* unable to spawn processes
* unable to mutate `ctx`
* bounded by timeout and memory
* interruptible from Go

That last bit is important because Goja supports interrupting long-running scripts from the host. ([GitHub][1])

Concretely:

* one VM per validator run
* freeze all host objects before exposing them
* no `require`, no dynamic import, no eval unless you intentionally allow it
* no timers unless you later build a controlled async layer
* precompile validators once, then instantiate/run per repo
* enforce wall-clock timeout with `vm.Interrupt`
* optionally cap result size and number of findings

## Version the host API very aggressively

This is the part many plugin systems get wrong.

I’d have validators declare:

```js
export const meta = {
  id: "no-write-all",
  apiVersion: "v1",
  engineVersion: ">=0.3.0",
};
```

And your host provides only stable namespaces like:

* `api.report`
* `api.gha`
* `api.perms`
* `api.trust`
* `api.patch`
* `api.meta`

Do not let validators poke at random Go structs. Give them a narrow façade. Then you can change internals without breaking the rule pack.

## The really important Go-side semantic helpers

These are the ones I’d invest in first.

### 1. Trigger normalization

GitHub lets `on:` be a string, array, or map. Normalize all of it to a canonical representation.

### 2. `uses:` parser

You want to distinguish:

* `actions/checkout@v5`
* `actions/checkout@v6`
* `docker://...`
* `./.github/actions/foo`
* `owner/repo/path@ref`
* reusable workflow refs

### 3. Full-SHA detection

GitHub recommends pinning to a full-length commit SHA. Your helper should say clearly whether a ref is immutable or not. ([GitHub Docs][3])

### 4. Effective permission calculator

This should codify GitHub’s documented rules once and for all. ([GitHub Docs][2])

### 5. Untrusted-checkout detector

This is one of the big ones:

* event is privileged
* checkout ref comes from PR head / fork / untrusted artifact
* job has writable token or secrets

### 6. `actions/checkout` hardening detector

Your host should know whether a checkout step has `persist-credentials: false`, because the README explicitly documents that setting as the opt-out. ([GitHub][4])

## A concrete validator pack structure

```text
policies/
  core/
    no-write-all.js
    pin-third-party-actions.js
    checkout-persist-creds.js
    no-privileged-untrusted-checkout.js
    restricted-default-token.js
  org/
    org-selected-actions.js
    org-fork-approval.js
  advanced/
    no-artifact-bridge.js
    no-cache-bridge.js
    reusable-workflow-trust.js
policy.yaml
```

And `policy.yaml` could say:

```yaml
extends:
  - core

rules:
  checkout-persist-creds:
    severity: high

  pin-third-party-actions:
    severity: high
    allow:
      - actions/checkout
      - github/codeql-action

  no-privileged-untrusted-checkout:
    severity: critical

waivers:
  - repo: my-org/legacy-repo
    rule: pin-third-party-actions
    expires: 2026-06-01
    reason: waiting for upstream fix
```

## The output model

Make every finding rich enough to be useful in a PR, terminal, or UI.

```json
{
  "ruleId": "no-privileged-untrusted-checkout",
  "severity": "critical",
  "repo": "my-org/example",
  "workflow": ".github/workflows/ci.yml",
  "job": "build",
  "step": "checkout PR head",
  "message": "Privileged workflow checks out untrusted code.",
  "whyItMatters": "This can expose write tokens or secrets to attacker-controlled code.",
  "location": {
    "path": ".github/workflows/ci.yml",
    "line": 42,
    "column": 9
  },
  "evidence": {...},
  "remediation": {...}
}
```

That is the difference between a toy linter and a tool people will actually use.

## My strongest recommendation

Do **not** expose “raw GitHub repo + raw YAML” as the main scripting interface.

Expose a **security graph**.

Something like:

* nodes: workflows, jobs, steps, tokens, secrets, artifacts, caches, environments
* edges: triggers, needs, uploads, downloads, checks-out, inherits, writes-with, reads-from
* trust labels: trusted, untrusted, privileged, external-input

Then many subtle rules become simple graph queries:

* “Does any privileged node ingest data from an untrusted node?”
* “Can any writable token reach a step that runs untrusted code?”
* “Can a mutable third-party action influence a privileged path?”
* “Can artifacts bridge from `pull_request` to `workflow_run`?”

That is the model that addresses the real class of workflow-security problems, instead of only checking static knobs.

[1]: https://github.com/dop251/goja "GitHub - dop251/goja: ECMAScript/JavaScript engine in pure Go · GitHub"
[2]: https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions "Workflow syntax for GitHub Actions - GitHub Docs"
[3]: https://docs.github.com/en/enterprise-cloud%40latest/actions/reference/security/secure-use "Secure use reference - GitHub Enterprise Cloud Docs"
[4]: https://github.com/actions/checkout/blob/main/README.md?plain=1 "checkout/README.md at main · actions/checkout · GitHub"
