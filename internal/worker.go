package internal

import (
	"sync"
)

type Runnable interface {
	Run()
}

type Worker struct {
	workerPool   chan<- *Worker
	runnableChan chan Runnable
	stop         chan bool

	sync.Mutex
	isStopped bool
}

func NewWorker(pool chan<- *Worker) *Worker {
	return &Worker{
		workerPool:   pool,
		runnableChan: make(chan Runnable),
		stop:         make(chan bool),
	}
}

func (p *Worker) Start() {
	go func() {
		var runnable Runnable
		var ok bool
		for {
			p.workerPool <- p
			select {
			case runnable, ok = <-p.runnableChan:
				if !ok {
					return
				}
				runnable.Run()
			case <-p.stop:
				p.stop <- true
				return
			}
		}
	}()
}

func (p *Worker) Stop() {
	p.stop <- true
	<-p.stop
}
