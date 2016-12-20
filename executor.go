package concurrent

import (
	"time"
)

type Executor interface {
	Execute(command interface{})
}

type ExecutorService interface {
	Executor

	AwaitTermination(timeout time.Duration) (terminated bool)

	InvokeAll(tasks []interface{}) (future []Future, err error)
	InvokeAllDuration(tasks []interface{}, timeout time.Duration) (future []Future, err error)

	InvokeAny(tasks []interface{}) (future []Future, err error)
	InvokeAnyDuration(tasks []interface{}, timeout time.Duration) (future []Future, err error)

	IsShutdown() bool
	IsTerminated() bool

	Shutdown() (err error)
	ShutdownNow() (runnables []Runnable, err error)

	Submit(task interface{}) (future Future, err error)
}
