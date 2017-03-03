package concurrent

import (
	"context"
	"sync"
	"time"

	"github.com/go-akka/concurrent/internal"
)

type FixedRoutinePool struct {
	nRoutine int

	taskBackup []RunnableFuture

	statusLocker sync.Mutex
	initOnce     sync.Once

	dispatcher *internal.Dispatcher

	shutdownChan   chan struct{}
	terminatedChan chan struct{}
	isShutdownNow  bool
}

func NewFixedRoutinePool(nRoutine, queueSize int) *FixedRoutinePool {
	pool := &FixedRoutinePool{
		nRoutine:       nRoutine,
		dispatcher:     internal.NewDispatcher(nRoutine, queueSize),
		shutdownChan:   make(chan struct{}),
		terminatedChan: make(chan struct{}),
	}

	return pool
}

func (p *FixedRoutinePool) AwaitTermination(timeout time.Duration) (terminated bool) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	select {
	case <-ctx.Done():
		return false
	case <-p.terminatedChan:
		{
			return true
		}
	}
}

func (p *FixedRoutinePool) InvokeAll(tasks []interface{}) (future []Future, err error) {
	return p.InvokeAllDuration(tasks, -1)
}

func (p *FixedRoutinePool) InvokeAllDuration(tasks []interface{}, timeout time.Duration) (futures []Future, err error) {
	if p.IsShutdown() {
		return
	}

	var fs []Future

	// try cancel unprocessed tasks while error raise
	defer func() {
		if err != nil {
			if len(fs) > 0 {
				for i := 0; i < len(fs); i++ {
					fs[i].Cancel(true)
				}
			}
		}
	}()

	wg := &sync.WaitGroup{}

	for i := 0; i < len(tasks); i++ {

		var future Future
		var e error

		if future, e = p.Submit(tasks[i]); e != nil {
			err = e
			break
		}

		fs = append(fs, future)
		wg.Add(1)

		ctx := context.Background()

		if timeout > 0 {
			ctx, _ = context.WithTimeout(ctx, timeout)
		}

		go func(future Future, ctx context.Context) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					if !future.IsDone() {
						future.Cancel(true)
					}
					return
				default:
					if future.IsDone() || future.IsCancelled() {
						return
					}

					if p.isShutdownNow {
						return
					}
				}
				time.Sleep(time.Millisecond * 100)
			}
		}(fs[i], ctx)
	}

	wg.Wait()
	futures = fs

	p.clearBackup()

	return
}

func (p *FixedRoutinePool) InvokeAny(tasks []interface{}) (future []Future, err error) {
	return
}

func (p *FixedRoutinePool) InvokeAnyDuration(tasks []interface{}, timeout time.Duration) (futures []Future, err error) {
	return
}

func (p *FixedRoutinePool) IsShutdown() bool {
	select {
	case <-p.shutdownChan:
		{
			return true
		}
	default:
		return false
	}
	return false
}

func (p *FixedRoutinePool) IsTerminated() bool {
	select {
	case <-p.terminatedChan:
		{
			return true
		}
	default:
		return false
	}
	return false
}

func (p *FixedRoutinePool) Shutdown() (err error) {

	if p.IsShutdown() {
		return
	}

	p.statusLocker.Lock()
	defer p.statusLocker.Unlock()

	close(p.shutdownChan)

	go func() {
		totalCount := len(p.taskBackup)
		taskCount := totalCount
		for taskCount > 0 {
			taskCount = totalCount
			for i := 0; i < totalCount; i++ {
				if p.taskBackup[i].IsDone() || p.taskBackup[i].IsCancelled() {
					taskCount--
				}
			}
			time.Sleep(time.Millisecond * 100)
		}

		p.dispatcher.Stop()

		close(p.terminatedChan)
	}()

	p.clearBackup()

	return
}

func (p *FixedRoutinePool) ShutdownNow() (runnables []Runnable, err error) {

	if p.IsShutdown() {
		return
	}

	// p.statusLocker.Lock()
	// defer p.statusLocker.Unlock()
	close(p.shutdownChan)

	p.isShutdownNow = true

	p.dispatcher.Stop()

	var futures []RunnableFuture

bfor:
	for {
		select {
		case r := <-p.dispatcher.Queue():
			{
				f := r.(RunnableFuture)
				futures = append(futures, f)
			}
		default:
			break bfor
		}
	}

	for i := 0; i < len(futures); i++ {
		runnables = append(runnables, futures[i])
	}

	close(p.terminatedChan)

	p.clearBackup()

	return
}

func (p *FixedRoutinePool) Submit(fn interface{}) (future Future, err error) {
	if p.IsShutdown() {
		err = ErrRejectedExecution
		return
	}

	p.statusLocker.Lock()
	defer p.statusLocker.Unlock()

	f := NewFutureTask(fn, ExecutionContextOption(p))

	p.taskBackup = append(p.taskBackup, f)
	future = f

	return
}

func (p *FixedRoutinePool) Execute(runnable Runnable) {
	p.dispatcher.Submit(runnable)
}

func (p *FixedRoutinePool) ReportFailure(cause error) {}

func (p *FixedRoutinePool) clearBackup() {
	for i := 0; i < len(p.taskBackup); i++ {
		p.taskBackup[i] = nil
	}
	p.taskBackup = nil
}
