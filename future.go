package concurrent

import (
	"time"
)

type cancelTask struct {
	mayInterruptIfRunning bool
}

type Runnable interface {
	Run()
}

type Future interface {
	Get() (result *Result)
	GetDuration(dur time.Duration) (result *Result)
	IsDone() bool
	IsCancelled() bool
	Cancel(mayInterruptIfRunning bool) bool
}

type RunnableFuture interface {
	Runnable
	Future
}
