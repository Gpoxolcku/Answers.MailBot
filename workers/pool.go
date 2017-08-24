package workers

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	ErrJobTimedOut = errors.New("job request timed out")
)

type Func func() Result

type Task struct {
	f Func

	wg     sync.WaitGroup
	result Result
}

type Pool struct {
	concurrency int
	tasksChan   chan *Task
	wg          sync.WaitGroup
}

type Result struct {
	Response string
	Code     int
}

func (p *Pool) Size() int {
	return p.concurrency
}

func NewPool(concurrency int) *Pool {
	return &Pool{
		concurrency: concurrency,
		tasksChan:   make(chan *Task),
	}
}

func (p *Pool) Run() {
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.runWorker()
	}
}

func (p *Pool) Stop() {
	close(p.tasksChan)
	p.wg.Wait()
}

func (p *Pool) AddTaskSync(f Func) Result {
	t := Task{
		f:  f,
		wg: sync.WaitGroup{},
	}

	t.wg.Add(1)
	p.tasksChan <- &t
	t.wg.Wait()

	return t.result
}

func (p *Pool) AddTaskSyncTimed(f Func, timeout time.Duration) (Result, error) {
	t := Task{
		f:  f,
		wg: sync.WaitGroup{},
	}

	t.wg.Add(1)
	select {
	case p.tasksChan <- &t:
		break
	case <-time.After(timeout):
		r := Result{"", http.StatusInternalServerError}
		return r, ErrJobTimedOut
	}

	t.wg.Wait()

	return t.result, nil
}

func (p *Pool) runWorker() {
	for t := range p.tasksChan {
		t.result = t.f()
		t.wg.Done()
	}
	p.wg.Done()
}
