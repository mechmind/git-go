package git

import (
	"errors"
	"io"
)

var (
	ErrObjectOverflow    = errors.New("object size overflow")
	ErrIncompletedObject = errors.New("object was not fully written")
)

type exactSizeWriter struct {
	bytesLeft uint64
	writer    io.Writer
}

func (e *exactSizeWriter) Write(data []byte) (n int, err error) {
	length := uint64(len(data))
	if length == 0 {
		return 0, nil
	}

	if length > e.bytesLeft {
		// FIXME: should be instant fail
		// write remaining chunk and return overflow error
		println("want to write ", length, " bytes, but left only ", e.bytesLeft)
		data = data[:int(e.bytesLeft)]
		n, err := e.writer.Write(data)
		e.bytesLeft -= uint64(n)
		if err != nil {
			return n, err
		} else {
			return n, ErrObjectOverflow
		}
	} else {
		n, err := e.writer.Write(data)
		e.bytesLeft -= uint64(n)
		return n, err
	}
}

func (e *exactSizeWriter) Close() error {
	if e.bytesLeft != 0 {
		return ErrIncompletedObject
	}
	return nil
}
