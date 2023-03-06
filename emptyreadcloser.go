package mkdeb

import (
	"io"
)

type emptyReadCloser struct{}

func (emptyReadCloser) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (emptyReadCloser) Close() error {
	return nil
}
