package concurrent

import (
	"context"
	"sync"
	"time"
)

type FixedRoutinePool struct {
	nRoutine int

	task chan RunnableFuture

	taskStack  *RunnableFutureStack
	taskBackup []RunnableFuture

	statusLocker sync.Mutex
	initOnce     sync.Once

	dispatcherDown chan struct{}
	workerDown     map[int]chan struct{}
	shutdownChan   chan struct{}
	terminatedChan chan struct{}
	isShutdownNow  bool
}

func NewFixedRoutinePool(nRoutine int) ExecutorService {
	pool := &FixedRoutinePool{
		nRoutine:       nRoutine,
		taskStack:      NewRunnableFutureStack(),
		workerDown:     make(map[int]chan struct{}, nRoutine),
		task:           make(chan RunnableFuture),
		shutdownChan:   make(chan struct{}),
		terminatedChan: make(chan struct{}),
	}

	pool.init()

	return pool
}

func (p *FixedRoutinePool) init() {
	p.initOnce.Do(func() {
		p.beginWorkers()
		p.beginDispatcher()
	})
}

func (p *FixedRoutinePool) beginWorkers() {

	for i := 0; i < p.nRoutine; i++ {
		downchan := make(chan struct{})
		p.workerDown[i] = downchan
		go p.handler(i, downchan)
	}
}

func (p *FixedRoutinePool) beginDispatcher() {

	p.dispatcherDown = make(chan struct{})

	go func() {
		for {
			select {
			case <-p.dispatcherDown:
				return
			default:
			}

			var task RunnableFuture
			task = p.taskStack.Pop()
			if task != nil {
				p.task <- task
			}

		}
	}()
}

func (p *FixedRoutinePool) handler(id int, shutdownChan chan struct{}) {
	for {
		select {
		case <-shutdownChan:
			return
		case task, ok := <-p.task:
			{
				if !ok {
					return
				}

				if p.isShutdownNow {
					task.Cancel(true)
					task.Run()
					continue
				}

				task.Run()
			}
		}
	}
}

func (p *FixedRoutinePool) AwaitTermination(timeout time.Duration) (terminated bool) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	select {
	case <-ctx.Done():
		return false
	case <-p.shutdownChan:
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
		if future, err = p.Submit(tasks[i]); err != nil {
			return
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
					if future.IsDone() {
						return
					}

					if p.isShutdownNow {
						return
					}
				}
				time.Sleep(time.Millisecond * 100)
			}
		}(future, ctx)
	}

	wg.Wait()
	futures = fs

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
	p.statusLocker.Lock()
	defer p.statusLocker.Unlock()

	if p.IsShutdown() {
		return
	}

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
		close(p.shutdownChan)
		for _, c := range p.workerDown {
			close(c)
		}
		close(p.terminatedChan)
		close(p.dispatcherDown)
	}()

	return
}

func (p *FixedRoutinePool) ShutdownNow() (runnables []Runnable, err error) {
	p.statusLocker.Lock()
	defer p.statusLocker.Unlock()

	if p.IsShutdown() {
		return
	}

	p.isShutdownNow = true

	futures := p.taskStack.PopAll()
	close(p.dispatcherDown)

	for _, c := range p.workerDown {
		close(c)
	}

bfor:
	for {
		select {
		case f, ok := <-p.task:
			{
				if !ok {
					break
				}
				futures = append(futures, f)
			}
		default:
			break bfor
		}
	}

	close(p.task)

	for i := 0; i < len(futures); i++ {
		runnables = append(runnables, futures[i])
	}

	close(p.shutdownChan)
	close(p.terminatedChan)

	return
}

func (p *FixedRoutinePool) Submit(fn interface{}) (future Future, err error) {
	p.statusLocker.Lock()
	defer p.statusLocker.Unlock()

	if p.IsShutdown() {
		err = ErrRejectedExecution
		return
	}

	f := NewFutureTask(fn)
	p.taskStack.Push(f)
	p.taskBackup = append(p.taskBackup, f)

	future = f

	return
}

func (p *FixedRoutinePool) Execute(command interface{}) {
	p.Submit(command)
}
