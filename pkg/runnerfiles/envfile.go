package runnerfiles

import (
	"fmt"

	"github.com/pkg/errors"
)

type EnvFile struct {
	Path string
}

func (f EnvFile) Set(name string, value string) error {
	if f.Path == "" {
		return errors.New("runner env file path is empty")
	}
	return appendString(f.Path, fmt.Sprintf("%s=%s\n", name, value))
}

func (f EnvFile) SetMultiline(name string, value string) error {
	if f.Path == "" {
		return errors.New("runner env file path is empty")
	}
	delimiter := "__GOJA_GHA_ENV__"
	return appendString(
		f.Path,
		fmt.Sprintf("%s<<%s\n%s\n%s\n", name, delimiter, normalizeMultilineValue(value), delimiter),
	)
}
