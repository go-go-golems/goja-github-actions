package githubapi

import "context"

func (c *Client) ListRepoWorkflows(ctx context.Context, owner string, repo string) (*RequestResult, error) {
	return c.DoRoute(ctx, "GET /repos/{owner}/{repo}/actions/workflows", map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	})
}
