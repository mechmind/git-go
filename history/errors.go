package history

import (
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrTooFewRoots = errors.New("too few roots")
