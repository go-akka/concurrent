package internal

type Dispatcher struct {
	workerPool chan *Worker
	queue      chan Runnable
	stop       chan bool
}

func NewDispatcher(workerCount int, queueSize int) *Dispatcher {
	d := &Dispatcher{
		workerPool: make(chan *Worker, workerCount),
		queue:      make(chan Runnable, queueSize),
		stop:       make(chan bool),
	}

	for i := 0; i < cap(d.workerPool); i++ {
		worker := NewWorker(d.workerPool)
		worker.Start()
	}

	go d.Dispatch()

	return d
}

func (p *Dispatcher) Queue() <-chan Runnable {
	return p.queue
}

func (p *Dispatcher) Submit(r Runnable) {
	p.queue <- r
}

func (p *Dispatcher) Dispatch() {
	for {
		select {
		case future := <-p.queue:
			worker := <-p.workerPool
			worker.runnableChan <- future
		case stop := <-p.stop:
			if stop {
				for i := 0; i < cap(p.workerPool); i++ {
					worker := <-p.workerPool

					worker.stop <- true
					<-worker.stop
				}

				p.stop <- true
				return
			}
		}
	}
}

func (p *Dispatcher) Stop() {
	p.stop <- true
	<-p.stop
}
