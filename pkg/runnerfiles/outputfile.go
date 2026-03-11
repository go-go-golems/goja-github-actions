package runnerfiles

import (
	"fmt"

	"github.com/pkg/errors"
)

type OutputFile struct {
	Path string
}

func (f OutputFile) Set(name string, value string) error {
	if f.Path == "" {
		return errors.New("runner output file path is empty")
	}
	return appendString(f.Path, fmt.Sprintf("%s=%s\n", name, value))
}

func (f OutputFile) SetMultiline(name string, value string) error {
	if f.Path == "" {
		return errors.New("runner output file path is empty")
	}
	delimiter := "__GOJA_GHA_OUTPUT__"
	return appendString(
		f.Path,
		fmt.Sprintf("%s<<%s\n%s\n%s\n", name, delimiter, normalizeMultilineValue(value), delimiter),
	)
}
