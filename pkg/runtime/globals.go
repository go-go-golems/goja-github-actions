package runtime

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/pkg/errors"
)

type processInitializer struct {
	settings *Settings
}

func NewProcessInitializer(settings *Settings) ggjengine.RuntimeInitializer {
	return &processInitializer{settings: settings}
}

func (i *processInitializer) ID() string {
	return "goja-gha-process"
}

func (i *processInitializer) InitRuntime(ctx *ggjengine.RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return errors.New("runtime context is incomplete")
	}
	if i.settings.State == nil {
		i.settings.State = &State{
			Environment: i.settings.ProcessEnv(),
		}
	}

	processObject := ctx.VM.NewObject()
	envObject := ctx.VM.NewObject()

	env := i.settings.ProcessEnv()
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := envObject.Set(key, env[key]); err != nil {
			return errors.Wrapf(err, "set process.env.%s", key)
		}
	}

	if err := processObject.Set("env", envObject); err != nil {
		return errors.Wrap(err, "set process.env")
	}
	if err := processObject.Set("cwd", func() string { return i.settings.ExecutionRoot() }); err != nil {
		return errors.Wrap(err, "set process.cwd")
	}
	if err := processObject.Set("workingDirectory", i.settings.ExecutionRoot()); err != nil {
		return errors.Wrap(err, "set process.workingDirectory")
	}
	if err := processObject.Set("exitCode", i.settings.State.ExitCode); err != nil {
		return errors.Wrap(err, "set process.exitCode")
	}

	stdoutObject, err := newStreamObject(ctx.VM, os.Stdout)
	if err != nil {
		return err
	}
	stderrObject, err := newStreamObject(ctx.VM, os.Stderr)
	if err != nil {
		return err
	}
	if err := processObject.Set("stdout", stdoutObject); err != nil {
		return errors.Wrap(err, "set process.stdout")
	}
	if err := processObject.Set("stderr", stderrObject); err != nil {
		return errors.Wrap(err, "set process.stderr")
	}

	if err := ctx.VM.Set("process", processObject); err != nil {
		return errors.Wrap(err, "set process global")
	}

	return nil
}

func newStreamObject(vm *goja.Runtime, writer io.Writer) (*goja.Object, error) {
	stream := vm.NewObject()
	if err := stream.Set("write", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		}

		_, _ = io.WriteString(writer, fmt.Sprint(call.Arguments[0].Export()))
		return goja.Undefined()
	}); err != nil {
		return nil, errors.Wrap(err, "set stream.write")
	}
	return stream, nil
}

func (s *Settings) ProcessEnv() map[string]string {
	if s.State != nil && s.State.Environment != nil {
		return cloneEnvironment(s.State.Environment)
	}

	env := map[string]string{}
	for key, value := range s.AmbientEnvironment {
		env[key] = value
	}

	if s.Workspace != "" {
		env["GITHUB_WORKSPACE"] = s.Workspace
	}
	if s.EventPath != "" {
		env["GITHUB_EVENT_PATH"] = s.EventPath
	}
	if s.ActionPath != "" {
		env["GITHUB_ACTION_PATH"] = s.ActionPath
	}
	if s.RunnerEnvFile != "" {
		env["GITHUB_ENV"] = s.RunnerEnvFile
	}
	if s.RunnerOutputFile != "" {
		env["GITHUB_OUTPUT"] = s.RunnerOutputFile
	}
	if s.RunnerPathFile != "" {
		env["GITHUB_PATH"] = s.RunnerPathFile
	}
	if s.RunnerSummaryFile != "" {
		env["GITHUB_STEP_SUMMARY"] = s.RunnerSummaryFile
	}
	if s.GitHubToken != "" {
		env["GITHUB_TOKEN"] = s.GitHubToken
	}

	return env
}

func cloneEnvironment(source map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range source {
		env[key] = value
	}
	return env
}
