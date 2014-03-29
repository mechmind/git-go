package git

import (
	"io"
	"bytes"
	"errors"
	"io/ioutil"
)

var (
	ErrInvalidDeltaOpcode = errors.New("invalid delta opcode")
)

// Read delta headers and return Reader of target object.
// For now, just read apply delta immediately
func newDeltaApplier(src, delta io.ReadCloser, size int64) (srcSize, objSize uint64, obj io.ReadCloser, err error) {
	srcSizeInt, err := readVarInt(delta)
	if err != nil {
		return 0, 0, nil, err
	}

	objSizeInt, err := readVarInt(delta)
	if err != nil {
		return 0, 0, nil, err
	}

	srcBuf, err := ioutil.ReadAll(src)
	if err != nil {
		return 0, 0, nil, err
	}

	deltaBuf, err := ioutil.ReadAll(delta)
	if err != nil {
		return 0, 0, nil, err
	}

	objBuf := make([]byte, objSizeInt)
	err = applyDeltaBuf(srcBuf, deltaBuf, objBuf)
	if err != nil {
		return 0, 0, nil, err
	}

	return uint64(srcSizeInt), uint64(objSizeInt),
		ioutil.NopCloser(bytes.NewBuffer(objBuf)), nil
}

func applyDeltaBuf(src, delta, obj []byte) error {
	var pos int
	var c byte

	for pos < len(delta) {
		// read delta opcode
		c = delta[pos]
		pos++
		if c & 0x80 > 0 {
			// this is copy opcode
			// read offset
			var offset, size uint32
			if c & 0x01 > 0 {
				offset = uint32(delta[pos])
				pos++
			}

			if c & 0x02 > 0 {
				offset += uint32(delta[pos]) << 8
				pos++
			}

			if c & 0x04 > 0 {
				offset += uint32(delta[pos]) << 16
				pos++
			}

			if c & 0x08 > 0 {
				offset += uint32(delta[pos]) << 24
				pos++
			}

			// read size
			if c & 0x10 > 0 {
				size = uint32(delta[pos])
				pos++
			}

			if c & 0x20 > 0 {
				size += uint32(delta[pos]) << 8
				pos++
			}

			if c & 0x40 > 0 {
				size += uint32(delta[pos]) << 16
				pos++
			}

			if size == 0 {
				size = 0x10000
			}

			n := copy(obj, src[int(offset):int(offset+size)])
			obj = obj[n:]
		} else if c > 0 {
			n := copy(obj, delta[pos:pos+int(c)])
			obj = obj[n:]
			pos += int(c)
		} else {
			return ErrInvalidDeltaOpcode
		}
	}

	return nil
}
