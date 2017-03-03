package global

import (
	"github.com/go-akka/concurrent"
)

func init() {
	concurrent.GlobalExecutionContext = concurrent.NewFixedRoutinePool(10, 10)
}
