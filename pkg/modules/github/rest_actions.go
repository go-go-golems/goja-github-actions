package githubmodule

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/goja-github-actions/pkg/githubapi"
)

func (m *Module) newActionsObject(client *githubapi.Client) *goja.Object {
	actionsObject := m.vm.NewObject()

	m.must(actionsObject.Set("getGithubActionsPermissionsRepository", func(params map[string]interface{}) interface{} {
		result, err := client.GetRepoActionsPermissions(m.withClientContext(), toString(params["owner"]), toString(params["repo"]))
		m.must(err)
		return normalizeResult(result)
	}))
	m.must(actionsObject.Set("getAllowedActionsRepository", func(params map[string]interface{}) interface{} {
		result, err := client.GetRepoSelectedActions(m.withClientContext(), toString(params["owner"]), toString(params["repo"]))
		m.must(err)
		return normalizeResult(result)
	}))
	m.must(actionsObject.Set("getWorkflowPermissionsRepository", func(params map[string]interface{}) interface{} {
		result, err := client.GetRepoWorkflowPermissions(m.withClientContext(), toString(params["owner"]), toString(params["repo"]))
		m.must(err)
		return normalizeResult(result)
	}))
	m.must(actionsObject.Set("listRepoWorkflows", func(params map[string]interface{}) interface{} {
		result, err := client.ListRepoWorkflows(m.withClientContext(), toString(params["owner"]), toString(params["repo"]))
		m.must(err)
		return normalizeResult(result)
	}))

	return actionsObject
}
