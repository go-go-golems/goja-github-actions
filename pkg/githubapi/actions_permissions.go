package githubapi

import "context"

func (c *Client) GetRepoActionsPermissions(ctx context.Context, owner string, repo string) (*RequestResult, error) {
	return c.DoRoute(ctx, "GET /repos/{owner}/{repo}/actions/permissions", map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	})
}

func (c *Client) GetRepoSelectedActions(ctx context.Context, owner string, repo string) (*RequestResult, error) {
	return c.DoRoute(ctx, "GET /repos/{owner}/{repo}/actions/permissions/selected-actions", map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	})
}

func (c *Client) GetRepoWorkflowPermissions(ctx context.Context, owner string, repo string) (*RequestResult, error) {
	return c.DoRoute(ctx, "GET /repos/{owner}/{repo}/actions/permissions/workflow", map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	})
}
