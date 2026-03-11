package coremodule

import (
	"strings"

	"github.com/dop251/goja"
)

type summaryBuilder struct {
	file interface {
		Append(string) error
		Clear() error
	}
	lines []string
}

func newSummaryBuilder(file interface {
	Append(string) error
	Clear() error
}) *summaryBuilder {
	return &summaryBuilder{file: file}
}

func (m *Module) setOutput(name string, value goja.Value) {
	stringValue := stringifyValue(value)
	if strings.Contains(stringValue, "\n") {
		m.must(m.deps.OutputFile.SetMultiline(name, stringValue))
		return
	}
	m.must(m.deps.OutputFile.Set(name, stringValue))
}

func (m *Module) exportVariable(name string, value goja.Value) {
	stringValue := stringifyValue(value)
	if strings.Contains(stringValue, "\n") {
		m.must(m.deps.EnvFile.SetMultiline(name, stringValue))
	} else {
		m.must(m.deps.EnvFile.Set(name, stringValue))
	}
	m.updateProcessEnv(name, stringValue)
}

func (m *Module) addPath(path string) {
	m.must(m.deps.PathFile.Add(path))
	m.updateProcessPath(path)
}

func (m *Module) setFailed(message string) {
	if strings.TrimSpace(message) != "" {
		m.errorMessage(message)
	}
	m.setExitCode(1, message)
}

func (m *Module) newSummaryObject() *goja.Object {
	summaryObject := m.vm.NewObject()

	m.must(summaryObject.Set("addRaw", func(text string) *goja.Object {
		m.summary.lines = append(m.summary.lines, text)
		return summaryObject
	}))
	m.must(summaryObject.Set("addHeading", func(text string) *goja.Object {
		m.summary.lines = append(m.summary.lines, "# "+text+"\n")
		return summaryObject
	}))
	m.must(summaryObject.Set("write", func() *goja.Object {
		m.must(m.summary.file.Append(strings.Join(m.summary.lines, "")))
		return summaryObject
	}))
	m.must(summaryObject.Set("clear", func() *goja.Object {
		m.summary.lines = nil
		m.must(m.summary.file.Clear())
		return summaryObject
	}))

	return summaryObject
}
