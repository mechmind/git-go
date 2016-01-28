package rawgit

import (
	"errors"
	"fmt"
)

type BoundError struct {
	boundTo Object
	err     error
}

func BindError(to Object, err error) BoundError {
	return BoundError{to, err}
}

func (be BoundError) Error() string {
	return fmt.Sprintf("[%s %s] %s", be.boundTo.OID(), be.boundTo.OType().String(), be.err.Error())
}

var (
	ERR_INVALID_REF        = errors.New("invalid reference")
	ERR_NOT_A_SYMBOLIC_REF = errors.New("not a symbolic reference")
)
