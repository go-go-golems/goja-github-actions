package iomodule

import (
	stdio "io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type copyOptions struct {
	Force     bool `json:"force"`
	Recursive bool `json:"recursive"`
}

func (m *Module) readdir(path string) []string {
	entries, err := os.ReadDir(m.resolvePath(path))
	m.must(err)

	ret := make([]string, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, entry.Name())
	}
	return sortedEntries(ret)
}

func (m *Module) readFile(path string) string {
	content, err := os.ReadFile(m.resolvePath(path))
	m.must(err)
	return string(content)
}

func (m *Module) writeFile(path string, content string) {
	resolved := m.resolvePath(path)
	m.must(os.MkdirAll(filepath.Dir(resolved), 0o755))
	m.must(os.WriteFile(resolved, []byte(content), 0o644))
}

func (m *Module) mkdirP(path string) {
	m.must(os.MkdirAll(m.resolvePath(path), 0o755))
}

func (m *Module) rmRF(path string) {
	m.must(os.RemoveAll(m.resolvePath(path)))
}

func (m *Module) copy(source string, destination string, rawOptions map[string]interface{}) {
	options := copyOptions{}
	if rawOptions != nil {
		m.must(m.vm.ExportTo(m.vm.ToValue(rawOptions), &options))
	}
	if !options.Recursive {
		options.Recursive = true
	}

	m.must(copyPath(m.resolvePath(source), m.resolvePath(destination), options))
}

func (m *Module) move(source string, destination string) {
	from := m.resolvePath(source)
	to := m.resolvePath(destination)
	m.must(os.MkdirAll(filepath.Dir(to), 0o755))
	m.must(os.Rename(from, to))
}

func (m *Module) which(tool string, check bool) string {
	path, err := lookupPath(tool)
	if err != nil {
		if check {
			m.must(err)
		}
		return ""
	}
	return path
}

func copyPath(source string, destination string, options copyOptions) error {
	info, err := os.Stat(source)
	if err != nil {
		return errors.Wrapf(err, "stat source %s", source)
	}

	if info.IsDir() {
		if !options.Recursive {
			return errors.Errorf("source %s is a directory and recursive copy is disabled", source)
		}
		return copyDirectory(source, destination)
	}
	return copyFile(source, destination)
}

func copyDirectory(source string, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return errors.Wrapf(err, "compute relative path for %s", path)
		}
		target := filepath.Join(destination, relative)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}

func copyFile(source string, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return errors.Wrapf(err, "open source file %s", source)
	}
	defer func() {
		_ = in.Close()
	}()

	info, err := in.Stat()
	if err != nil {
		return errors.Wrapf(err, "stat source file %s", source)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return errors.Wrapf(err, "create destination directory for %s", destination)
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return errors.Wrapf(err, "open destination file %s", destination)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := stdio.Copy(out, in); err != nil {
		return errors.Wrapf(err, "copy %s to %s", source, destination)
	}
	return nil
}
