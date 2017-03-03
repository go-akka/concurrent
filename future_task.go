package concurrent

import (
	"context"
	"reflect"
	"sync/atomic"
	"time"
)

type FutureTask struct {
	fnCall interface{}
	result chan *Result

	cancel chan cancelTask

	cancelled int32
	done      int32
	running   int32

	opts *FutureOptions

	callbacks []interface{}

	then      *FutureTask
	topFuture *FutureTask
}

func NewFutureTask(fn interface{}, opts ...FutureOption) RunnableFuture {
	future := &FutureTask{
		fnCall: fn,
		opts:   &FutureOptions{ExecutionContext: GlobalExecutionContext},
		result: make(chan *Result, 1),
		cancel: make(chan cancelTask, 1),
	}

	for i := 0; i < len(opts); i++ {
		opts[i](future.opts)
	}

	if future.opts.ExecutionContext == nil {
		panic("ExecutionContext is nil, you could import _ \"github.com/go-akka/concurrent/global\" for use global ExecutionContext or define your own ExecutionContext by concurrent.ExecutionContextOption")
	}

	future.opts.ExecutionContext.Execute(future)

	return future
}

func (p *FutureTask) Get() (result *Result) {
	return p.getWithContext(context.Background())
}

func (p *FutureTask) GetDuration(timeout time.Duration) (result *Result) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return p.getWithContext(ctx)
}

func (p *FutureTask) IsDone() (done bool) {
	return p.done > 0
}

func (p *FutureTask) IsCancelled() (cancelled bool) {
	return p.cancelled > 0
}

func (p *FutureTask) isRunning() (running bool) {
	return p.running > 0
}

func (p *FutureTask) Cancel(mayInterruptIfRunning bool) bool {

	if p.IsCancelled() {
		return true
	}

	if p.isRunning() {
		return false
	}

	p.cancel <- cancelTask{mayInterruptIfRunning: true}

	return true
}

func (p *FutureTask) Run() {

	if p.IsCancelled() || p.IsDone() || p.isRunning() {
		return
	}

	select {
	case cancel := <-p.cancel:
		{
			if cancel.mayInterruptIfRunning {
				atomic.CompareAndSwapInt32(&p.cancelled, 0, 1)
				return
			}
		}
	default:
	}

	atomic.CompareAndSwapInt32(&p.running, 0, 1)

	var retVals []reflect.Value

	switch fn := p.fnCall.(type) {
	case Runnable:
		fn.Run()
	default:
		fnVal := reflect.ValueOf(fn)
		retVals = fnVal.Call(nil)
	}

	result := &Result{values: retVals}
	p.result <- result

	atomic.CompareAndSwapInt32(&p.done, 0, 1)
	atomic.CompareAndSwapInt32(&p.running, 1, 0)

	p.executeCallback(result)

	if p.then != nil {
		p.opts.ExecutionContext.Execute(p.then)
	}

	return
}

func (p *FutureTask) OnComplete(fn interface{}) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		panic("OnComplete params should be a func")
	}
	p.callbacks = append(p.callbacks, fn)
}

func (p *FutureTask) AndThen(fn interface{}) Future {

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		panic("AndThen params should be a func")
	}

	var topFuture *FutureTask
	if p.topFuture != nil {
		topFuture = p.topFuture
	} else {
		topFuture = p
	}

	future := &FutureTask{
		topFuture: topFuture,
		opts:      &FutureOptions{ExecutionContext: GlobalExecutionContext},
		result:    make(chan *Result, 1),
		cancel:    make(chan cancelTask, 1),
	}

	future.fnCall = func() {
		if err := topFuture.Get().V(fn); err != nil {
			future.opts.ExecutionContext.ReportFailure(err)
		}
	}

	p.then = future
	return future
}

func (p *FutureTask) executeCallback(v *Result) {
	for i := 0; i < len(p.callbacks); i++ {
		if cause := v.V(p.callbacks[i]); cause != nil {
			p.opts.ExecutionContext.ReportFailure(cause)
		}
	}
}

func (p *FutureTask) getWithContext(ctx context.Context) (result *Result) {

	if p.IsCancelled() {
		return
	}

	select {
	case <-ctx.Done():
		return nil
	case result = <-p.result:
		p.result <- result
		return
	}
}
