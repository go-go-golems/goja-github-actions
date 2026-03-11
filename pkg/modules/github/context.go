package githubmodule

import (
	"encoding/json"

	"github.com/dop251/goja"
)

func (m *Module) newContextObject() *goja.Object {
	payload, err := json.Marshal(m.context)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	var exported map[string]interface{}
	if err := json.Unmarshal(payload, &exported); err != nil {
		panic(m.vm.NewGoError(err))
	}

	return m.vm.ToValue(exported).ToObject(m.vm)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
