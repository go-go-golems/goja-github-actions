package iomodule

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

const moduleName = "@actions/io"

type Dependencies struct {
	Settings *gharuntime.Settings
}

type Module struct {
	vm   *goja.Runtime
	deps *Dependencies
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-actions-io",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			mod := &Module{vm: vm, deps: deps}
			exports := moduleObj.Get("exports").(*goja.Object)

			modules.SetExport(exports, moduleName, "readdir", mod.readdir)
			modules.SetExport(exports, moduleName, "readFile", mod.readFile)
			modules.SetExport(exports, moduleName, "writeFile", mod.writeFile)
			modules.SetExport(exports, moduleName, "mkdirP", mod.mkdirP)
			modules.SetExport(exports, moduleName, "rmRF", mod.rmRF)
			modules.SetExport(exports, moduleName, "cp", mod.copy)
			modules.SetExport(exports, moduleName, "mv", mod.move)
			modules.SetExport(exports, moduleName, "which", mod.which)
		},
	}
}

func (m *Module) workingDirectory() string {
	if m.deps == nil || m.deps.Settings == nil {
		return "."
	}
	return m.deps.Settings.ExecutionRoot()
}

func (m *Module) resolvePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return m.workingDirectory()
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(m.workingDirectory(), trimmed)
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}

func sortedEntries(entries []string) []string {
	ret := append([]string(nil), entries...)
	sort.Strings(ret)
	return ret
}

func lookupPath(binary string) (string, error) {
	return exec.LookPath(binary)
}
