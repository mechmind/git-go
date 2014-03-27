package git

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

const PACK_V2_MAGIC = -9154717 // decoded int value of '\377t0c'
var ErrInvalidPackVersion = errors.New("invalid pack version")
var ErrInvalidPackLength = errors.New("invalid pack length")
var ErrInvalidPackFileHeader = errors.New("invalid pack file header")
var ErrOffsetIdOutOfRange = errors.New("extended offset id is out of range")
var ErrInvalidDeltaBaseSize = errors.New("invalid base object size in delta")

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// packfile reader
type PackFile interface {
	Close() error
	OpenObjectAt(offset int64) (objInfo ObjectInfo, src io.Reader, err error)
}

// inMemoryPackFile represents packfile, entirely loaded in memory
// useful only for read-once packfiles (i.e. from archives)
type inMemoryPackFile struct {
	buf []byte
	count int32
	closed bool
}

func LoadPackFile(src io.Reader) (PackFile, error) {
	count, err := readPackFileHeader(src)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}

	inMemPack := &inMemoryPackFile{buf: content, count: count}
	return inMemPack, nil
}

func (p *inMemoryPackFile) Close() error {
	if p.closed {
		return errors.New("already closed")
	}

	p.buf = nil
	p.closed = true
	return nil
}

func (p *inMemoryPackFile) OpenObjectAt(offs int64) (info ObjectInfo, data io.Reader, err error) {
	reader := bytes.NewReader(p.buf[int(offs):])
	objType, objSize, err := readPackEntryHeader(reader)
	if err != nil {
		return
	}

	info.Type = objType
	info.Size = objSize
	return info, reader, nil
}

type seekablePackFile struct {
	storage ReadSeekCloser
	count int32
}

func OpenPackFile(src ReadSeekCloser) (PackFile, error) {
	count, err := readPackFileHeader(src)
	if err != nil {
		return nil, err
	}

	return &seekablePackFile{src, count}, nil
}

func (p *seekablePackFile) Close() error {
	return p.storage.Close()
}

func (p *seekablePackFile) OpenObjectAt(offs int64) (info ObjectInfo, data io.Reader, err error) {
	_, err = p.storage.Seek(offs, 0)
	objType, objSize, err := readPackEntryHeader(p.storage)
	if err != nil {
		return
	}

	return ObjectInfo{Type: objType, Size: objSize}, p.storage, nil
}

func readPackFileHeader(src io.Reader) (int32, error) {
	var sig [4]byte
	_, err := src.Read(sig[:])
	if sig != [4]byte{'P', 'A', 'C', 'K'} {
		return 0, ErrInvalidPackFileHeader
	}

	var version, count int32
	err = binary.Read(src, binary.BigEndian, &version)
	if err != nil {
		return 0, err
	}

	if version != 2 {
		return 0, ErrInvalidPackVersion
	}

	err = binary.Read(src, binary.BigEndian, &count)
	if err != nil {
		return 0, err
	}
	return count, err
}

func readPackEntryHeader(src io.Reader) (int8, uint64, error) {
	var buf = make([]byte, 1)
	// read first byte, with 3bit type
	_, err := src.Read(buf)
	if err != nil {
		return 0, 0, err
	}

	var objType int8 = int8((buf[0] >> 4) & 0x7)
	var size uint64 = uint64(buf[0] & 0xf)

	// while there is a 'more' bit, read next byte
	var shift uint = 4
	for buf[0] & 0x80 == 0x80 {
		_, err = src.Read(buf)
		if err != nil {
			return 0, 0, err
		}
		size += uint64(buf[0] & 0x7f) << shift
		shift += 7
	}

	return objType, size, nil
}

type IDXFile struct {
	offsets map[string]int64
}

func (i *IDXFile) LookupObject(hash string) int64 {
	offset, ok := i.offsets[hash]
	if !ok {
		return -1
	}
	return offset
}

