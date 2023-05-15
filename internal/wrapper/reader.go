package wrapper

import (
	"bytes"
	"fmt"
	"io"
)

func WrapHTTPBody(body io.ReadCloser) (io.ReadCloser, error) {
	w, err := NewBodyReader(body)
	if err != nil {
		return nil, fmt.Errorf("creating reader: %w", err)
	}
	if err = body.Close(); err != nil {
		return nil, fmt.Errorf("closing original body: %w", err)
	}
	return w, nil
}

func NewBodyReader(r io.Reader) (*BodyReader, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("draining reader: %w", err)
	}
	br := &BodyReader{b: bytes.NewReader(buf)}
	return br, nil
}

func NewBodyReaderFromRaw(data []byte) *BodyReader {
	return &BodyReader{b: bytes.NewReader(data)}
}

type BodyReader struct {
	b *bytes.Reader
}

func (r *BodyReader) Read(b []byte) (int, error) {
	n, err := r.b.Read(b)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("reading buffer: %w", err)
	}
	return n, err
}

func (r *BodyReader) Close() error {
	if _, err := r.b.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking: %w", err)
	}
	return nil
}
