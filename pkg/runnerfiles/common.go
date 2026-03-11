package runnerfiles

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ensureParentDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("runner file path is empty")
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.Wrapf(err, "create parent directory for %s", path)
	}

	return nil
}

func appendString(path string, value string) error {
	if err := ensureParentDirectory(path); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Wrapf(err, "open runner file %s", path)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.WriteString(value); err != nil {
		return errors.Wrapf(err, "append runner file %s", path)
	}

	return nil
}

func normalizeMultilineValue(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\r", "\n")
}
