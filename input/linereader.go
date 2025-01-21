package input

import (
	"bytes"
	"io"
)

type Reader struct {
	file *FileState
	reader io.Reader
	buf []byte
	bufferedLine string
}

func NewReader(file *FileState, reader io.Reader) *Reader {
	return &Reader{file: file, reader: reader, buf: make([]byte, 0, 1024)}
}

func (r *Reader) PeekString(delim string) (string, error) {
	if delim == "" {
		return "", nil
	}
	delimByte := []byte(delim)
	buf := make([]byte, 1)
	for {
		n, err := r.reader.Read(buf)
		if n == 0 {
			return "", err
		}
		r.buf = append(r.buf, buf[0])
		if bytes.HasSuffix(r.buf, delimByte) {
			str := string(r.buf)
			r.buf = r.buf[:0]
			r.bufferedLine += str
			return str, nil
		}
	}
}

func (r *Reader) ReadString(delim string) (string, error) {
	_, err := r.PeekString(delim)
	if err != nil {
		return "", err
	}
	line := r.bufferedLine
	r.bufferedLine = ""
	r.file.Offset += int64(len(line))
	return line, nil
}
