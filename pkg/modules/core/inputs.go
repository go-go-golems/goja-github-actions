package coremodule

import (
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

type inputOptions struct {
	Required       bool  `json:"required"`
	TrimWhitespace *bool `json:"trimWhitespace"`
}

func (m *Module) getInput(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).String()
	options := m.decodeInputOptions(call.Argument(1))
	value := m.state().Environment[normalizeInputKey(name)]
	if options.TrimWhitespace == nil || *options.TrimWhitespace {
		value = strings.TrimSpace(value)
	}
	if options.Required && value == "" {
		panic(m.vm.NewGoError(requiredError(name)))
	}
	return m.vm.ToValue(value)
}

func (m *Module) getBooleanInput(call goja.FunctionCall) goja.Value {
	value := m.getInput(call).String()
	if value == "" {
		return m.vm.ToValue(false)
	}

	parsed, err := strconv.ParseBool(strings.ToLower(value))
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.vm.ToValue(parsed)
}

func (m *Module) getMultilineInput(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).String()
	options := m.decodeInputOptions(call.Argument(1))
	value := m.state().Environment[normalizeInputKey(name)]
	lines := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		if options.TrimWhitespace == nil || *options.TrimWhitespace {
			line = strings.TrimSpace(line)
		}
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if options.Required && len(lines) == 0 {
		panic(m.vm.NewGoError(requiredError(name)))
	}
	return m.vm.ToValue(lines)
}

func (m *Module) decodeInputOptions(value goja.Value) inputOptions {
	options := inputOptions{}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return options
	}

	if err := m.vm.ExportTo(value, &options); err != nil {
		panic(m.vm.NewGoError(err))
	}

	return options
}
