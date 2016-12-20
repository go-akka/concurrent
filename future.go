package concurrent

import (
	"context"
	"reflect"
	"sync"
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

type FutureTask struct {
	statelock sync.Mutex

	fnCall interface{}
	result chan *Result

	cancel chan cancelTask

	cancelled bool
	done      bool
	running   bool
}

func NewFutureTask(fn interface{}) RunnableFuture {
	return &FutureTask{
		fnCall: fn,
		result: make(chan *Result, 1),
		cancel: make(chan cancelTask, 1),
	}
}

func (p *FutureTask) Get() (result *Result) {
	return p.getWithContext(context.Background())
}

func (p *FutureTask) GetDuration(timeout time.Duration) (result *Result) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return p.getWithContext(ctx)
}

func (p *FutureTask) IsDone() bool {
	return p.done
}

func (p *FutureTask) IsCancelled() bool {
	p.statelock.Lock()
	defer p.statelock.Unlock()
	return p.cancelled
}

func (p *FutureTask) Cancel(mayInterruptIfRunning bool) bool {
	p.statelock.Lock()
	defer p.statelock.Unlock()

	if p.cancelled {
		return true
	}

	if p.running {
		return false
	}

	p.cancel <- cancelTask{mayInterruptIfRunning: true}

	return true
}

func (p *FutureTask) Run() {
	p.statelock.Lock()
	if p.cancelled || p.done {
		p.statelock.Unlock()
		return
	}
	p.statelock.Unlock()

	select {
	case cancel := <-p.cancel:
		{
			if cancel.mayInterruptIfRunning {
				p.statelock.Lock()
				p.cancelled = true
				p.statelock.Unlock()
				return
			}
		}
	default:
	}

	p.statelock.Lock()
	p.running = true
	p.statelock.Unlock()

	var retVals []reflect.Value

	switch fn := p.fnCall.(type) {
	case Runnable:
		fn.Run()
	default:
		fnVal := reflect.ValueOf(fn)
		retVals = fnVal.Call(nil)
	}

	p.result <- &Result{values: retVals}

	p.statelock.Lock()
	p.done = true
	p.running = false
	p.statelock.Unlock()

	return
}

func (p *FutureTask) getWithContext(ctx context.Context) (result *Result) {

	if p.IsCancelled() {
		return
	}

	select {
	case <-ctx.Done():
		return nil
	case result = <-p.result:
		return
	}
}