func (i *IDXFile) LookupHash(offset int64) string {
	for hash, hashOffset := range i.offsets {
		if offset == hashOffset {
			return hash
		}
	}
	return ""
}

func ReadIDXFile(src io.Reader) (*IDXFile, error) {
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
	println("reading v1 index file")
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
	println("reading v2 index file")
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
	_, err = io.CopyN(ioutil.Discard, src, 255 * 4)
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
	pack PackFile
	idx *IDXFile
}

func OpenPack(idxFile, packFile io.ReadCloser) (*Pack, error) {
	idx, err := ReadIDXFile(idxFile)
	if err != nil {
		return nil, err
	}

	var pack PackFile
	if seekablePackFile, ok := packFile.(ReadSeekCloser); ok {
		pack, err = OpenPackFile(seekablePackFile)
		if err != nil {
			return nil, err
		}
	} else {
		pack, err = LoadPackFile(packFile)
		if err != nil {
			return nil, err
		}
	}
	return &Pack{pack, idx}, nil
}

func (p *Pack) HasObject(hash string) bool {
	return p.idx.LookupObject(hash) != -1
}

func (p *Pack) OpenObject(hash string) (ObjectInfo, io.ReadCloser, error) {
	offset := p.idx.LookupObject(hash)
	if offset == -1 {
		return ObjectInfo{}, nil, ErrObjectNotFound
	}

	return p.openObject(hash, offset)
}

func (p *Pack) OpenObjectAt(offset int64) (ObjectInfo, io.ReadCloser, error) {
	hash := p.idx.LookupHash(offset)
	if hash == "" {
		return ObjectInfo{}, nil, ErrObjectNotFound
	}

	return p.openObject(hash, offset)
}

func (p *Pack) openObject(hash string, offset int64) (ObjectInfo, io.ReadCloser, error) {
	info, data, err := p.pack.OpenObjectAt(offset)
	if err != nil {
		return ObjectInfo{}, nil, err
	}

	info.Hash = hash
	if info.Type == TYPE_REF_DELTA {
		var hashbuf = make([]byte, 20)
		_, err = data.Read(hashbuf)
		if err != nil {
			return ObjectInfo{}, nil, err
		}

		srcHash := bytes2hash(hashbuf)
		srcInfo, src, err := p.OpenObject(srcHash)
		if err != nil {
			return ObjectInfo{}, nil, err
		}

		info.Type = srcInfo.Type

		zlibReader, err := zlib.NewReader(data)
		if err != nil {
			return ObjectInfo{}, nil, err
		}

		return applyDelta(src, zlibReader, info)

	} else if info.Type == TYPE_OFS_DELTA {
		baseOffset, err := readVarInt(data)
		if err != nil {
			return ObjectInfo{}, nil, err
		}

		baseAddr := offset - baseOffset
		srcInfo, src, err := p.OpenObjectAt(baseAddr)
		if err != nil {
			return ObjectInfo{}, nil, err
		}

		info.Type = srcInfo.Type

		zlibReader, err := zlib.NewReader(data)
		if err != nil {
			return ObjectInfo{}, nil, err
		}
		return applyDelta(src, zlibReader, info)
	}

	zlibReader, err := zlib.NewReader(data)
	if err != nil {
		return ObjectInfo{}, nil, err
	}

	return info, newObjectReader(zlibReader, info.Size), nil
}

func applyDelta(src, delta io.ReadCloser, info ObjectInfo) (ObjectInfo, io.ReadCloser, error) {
	_, objSize, applier, err := newDeltaApplier(src, delta, int64(info.Size))
	if err != nil {
		return ObjectInfo{}, nil, err
	}
	info.Size = objSize
	return info, newObjectReader(applier, info.Size), nil
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
		num += int64(buf[0] & 0x7f) << shift
		shift += 7
		if (buf[0] & 0x80) > 0 {
			break
		}
	}
	return num, nil
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
