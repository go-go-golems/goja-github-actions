package execmodule

import (
	"context"
	"os"
	osExec "os/exec"
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
	"github.com/pkg/errors"
)

const moduleName = "@actions/exec"

type Dependencies struct {
	Settings *gharuntime.Settings
}

type Module struct {
	vm       *goja.Runtime
	deps     *Dependencies
	bindings *gharuntime.Bindings
}

type execOptions struct {
	Cwd              string            `json:"cwd"`
	Env              map[string]string `json:"env"`
	IgnoreReturnCode bool              `json:"ignoreReturnCode"`
	Silent           bool              `json:"silent"`
	CaptureOutput    bool              `json:"captureOutput"`
	Listeners        *goja.Object
}

type execListeners struct {
	stdout goja.Callable
	stderr goja.Callable
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-actions-exec",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			bindings, ok := gharuntime.LookupBindings(vm)
			if !ok {
				panic(vm.NewGoError(errors.New("runtime bindings are not available")))
			}

			mod := &Module{
				vm:       vm,
				deps:     deps,
				bindings: bindings,
			}

			exports := moduleObj.Get("exports").(*goja.Object)
			modules.SetExport(exports, moduleName, "exec", mod.exec)
		},
	}
}

func (m *Module) exec(call goja.FunctionCall) goja.Value {
	command := strings.TrimSpace(call.Argument(0).String())
	if command == "" {
		panic(m.vm.NewTypeError("exec(command, args?, options?) requires a command"))
	}

	args, options := m.parseArgsAndOptions(call)
	listeners := m.parseListeners(options)
	promise, resolve, reject := m.vm.NewPromise()

	go func() {
		result, err := m.runCommand(context.Background(), command, args, options, listeners)
		postErr := m.bindings.Owner.Post(context.Background(), "actions-exec-settle", func(_ context.Context, vm *goja.Runtime) {
			if err != nil {
				_ = reject(vm.ToValue(err.Error()))
				return
			}
			_ = resolve(vm.ToValue(result.toMap()))
		})
		_ = postErr
	}()

	return m.vm.ToValue(promise)
}

func (m *Module) parseArgsAndOptions(call goja.FunctionCall) ([]string, execOptions) {
	args := []string{}
	options := execOptions{}

	switch len(call.Arguments) {
	case 0, 1:
		return args, options
	}

	second := call.Argument(1)
	if second != nil && !goja.IsUndefined(second) && !goja.IsNull(second) {
		exported := second.Export()
		switch exported.(type) {
		case []interface{}:
			m.must(m.vm.ExportTo(second, &args))
		default:
			options = m.decodeOptions(second)
		}
	}

	if len(call.Arguments) > 2 {
		third := call.Argument(2)
		if third != nil && !goja.IsUndefined(third) && !goja.IsNull(third) {
			options = m.decodeOptions(third)
		}
	}

	return args, options
}

func (m *Module) decodeOptions(value goja.Value) execOptions {
	options := execOptions{}
	object := value.ToObject(m.vm)
	if object == nil {
		return options
	}

	options.Cwd = stringProperty(object, "cwd")
	options.IgnoreReturnCode = boolProperty(object, "ignoreReturnCode")
	options.Silent = boolProperty(object, "silent")
	options.CaptureOutput = boolProperty(object, "captureOutput")

	envValue := object.Get("env")
	if envValue != nil && !goja.IsUndefined(envValue) && !goja.IsNull(envValue) {
		env := map[string]string{}
		m.must(m.vm.ExportTo(envValue, &env))
		options.Env = env
	}

	listenersValue := object.Get("listeners")
	if listenersValue != nil && !goja.IsUndefined(listenersValue) && !goja.IsNull(listenersValue) {
		options.Listeners = listenersValue.ToObject(m.vm)
	}

	return options
}

func (m *Module) parseListeners(options execOptions) execListeners {
	if options.Listeners == nil {
		return execListeners{}
	}

	listeners := execListeners{}
	if callback, ok := goja.AssertFunction(options.Listeners.Get("stdout")); ok {
		listeners.stdout = callback
	}
	if callback, ok := goja.AssertFunction(options.Listeners.Get("stderr")); ok {
		listeners.stderr = callback
	}
	return listeners
}

func (m *Module) runCommand(
	ctx context.Context,
	command string,
	args []string,
	options execOptions,
	listeners execListeners,
) (*execResult, error) {
	cmd := osExec.CommandContext(ctx, command, args...)
	cmd.Dir = m.cwd(options)
	cmd.Env = environmentSlice(m.environment(options))

	stdoutWriter := &listenerWriter{
		runner:   m.bindings.Owner,
		callback: listeners.stdout,
		vm:       m.vm,
	}
	stderrWriter := &listenerWriter{
		runner:   m.bindings.Owner,
		callback: listeners.stderr,
		vm:       m.vm,
	}

	stdoutBuffer := newBufferWriter()
	stderrBuffer := newBufferWriter()

	cmd.Stdout = combineWriters(
		stdoutBuffer,
		stdoutWriter,
		optionalWriter(!options.Silent, os.Stdout),
	)
	cmd.Stderr = combineWriters(
		stderrBuffer,
		stderrWriter,
		optionalWriter(!options.Silent, os.Stderr),
	)

	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		exitCode = -1
	}
	result := &execResult{
		exitCode: exitCode,
		stdout:   stdoutBuffer.String(),
		stderr:   stderrBuffer.String(),
	}

	if err != nil && !options.IgnoreReturnCode {
		return nil, errors.Wrapf(err, "exec %s %s failed", command, strings.Join(args, " "))
	}

	return result, nil
}

func (m *Module) cwd(options execOptions) string {
	if strings.TrimSpace(options.Cwd) != "" {
		return options.Cwd
	}
	if m.deps != nil && m.deps.Settings != nil && strings.TrimSpace(m.deps.Settings.WorkingDirectory) != "" {
		return m.deps.Settings.WorkingDirectory
	}
	return "."
}

func (m *Module) environment(options execOptions) map[string]string {
	base := map[string]string{}
	if m.deps != nil && m.deps.Settings != nil {
		base = m.deps.Settings.ProcessEnv()
	}
	for key, value := range options.Env {
		base[key] = value
	}
	return base
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}

func stringProperty(object *goja.Object, key string) string {
	value := object.Get(key)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ""
	}
	return value.String()
}

func boolProperty(object *goja.Object, key string) bool {
	value := object.Get(key)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return false
	}
	return value.ToBoolean()
}

type listenerWriter struct {
	runner   runtimeowner.Runner
	callback goja.Callable
	vm       *goja.Runtime
}

type execResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func (r *execResult) toMap() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"exitCode": r.exitCode,
		"stdout":   r.stdout,
		"stderr":   r.stderr,
	}
}

func (w *listenerWriter) Write(p []byte) (int, error) {
	if w == nil || w.callback == nil {
		return len(p), nil
	}

	chunk := string(append([]byte(nil), p...))
	err := w.runner.Post(context.Background(), "actions-exec-listener", func(_ context.Context, vm *goja.Runtime) {
		_, _ = w.callback(goja.Undefined(), vm.ToValue(chunk))
	})
	if err != nil {
		return 0, err
	}

	return len(p), nil
}
