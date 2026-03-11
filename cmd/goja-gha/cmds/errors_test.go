package cmds

import (
	"net/http"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/goja-github-actions/pkg/githubapi"
	"github.com/pkg/errors"
)

func TestFormatCLIErrorForWrappedGitHubAPIError(t *testing.T) {
	t.Parallel()

	vm := goja.New()
	raw := &githubapi.APIError{
		Status:  http.StatusConflict,
		Message: "Conflict",
		Route:   "GET /repos/{owner}/{repo}/actions/permissions/selected-actions",
	}

	var err error
	vm.Set("fail", func() {
		panic(vm.NewGoError(raw))
	})
	_, err = vm.RunString(`fail()`)
	if err == nil {
		t.Fatal("expected JS execution to fail")
	}

	formatted := FormatCLIError(errors.Wrap(err, "execute exported function"))
	if !strings.Contains(formatted, "GitHub API request failed") {
		t.Fatalf("formatted error missing heading: %s", formatted)
	}
	if !strings.Contains(formatted, "Route: GET /repos/{owner}/{repo}/actions/permissions/selected-actions") {
		t.Fatalf("formatted error missing route: %s", formatted)
	}
	if !strings.Contains(formatted, "Status: 409 Conflict") {
		t.Fatalf("formatted error missing status: %s", formatted)
	}
	if strings.Contains(formatted, "GoError:") {
		t.Fatalf("formatted error still contains GoError wrapper: %s", formatted)
	}
	if strings.Contains(formatted, "execute exported function") {
		t.Fatalf("formatted error still contains execution wrapper: %s", formatted)
	}
}

func TestFormatCLIErrorForJavaScriptException(t *testing.T) {
	t.Parallel()

	vm := goja.New()
	_, err := vm.RunString(`throw new Error("boom")`)
	if err == nil {
		t.Fatal("expected JS execution to fail")
	}

	formatted := FormatCLIError(err)
	if !strings.Contains(formatted, "JavaScript execution failed") {
		t.Fatalf("formatted error missing heading: %s", formatted)
	}
	if !strings.Contains(formatted, "Message: Error: boom") {
		t.Fatalf("formatted error missing JS message: %s", formatted)
	}
}

func TestFormatCLIErrorForWrappedGoErrorString(t *testing.T) {
	t.Parallel()

	err := errors.New("execute exported function: GoError: runner output file path is empty at github.com/go-go-golems/goja-github-actions/pkg/modules/core.(*Module).setOutput-fm (native)")
	formatted := FormatCLIError(err)

	if !strings.Contains(formatted, "JavaScript execution failed") {
		t.Fatalf("formatted error missing heading: %s", formatted)
	}
	if !strings.Contains(formatted, "Message: runner output file path is empty") {
		t.Fatalf("formatted error missing cleaned message: %s", formatted)
	}
	if strings.Contains(formatted, "(native)") {
		t.Fatalf("formatted error still contains native stack marker: %s", formatted)
	}
}
