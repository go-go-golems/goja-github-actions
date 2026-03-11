package coremodule

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/goja-github-actions/pkg/runnerfiles"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
	"github.com/pkg/errors"
)

const moduleName = "@actions/core"

type Dependencies struct {
	Settings    *gharuntime.Settings
	EnvFile     runnerfiles.EnvFile
	OutputFile  runnerfiles.OutputFile
	PathFile    runnerfiles.PathFile
	SummaryFile runnerfiles.SummaryFile
}

type Module struct {
	vm      *goja.Runtime
	deps    *Dependencies
	summary *summaryBuilder
}

func NewDependencies(settings *gharuntime.Settings) *Dependencies {
	return &Dependencies{
		Settings:    settings,
		EnvFile:     runnerfiles.EnvFile{Path: settings.RunnerEnvFile},
		OutputFile:  runnerfiles.OutputFile{Path: settings.RunnerOutputFile},
		PathFile:    runnerfiles.PathFile{Path: settings.RunnerPathFile},
		SummaryFile: runnerfiles.SummaryFile{Path: settings.RunnerSummaryFile},
	}
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-actions-core",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			mod := &Module{
				vm:      vm,
				deps:    deps,
				summary: newSummaryBuilder(deps.SummaryFile),
			}
			exports := moduleObj.Get("exports").(*goja.Object)

			modules.SetExport(exports, moduleName, "getInput", mod.getInput)
			modules.SetExport(exports, moduleName, "getBooleanInput", mod.getBooleanInput)
			modules.SetExport(exports, moduleName, "getMultilineInput", mod.getMultilineInput)
			modules.SetExport(exports, moduleName, "setOutput", mod.setOutput)
			modules.SetExport(exports, moduleName, "exportVariable", mod.exportVariable)
			modules.SetExport(exports, moduleName, "addPath", mod.addPath)
			modules.SetExport(exports, moduleName, "setFailed", mod.setFailed)
			modules.SetExport(exports, moduleName, "debug", mod.debug)
			modules.SetExport(exports, moduleName, "info", mod.info)
			modules.SetExport(exports, moduleName, "notice", mod.notice)
			modules.SetExport(exports, moduleName, "warning", mod.warning)
			modules.SetExport(exports, moduleName, "error", mod.errorMessage)
			modules.SetExport(exports, moduleName, "setSecret", mod.setSecret)
			modules.SetExport(exports, moduleName, "startGroup", mod.startGroup)
			modules.SetExport(exports, moduleName, "endGroup", mod.endGroup)
			modules.SetExport(exports, moduleName, "group", mod.group)
			modules.SetExport(exports, moduleName, "summary", mod.newSummaryObject())
		},
	}
}

func (m *Module) state() *gharuntime.State {
	if m.deps.Settings.State == nil {
		m.deps.Settings.State = &gharuntime.State{
			Environment: m.deps.Settings.ProcessEnv(),
		}
	}
	return m.deps.Settings.State
}

func (m *Module) updateProcessEnv(key string, value string) {
	state := m.state()
	if state.Environment == nil {
		state.Environment = map[string]string{}
	}
	state.Environment[key] = value

	processValue := m.vm.Get("process")
	if goja.IsUndefined(processValue) || goja.IsNull(processValue) {
		return
	}
	processObject := processValue.ToObject(m.vm)
	envValue := processObject.Get("env")
	if goja.IsUndefined(envValue) || goja.IsNull(envValue) {
		return
	}
	envObject := envValue.ToObject(m.vm)
	_ = envObject.Set(key, value)
}

func (m *Module) updateProcessPath(path string) {
	current := m.state().Environment["PATH"]
	if current == "" {
		m.updateProcessEnv("PATH", path)
		return
	}
	m.updateProcessEnv("PATH", path+string(os.PathListSeparator)+current)
}

func (m *Module) setExitCode(code int, failureMessage string) {
	state := m.state()
	state.ExitCode = code
	state.FailureMessage = failureMessage

	processValue := m.vm.Get("process")
	if goja.IsUndefined(processValue) || goja.IsNull(processValue) {
		return
	}
	processObject := processValue.ToObject(m.vm)
	_ = processObject.Set("exitCode", code)
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}

func normalizeInputKey(name string) string {
	normalized := strings.TrimSpace(strings.ToUpper(name))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return "INPUT_" + normalized
}

func stringifyValue(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ""
	}
	exported := value.Export()
	switch typed := exported.(type) {
	case string:
		return typed
	default:
		payload, err := json.Marshal(typed)
		if err == nil {
			return string(payload)
		}
		return fmt.Sprint(typed)
	}
}

func requiredError(name string) error {
	return errors.Errorf("input %q is required", name)
}
