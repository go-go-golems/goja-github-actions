package runtime

import (
	"sync"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/pkg/errors"
)

type Bindings struct {
	Settings *Settings
	Owner    runtimeowner.Runner
}

var runtimeBindings sync.Map

type bindingsInitializer struct {
	settings *Settings
}

func NewBindingsInitializer(settings *Settings) ggjengine.RuntimeInitializer {
	return &bindingsInitializer{settings: settings}
}

func (i *bindingsInitializer) ID() string {
	return "goja-gha-bindings"
}

func (i *bindingsInitializer) InitRuntime(ctx *ggjengine.RuntimeContext) error {
	if ctx == nil || ctx.VM == nil || ctx.Owner == nil {
		return errors.New("runtime context is incomplete")
	}

	runtimeBindings.Store(ctx.VM, &Bindings{
		Settings: i.settings,
		Owner:    ctx.Owner,
	})
	return nil
}

func LookupBindings(vm *goja.Runtime) (*Bindings, bool) {
	if vm == nil {
		return nil, false
	}

	value, ok := runtimeBindings.Load(vm)
	if !ok {
		return nil, false
	}

	bindings, ok := value.(*Bindings)
	return bindings, ok
}

func UnregisterBindings(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	runtimeBindings.Delete(vm)
}
