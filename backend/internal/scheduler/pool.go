package scheduler

type Pool struct {
	jobs    chan func()
	stop    chan struct{}
	stopped bool
}

func NewPool(maxWorkers int) *Pool {
	p := &Pool{
		jobs: make(chan func(), 100),
		stop: make(chan struct{}),
	}
	for i := 0; i < maxWorkers; i++ {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	for {
		select {
		case <-p.stop:
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			job()
		}
	}
}

func (p *Pool) Submit(job func()) {
	if p.stopped {
		return
	}
	select {
	case p.jobs <- job:
	case <-p.stop:
	}
}

func (p *Pool) Stop() {
	if p.stopped {
		return
	}
	p.stopped = true
	close(p.stop)
}
