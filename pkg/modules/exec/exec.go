package execmodule

import (
	"bytes"
	"io"
	"sort"
)

type bufferWriter struct {
	bytes.Buffer
}

func newBufferWriter() *bufferWriter {
	return &bufferWriter{}
}

func combineWriters(writers ...io.Writer) io.Writer {
	active := make([]io.Writer, 0, len(writers))
	for _, writer := range writers {
		if writer != nil {
			active = append(active, writer)
		}
	}
	return io.MultiWriter(active...)
}

func optionalWriter(enabled bool, writer io.Writer) io.Writer {
	if !enabled {
		return nil
	}
	return writer
}

func environmentSlice(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	ret := make([]string, 0, len(keys))
	for _, key := range keys {
		ret = append(ret, key+"="+env[key])
	}
	return ret
}
