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
	ErrNotASymbolicRef = errors.New("not a symbolic reference")
	ErrNotACommit      = errors.New("not a commit object")
	ErrNotATree        = errors.New("not a tree object")
	ErrInvalidPath     = errors.New("invalid path")
	ErrNotFound        = errors.New("not found")
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

var ErrObjectNotFound = errors.New("object not found")

var ErrInvalidObjectType = errors.New("invalid object type")

var ErrBufferDepleted = errors.New("buffer depleted")

var ErrAlreadyClosed = errors.New("already closed")

var ErrInvalidRef = errors.New("invalid ref")
