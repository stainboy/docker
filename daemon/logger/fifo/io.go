package fifo

import (
	"io"
)

func mkfifo(file string, delayDeletion bool) (io.ReadWriteCloser, error) {
	// TODO:
	return nil, nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopWriterCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopWriterCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}
