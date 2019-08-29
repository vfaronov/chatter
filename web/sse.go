package web

import (
	"bytes"
	"io"
)

// dataWriter is an io.Writer that forwards writes to the underlying one, but
// inserts "data: " after each newline, to preserve text/event-stream framing.
type dataWriter struct {
	w io.Writer
}

// Write implements the io.Writer interface.
func (dw dataWriter) Write(p []byte) (n int, err error) {
	for {
		pos := bytes.IndexByte(p, '\n')
		if pos < 0 {
			break
		}
		nn, err := dw.w.Write(p[:pos+1])
		n += nn
		if err != nil {
			return n, err
		}
		p = p[pos+1:]
		_, err = dw.w.Write([]byte("data: "))
		if err != nil {
			return n, err
		}
	}
	nn, err := dw.w.Write(p)
	n += nn
	return n, err
}
