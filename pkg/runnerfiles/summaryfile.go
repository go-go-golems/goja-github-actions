package runnerfiles

import (
	"os"

	"github.com/pkg/errors"
)

type SummaryFile struct {
	Path string
}

func (f SummaryFile) Append(markdown string) error {
	if f.Path == "" {
		return errors.New("runner summary file path is empty")
	}
	return appendString(f.Path, normalizeMultilineValue(markdown))
}

func (f SummaryFile) Clear() error {
	if f.Path == "" {
		return errors.New("runner summary file path is empty")
	}
	if err := ensureParentDirectory(f.Path); err != nil {
		return err
	}
	return errors.Wrap(os.WriteFile(f.Path, []byte{}, 0o644), "clear summary file")
}
