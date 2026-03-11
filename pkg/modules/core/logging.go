package coremodule

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
)

func (m *Module) debug(message string) {
	_, _ = fmt.Fprintf(os.Stdout, "::debug::%s\n", message)
}

func (m *Module) info(message string) {
	_, _ = fmt.Fprintln(os.Stdout, message)
}

func (m *Module) notice(message string) {
	_, _ = fmt.Fprintf(os.Stdout, "::notice::%s\n", message)
}

func (m *Module) warning(message string) {
	_, _ = fmt.Fprintf(os.Stdout, "::warning::%s\n", message)
}

func (m *Module) errorMessage(message string) {
	_, _ = fmt.Fprintf(os.Stdout, "::error::%s\n", message)
}

func (m *Module) setSecret(secret string) {
	_, _ = fmt.Fprintf(os.Stdout, "::add-mask::%s\n", secret)
}

func (m *Module) startGroup(name string) {
	_, _ = fmt.Fprintf(os.Stdout, "::group::%s\n", name)
}

func (m *Module) endGroup() {
	_, _ = fmt.Fprintln(os.Stdout, "::endgroup::")
}

func (m *Module) group(name string, fn goja.Value) goja.Value {
	callable, ok := goja.AssertFunction(fn)
	if !ok {
		panic(m.vm.NewTypeError("group expects a function"))
	}

	m.startGroup(name)
	defer m.endGroup()

	result, err := callable(goja.Undefined())
	if err != nil {
		panic(err)
	}
	return result
}
