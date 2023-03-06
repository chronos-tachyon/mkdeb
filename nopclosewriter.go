package mkdeb

import (
	"io"
)

type nopCloseWriter struct {
	file io.Writer
}

func (ncw nopCloseWriter) Write(p []byte) (int, error) {
	return ncw.file.Write(p)
}

func (ncw nopCloseWriter) Close() error {
	return nil
}
