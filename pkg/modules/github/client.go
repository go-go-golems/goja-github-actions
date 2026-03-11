package githubmodule

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/goja-github-actions/pkg/githubapi"
)

func (m *Module) newOctokitObject(client *githubapi.Client) *goja.Object {
	octokit := m.vm.NewObject()
	restObject := m.vm.NewObject()

	m.must(octokit.Set("request", func(route string, params map[string]interface{}) interface{} {
		result, err := client.DoRoute(m.withClientContext(), route, params)
		m.must(err)
		return normalizeResult(result)
	}))

	m.must(octokit.Set("paginate", func(route string, params map[string]interface{}) interface{} {
		return m.paginate(client, route, params)
	}))

	m.must(restObject.Set("actions", m.newActionsObject(client)))
	m.must(octokit.Set("rest", restObject))
	return octokit
}

func (m *Module) paginate(client *githubapi.Client, route string, params map[string]interface{}) interface{} {
	allData := []interface{}{}
	currentRoute := route
	currentParams := cloneParams(params)

	for {
		result, err := client.DoRoute(m.withClientContext(), currentRoute, currentParams)
		m.must(err)

		normalized := normalizeResult(result)
		data := normalized["data"]
		switch data := data.(type) {
		case []interface{}:
			allData = append(allData, data...)
		case map[string]interface{}:
			if workflows, ok := data["workflows"].([]interface{}); ok {
				allData = append(allData, workflows...)
			} else if items, ok := data["items"].([]interface{}); ok {
				allData = append(allData, items...)
			} else {
				allData = append(allData, data)
			}
		case nil:
		default:
			allData = append(allData, data)
		}

		headers, _ := normalized["headers"].(map[string][]string)
		nextURL := parseNextLink(headers[http.CanonicalHeaderKey("Link")])
		if nextURL == "" {
			break
		}

		currentRoute = http.MethodGet + " " + nextURL
		currentParams = map[string]interface{}{}
	}

	return allData
}

func parseNextLink(values []string) string {
	for _, value := range values {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			if !strings.Contains(part, `rel="next"`) {
				continue
			}
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func cloneParams(params map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	for key, value := range params {
		ret[key] = value
	}
	return ret
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func normalizeResult(result *githubapi.RequestResult) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"status":  result.Status,
		"data":    result.Data,
		"headers": result.Headers,
	}
}
