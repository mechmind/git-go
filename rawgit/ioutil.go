package rawgit

import (
	"io"
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

func scanUntil(src io.Reader, needle byte, buf []byte) ([]byte, error) {
	buf = buf[:0]
	for i := 0; i < cap(buf)-1; i++ {
		_, err := src.Read(buf[i : i+1])
		if err != nil {
			return nil, err
		}
		if buf[i : i+1][0] == needle {
			return buf[:i], nil
		}
	}
	return nil, ErrBufferDepleted
}

func readVarInt(src io.Reader) (int64, error) {
	var num int64
	var buf = make([]byte, 1)
	var shift uint

	for {
		_, err := src.Read(buf)
		if err != nil {
			return 0, err
		}
		num += int64(buf[0]&0x7f) << shift
		shift += 7
		if (buf[0] & 0x80) == 0 {
			break
		}
	}
	return num, nil
}
