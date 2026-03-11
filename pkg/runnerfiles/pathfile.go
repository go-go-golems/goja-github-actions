package runnerfiles

import "github.com/pkg/errors"

type PathFile struct {
	Path string
}

func (f PathFile) Add(path string) error {
	if f.Path == "" {
		return errors.New("runner path file path is empty")
	}
	return appendString(f.Path, path+"\n")
}
