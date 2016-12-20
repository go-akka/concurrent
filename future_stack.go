package concurrent

import (
	"sync"
)

type RunnableFutureStack struct {
	lock   sync.Mutex
	values []RunnableFuture
}

func NewRunnableFutureStack() *RunnableFutureStack {
	return &RunnableFutureStack{sync.Mutex{}, make([]RunnableFuture, 0)}
}

func (p *RunnableFutureStack) Push(v RunnableFuture) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.values = append(p.values, v)
}

func (p *RunnableFutureStack) Pop() RunnableFuture {
	p.lock.Lock()
	defer p.lock.Unlock()

	l := len(p.values)
	if l == 0 {
		return nil
	}

	res := p.values[l-1]
	p.values = p.values[:l-1]
	return res
}

func (p *RunnableFutureStack) PopAll() []RunnableFuture {
	p.lock.Lock()
	defer p.lock.Unlock()

	l := len(p.values)
	if l == 0 {
		return nil
	}

	res := p.values[0:]
	p.values = nil
	return res
}

func (p *RunnableFutureStack) Length() int {
	p.lock.Lock()
	defer p.lock.Unlock()

	return len(p.values)
}
