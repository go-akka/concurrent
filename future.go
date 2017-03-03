package concurrent

import (
	"time"
)

type FutureOptions struct {
	ExecutionContext ExecutionContext
}

type FutureOption func(*FutureOptions)

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

	OnComplete(fn interface{})
	AndThen(fn interface{}) Future
}

type RunnableFuture interface {
	Runnable
	Future
}

func ExecutionContextOption(ctx ExecutionContext) FutureOption {
	return func(p *FutureOptions) {
		p.ExecutionContext = ctx
	}
}
