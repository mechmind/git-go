package fsstor

import (
	"errors"
)

var (
	ErrInvalidDeltaOpcode = errors.New("invalid delta opcode")
	ErrNotFound           = errors.New("not found")
	ErrNotASymbolicRef    = errors.New("not a symbolic reference")
	ErrObjectOverflow     = errors.New("object size overflow")
	ErrIncompletedObject  = errors.New("object was not fully written")
)
var ErrBufferDepleted = errors.New("buffer depleted")
var ErrAlreadyClosed = errors.New("already closed")
var ErrInvalidPackVersion = errors.New("invalid pack version")
var ErrInvalidPackLength = errors.New("invalid pack length")
var ErrInvalidPackFileHeader = errors.New("invalid pack file header")
var ErrOffsetIdOutOfRange = errors.New("extended offset id is out of range")
var ErrInvalidDeltaBaseSize = errors.New("invalid base object size in delta")
var ErrObjectNotFound = errors.New("object not found")
var ErrInvalidObjectType = errors.New("invalid object type")
