package rawgit

import (
	"io"
)

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
