package rawgit

import (
	"errors"
	"fmt"
)

type BoundError struct {
	boundTo ObjectInfo
	err     error
}

func BindError(to ObjectInfo, err error) BoundError {
	return BoundError{to, err}
}

func (be BoundError) Error() string {
	return fmt.Sprintf("[%s %s] %s", be.boundTo.GetOID(), be.boundTo.GetOType().String(), be.err.Error())
}

var (
	ERR_INVALID_REF        = errors.New("invalid reference")
	ERR_NOT_A_SYMBOLIC_REF = errors.New("not a symbolic reference")
)

var (
	ErrObjectOverflow    = errors.New("object size overflow")
	ErrIncompletedObject = errors.New("object was not fully written")
)

var (
	ErrInvalidHashLength = errors.New("invalid hash length")
	ErrNoTree            = errors.New("no proper 'tree' record in commit object")
	ErrNoAuthor          = errors.New("no proper 'author' record in commit object")
	ErrNoCommitter       = errors.New("no proper 'commiter' record in commit object")
	ErrInvalidEmail      = errors.New("no proper email field in record")
	ErrInvalidEncoding   = errors.New("malformed encoding record")
	ErrInvalidRecord     = errors.New("invalid record")
)

var (
	ErrInvalidDeltaOpcode = errors.New("invalid delta opcode")
)

var ErrObjectNotFound = errors.New("object not found")

var ErrInvalidPackVersion = errors.New("invalid pack version")
var ErrInvalidPackLength = errors.New("invalid pack length")
var ErrInvalidPackFileHeader = errors.New("invalid pack file header")
var ErrOffsetIdOutOfRange = errors.New("extended offset id is out of range")
var ErrInvalidDeltaBaseSize = errors.New("invalid base object size in delta")

var ErrInvalidObjectType = errors.New("invalid object type")

var ErrBufferDepleted = errors.New("buffer depleted")

var ErrAlreadyClosed = errors.New("already closed")
