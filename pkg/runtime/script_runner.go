package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/pkg/errors"
)

func RunScript(ctx context.Context, settings *Settings) (goja.Value, error) {
	return RunScriptWithModules(ctx, settings)
}

func RunScriptWithModules(
	ctx context.Context,
	settings *Settings,
	modules ...ggjengine.ModuleSpec,
) (goja.Value, error) {
	rt, err := CreateRuntime(ctx, settings, modules...)
	if err != nil {
		return nil, err
	}
	defer func() {
		UnregisterBindings(rt.VM)
		_ = rt.Close(ctx)
	}()

	return RunScriptWithRuntime(ctx, rt, settings)
}

func RunScriptWithRuntime(
	ctx context.Context,
	rt *ggjengine.Runtime,
	settings *Settings,
) (goja.Value, error) {
	entrypoint := ModuleEntrypoint(settings)
	moduleValue, err := rt.Require.Require(entrypoint)
	if err != nil {
		return nil, errors.Wrapf(err, "load entrypoint %q", entrypoint)
	}

	if fn, ok := goja.AssertFunction(moduleValue); ok {
		result, err := callFunction(ctx, rt, fn)
		if err != nil {
			return nil, errors.Wrap(err, "execute exported function")
		}
		return awaitPromise(ctx, rt, result)
	}

	moduleObject := moduleValue.ToObject(rt.VM)
	if moduleObject == nil {
		return moduleValue, nil
	}

	for _, candidate := range []string{"main", "default"} {
		value := moduleObject.Get(candidate)
		if fn, ok := goja.AssertFunction(value); ok {
			result, err := callFunction(ctx, rt, fn)
			if err != nil {
				return nil, errors.Wrapf(err, "execute exported %s function", candidate)
			}
			return awaitPromise(ctx, rt, result)
		}
	}

	return awaitPromise(ctx, rt, moduleValue)
}

func callFunction(
	ctx context.Context,
	rt *ggjengine.Runtime,
	fn goja.Callable,
) (goja.Value, error) {
	value, err := rt.Owner.Call(ctx, "run-entrypoint", func(_ context.Context, vm *goja.Runtime) (any, error) {
		result, err := fn(goja.Undefined())
		if err != nil {
			return nil, err
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}

	result, ok := value.(goja.Value)
	if !ok {
		return nil, errors.Errorf("unexpected function result type %T", value)
	}
	return result, nil
}

type promiseSnapshot struct {
	State goja.PromiseState
	Value goja.Value
}

func awaitPromise(
	ctx context.Context,
	rt *ggjengine.Runtime,
	value goja.Value,
) (goja.Value, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return value, nil
	}

	promise, ok := value.Export().(*goja.Promise)
	if !ok {
		return value, nil
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		snapshotAny, err := rt.Owner.Call(ctx, "await-promise", func(_ context.Context, _ *goja.Runtime) (any, error) {
			return promiseSnapshot{
				State: promise.State(),
				Value: promise.Result(),
			}, nil
		})
		if err != nil {
			return nil, err
		}

		snapshot, ok := snapshotAny.(promiseSnapshot)
		if !ok {
			return nil, errors.Errorf("unexpected promise snapshot type %T", snapshotAny)
		}

		switch snapshot.State {
		case goja.PromiseStateFulfilled:
			if snapshot.Value == nil {
				return goja.Undefined(), nil
			}
			return snapshot.Value, nil
		case goja.PromiseStateRejected:
			if snapshot.Value == nil {
				return nil, errors.New("promise rejected")
			}
			return nil, errors.Errorf("promise rejected: %s", snapshot.Value.String())
		case goja.PromiseStatePending:
			if ctx != nil && ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if time.Now().After(deadline) {
				return nil, errors.New("timed out waiting for promise")
			}
			time.Sleep(10 * time.Millisecond)
		default:
			return nil, fmt.Errorf("unknown promise state %v", snapshot.State)
		}
	}
}
