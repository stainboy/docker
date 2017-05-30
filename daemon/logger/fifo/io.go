package fifo

import (
	"io"
	"os"
	"syscall"
	"time"

	cfifo "github.com/containerd/fifo"
)

// mkfifo creates the named pipe and open it from writer side
func mkfifo(file string, delayDeletion bool) (io.WriteCloser, error) {

	var w io.WriteCloser

	f, err := cfifo.OpenFifo(nil, file, syscall.O_CREAT|syscall.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}

	w = f

	if delayDeletion {
		w = wrapCloser(w, func() error {
			go func() {
				time.Sleep(1 * time.Minute)
				os.RemoveAll(file)
			}()
			return nil
		})
	}

	return w, nil
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

type action func() error

type myCloser struct {
	io.WriteCloser
	fn action
}

func (c *myCloser) Close() error {
	if err := c.WriteCloser.Close(); err != nil {
		return err
	}
	return c.fn()
}

func wrapCloser(w io.WriteCloser, fn action) io.WriteCloser {
	return &myCloser{w, fn}
}
