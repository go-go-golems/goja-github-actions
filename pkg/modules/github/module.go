package githubmodule

import (
	"context"
	"net/url"
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/goja-github-actions/pkg/contextdata"
	"github.com/go-go-golems/goja-github-actions/pkg/githubapi"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

const moduleName = "@actions/github"

type Dependencies struct {
	Settings *gharuntime.Settings
}

type Module struct {
	vm      *goja.Runtime
	deps    *Dependencies
	context *contextdata.GitHubContext
}

type octokitOptions struct {
	BaseURL string `json:"baseUrl"`
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-actions-github",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			ghContext, err := contextdata.BuildGitHubContext(deps.Settings)
			if err != nil {
				panic(vm.NewGoError(err))
			}

			mod := &Module{
				vm:      vm,
				deps:    deps,
				context: ghContext,
			}

			exports := moduleObj.Get("exports").(*goja.Object)
			modules.SetExport(exports, moduleName, "context", mod.newContextObject())
			modules.SetExport(exports, moduleName, "getOctokit", mod.getOctokit)
		},
	}
}

func (m *Module) getOctokit(call goja.FunctionCall) goja.Value {
	token := strings.TrimSpace(call.Argument(0).String())
	if token == "" {
		token = m.deps.Settings.GitHubToken
	}

	options := octokitOptions{}
	if arg := call.Argument(1); arg != nil && !goja.IsUndefined(arg) && !goja.IsNull(arg) {
		if err := m.vm.ExportTo(arg, &options); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}

	baseURL := options.BaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = firstNonEmpty(
			m.deps.Settings.ProcessEnv()["GITHUB_API_URL"],
			"https://api.github.com",
		)
	}

	client := githubapi.NewClient(token, normalizeBaseURL(baseURL))
	return m.newOctokitObject(client)
}

func (m *Module) withClientContext() context.Context {
	return context.Background()
}

func normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "https://api.github.com"
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		return strings.TrimRight(trimmed, "/")
	}
	return "https://api.github.com"
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}
