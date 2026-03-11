package githubapi

import "context"

func (c *Client) GetRepoWorkflowContents(ctx context.Context, owner string, repo string, ref string) (*RequestResult, error) {
	params := map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	}
	if ref != "" {
		params["ref"] = ref
	}
	return c.DoRoute(ctx, "GET /repos/{owner}/{repo}/contents/.github/workflows", params)
}
