package git

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

const PACK_V2_MAGIC = -9162653 // decoded int value of '\377t0c'
var ErrInvalidPackVersion = errors.New("invalid pack version")
var ErrInvalidPackLength = errors.New("invalid pack length")
var ErrOffsetIdOutOfRange = errors.New("extended offset id is out of range")

// for now, just keep entire index in memory

// packfile reader
type PackFile struct{
	storage io.ReadSeeker
}

type IDXFile struct {
	offsets map[string]int64
}

func (i *IDXFile) LookupObject(hash string) int64 {return 0}

func ReadIDXFile(src io.ReadSeeker) (*IDXFile, error) {
	var magic int32

	// read first value
	err := binary.Read(src, binary.BigEndian, &magic)
	if err != nil {
		return nil, err
	}

	if magic == PACK_V2_MAGIC {
		// this is v2 or greater pack index
		return readV2IDXFile(src)
	} else {
		// this is v1 pack index
		return readV1IDXFile(src)
	}
}

func readV1IDXFile(src io.Reader) (*IDXFile, error) {
	idx := &IDXFile{}

	// skip to last (256) fanout entry
	_, err := io.CopyN(ioutil.Discard, src, 254 * 4)
	if err != nil {
		return nil, err
	}

	var total int32
	// read last fanout entry (= total objects)
	err = binary.Read(src, binary.BigEndian, &total)
	if err != nil {
		return nil, err
	}

	idx.offsets = make(map[string]int64, total)

	var offset int32
	var hash = make([]byte, 20)

	var i int32
	for i = 0; i < total; i++ {
		// read offset table
		// read offset, then object name
		err = binary.Read(src, binary.BigEndian, &offset)
		if err != nil {
			return nil, err
		}

		_, err = src.Read(hash)
		if err != nil {
			return nil, err
		}
		idx.offsets[bytes2hash(hash)] = int64(offset)
	}

	return idx, nil
}

func readV2IDXFile(src io.Reader) (*IDXFile, error) {
	var idx = &IDXFile{}

	var version int32
	err := binary.Read(src, binary.BigEndian, &version)
	if err != nil {
		return nil, err
	}

	if version != 2 {
		return nil, ErrInvalidPackVersion
	}

	// skip to last (256) fanout entry
	_, err = io.CopyN(ioutil.Discard, src, 254 * 4)
	if err != nil {
		return nil, err
	}

	var total int32
	// read last fanout entry (= total objects)
	err = binary.Read(src, binary.BigEndian, &total)
	if err != nil {
		return nil, err
	}

	// read all hashes
	allHashes := make([]byte, int(total) * 20)
	_, err = src.Read(allHashes)
	if err != nil {
		return nil, err
	}

	// skip all crc32 sums
	io.CopyN(ioutil.Discard, src, int64(total) * 4)

	// read all primary offsets
	primaryOffsets := make([]int32, int(total))
	err = binary.Read(src, binary.BigEndian, primaryOffsets)
	if err != nil {
		return nil, err
	}

	// read extended offset table and checksums
	trailer, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}

	if len(trailer) < 40 {
		return nil, ErrInvalidPackLength
	}

	extOffsetBuf := trailer[:len(trailer) - 40]
	extOffsetCount := len(extOffsetBuf) / 8
	var extOffsets []int64

	if extOffsetCount > 0 {
		extOffsets = make([]int64, extOffsetCount)
		err = binary.Read(bytes.NewReader(extOffsetBuf), binary.BigEndian, extOffsets)
		if err != nil {
			return nil, err
		}
	}

	idx.offsets = make(map[string]int64, total)

	// load ids and offsets
	var i, offset4 int32
	var offset int64
	for i = 0; i < total; i++ {
		offset4 = primaryOffsets[i]
		if i < 0 {
			// it is an extended offset
			extOffsetId := int(offset & (1 << 31 - 1))
			if extOffsetId > len(extOffsets) {
				return nil, ErrOffsetIdOutOfRange
			}
			offset = extOffsets[extOffsetId]
		} else {
			offset = int64(offset4)
		}
		idx.offsets[bytes2hash(allHashes[i*20:(i+1)*20])] = offset
	}
	return idx, nil
}


type Pack struct {
	pack *PackFile
	idx *IDXFile
}

func hash2bytes(hash string) []byte {
	buf := make([]byte, 20)
	_, err := fmt.Sscanf(hash, "%x", &buf)
	if err != nil {
		return nil
	}
	return buf
}

func bytes2hash(bytes []byte) string {
	return fmt.Sprintf("%x", bytes)
}
