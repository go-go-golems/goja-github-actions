package workflowmodule

import (
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
	"github.com/go-go-golems/goja-github-actions/pkg/workflows"
	"github.com/pkg/errors"
)

const moduleName = "@goja-gha/workflows"

type Dependencies struct {
	Settings *gharuntime.Settings
}

type Module struct {
	vm   *goja.Runtime
	deps *Dependencies
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-workflows",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			mod := &Module{
				vm:   vm,
				deps: deps,
			}
			exports := moduleObj.Get("exports").(*goja.Object)
			modules.SetExport(exports, moduleName, "listFiles", mod.listFiles)
			modules.SetExport(exports, moduleName, "parseAll", mod.parseAll)
			modules.SetExport(exports, moduleName, "parseFile", mod.parseFile)
		},
	}
}

func (m *Module) listFiles() goja.Value {
	files, err := workflows.ListFiles(m.root())
	m.must(err)
	return m.vm.ToValue(files)
}

func (m *Module) parseAll() goja.Value {
	docs, err := workflows.ParseAll(m.root())
	m.must(err)
	return m.vm.ToValue(documentsToMaps(docs))
}

func (m *Module) parseFile(path string) goja.Value {
	doc, err := workflows.ParseFile(m.root(), strings.TrimSpace(path))
	m.must(err)
	return m.vm.ToValue(documentToMap(doc))
}

func (m *Module) root() string {
	if m.deps == nil || m.deps.Settings == nil {
		return "."
	}
	return m.deps.Settings.ExecutionRoot()
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}

func documentsToMaps(docs []workflows.Document) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		ret = append(ret, documentToMap(doc))
	}
	return ret
}

func documentToMap(doc workflows.Document) map[string]interface{} {
	return map[string]interface{}{
		"fileName":      doc.FileName,
		"path":          doc.Path,
		"name":          doc.Name,
		"triggerNames":  append([]string(nil), doc.TriggerNames...),
		"uses":          usesToMaps(doc.Uses),
		"checkoutSteps": checkoutStepsToMaps(doc.CheckoutSteps),
		"runSteps":      runStepsToMaps(doc.RunSteps),
		"permissions":   permissionsToMaps(doc.Permissions),
	}
}

func usesToMaps(values []workflows.UsesReference) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		ret = append(ret, map[string]interface{}{
			"kind":     value.Kind,
			"jobId":    value.JobID,
			"stepName": value.StepName,
			"uses":     value.Uses,
			"line":     value.Line,
		})
	}
	return ret
}

func checkoutStepsToMaps(values []workflows.CheckoutStep) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		entry := map[string]interface{}{
			"jobId":    value.JobID,
			"stepName": value.StepName,
			"uses":     value.Uses,
			"line":     value.Line,
		}
		if value.PersistCredentials == nil {
			entry["persistCredentials"] = nil
		} else {
			entry["persistCredentials"] = *value.PersistCredentials
		}
		if value.Ref == nil {
			entry["ref"] = nil
		} else {
			entry["ref"] = *value.Ref
		}
		if value.Repository == nil {
			entry["repository"] = nil
		} else {
			entry["repository"] = *value.Repository
		}
		ret = append(ret, entry)
	}
	return ret
}

func runStepsToMaps(values []workflows.RunStep) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		ret = append(ret, map[string]interface{}{
			"jobId":    value.JobID,
			"stepName": value.StepName,
			"run":      value.Run,
			"line":     value.Line,
		})
	}
	return ret
}

func permissionsToMaps(values []workflows.PermissionEntry) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		ret = append(ret, map[string]interface{}{
			"scope": value.Scope,
			"jobId": value.JobID,
			"line":  value.Line,
			"kind":  value.Kind,
			"value": normalizePermissionValue(value.Value),
		})
	}
	return ret
}

func normalizePermissionValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]string:
		ret := map[string]interface{}{}
		for key, value := range typed {
			ret[key] = value
		}
		return ret
	case nil, string:
		return typed
	default:
		return errors.Errorf("unsupported permission value type %T", value).Error()
	}
}
