package concurrent

import (
	"errors"
)

var (
	ErrRejectedExecution = errors.New("rejected execution task")
)
