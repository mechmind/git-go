package git

import (
	"io"
)

type DeltaApplier struct {
	limit int64
	src, delta io.ReadCloser
}

func newDeltaApplier(src, delta io.ReadCloser, size int64) (srcSize, objSize uint64, obj io.ReadCloser, err error) {
	srcSizeInt, err := readVarInt(delta)
	if err != nil {
		return 0, 0, nil, err
	}

	objSizeInt, err := readVarInt(delta)
	if err != nil {
		return 0, 0, nil, err
	}

	return uint64(srcSizeInt), uint64(objSizeInt), &DeltaApplier{size, src, delta}, nil
}

func (*DeltaApplier) Read(buf []byte) (int, error) {
	return 0, nil
}

func (*DeltaApplier) Close() error {
	return nil
}
