package wrapper

import (
	"bytes"
	"fmt"
	"io"
)

func WrapHTTPBody(body io.ReadCloser) (io.ReadCloser, error) {
	w, err := NewBodyReader(body)
	if err != nil {
		return nil, fmt.Errorf("can't create reader: %w", err)
	}
	if err = body.Close(); err != nil {
		return nil, fmt.Errorf("can't close original body: %w", err)
	}
	return w, nil
}

func NewBodyReader(r io.Reader) (*BodyReader, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("can't drain reader: %w", err)
	}
	br := &BodyReader{b: bytes.NewReader(buf)}
	return br, nil
}

type BodyReader struct {
	b *bytes.Reader
}

func (r *BodyReader) Read(b []byte) (int, error) {
	n, err := r.b.Read(b)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("can't read buffer: %w", err)
	}
	return n, err //nolint: wrapcheck // EOF or nil
}

func (r *BodyReader) Close() error {
	if _, err := r.b.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("can't seek: %w", err)
	}
	return nil
}
