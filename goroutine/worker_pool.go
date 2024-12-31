package goroutine

import (
	"sync"
	"time"
)

var defaultWorkerPool = NewWorkerPool(1024, time.Minute*2)

type GoOptions struct {
	PanicCallack func(err interface{})
}

type GoOption func(*GoOptions)

func WithPanicCallback(cb func(interface{})) GoOption {
	return func(o *GoOptions) {
		o.PanicCallack = cb
	}
}

func NewWorkerPool(size int, maxIdle time.Duration) *WorkerPool {
	return &WorkerPool{
		Size:            size,
		MaxIdelDuration: maxIdle,
	}
}

type WorkerPool struct {
	sync.RWMutex

	Size            int
	MaxIdelDuration time.Duration

	idleSize int
	idle     *worker

	busySize int
	busy     *worker
}

func (p *WorkerPool) IdleSize() (size int) {
	p.RLock()
	size = p.idleSize
	p.RUnlock()
	return
}

func (p *WorkerPool) BusySize() (size int) {
	p.RLock()
	size = p.busySize
	p.RUnlock()
	return
}

func (p *WorkerPool) Go(task func(), opts ...GoOption) {
	p.Lock()
	defer p.Unlock()

	j := newJob(task, opts...)

	if p.idleSize > 0 {
		// 1. remove from idle head
		w := p.idle
		p.idle = w.next
		if p.idle != nil {
			p.idle.prev = nil
		}
		p.idleSize = p.idleSize - 1

		// 2. add to busy head
		w.next = p.busy

		if w.next != nil {
			w.next.prev = w
		}

		p.busySize = p.busySize + 1

		w.Go(j)
		return
	}

	if p.busySize == p.Size {
		go func(j *job) {
			defer func() {
				if err := recover(); err != nil {
					if j.opts != nil && j.opts.PanicCallack != nil {
						j.opts.PanicCallack(err)
					}
				}

				j.task()
			}()
		}(j)
		return
	}

	w := newWorker(p)
	go w.Start()
	w.next = p.busy
	if w.next != nil {
		w.next.prev = w
	}
	p.busySize = p.busySize + 1
	w.Go(j)
}

func (p *WorkerPool) removeIdle(w *worker) {
	p.Lock()
	defer p.Unlock()

	prev := w.prev
	next := w.next

	if prev != nil {
		prev.next = next
	}

	if next != nil {
		next.prev = prev
	}
	// TODO: use sync.Pool
	p.idleSize = p.idleSize - 1
	w.prev = nil
	w.next = nil
	w = nil
}

func (p *WorkerPool) appendIdle(w *worker) {
	p.Lock()
	defer p.Unlock()

	// 1. remove from busy
	prev := w.prev
	next := w.next

	if prev != nil {
		prev.next = next
	}

	if next != nil {
		next.prev = prev
	}

	p.busySize = p.busySize - 1

	// 2. add to idle
	w.next = p.idle
	if w.next != nil {
		w.next.prev = w
	}

	p.idle = w
	p.idleSize = p.idleSize + 1
}

type job struct {
	task func()
	opts *GoOptions
}

func newJob(task func(), opts ...GoOption) *job {
	var gopts GoOptions

	for _, opt := range opts {
		opt(&gopts)
	}

	return &job{
		task: task,
		opts: &gopts,
	}
}

type worker struct {
	pool *WorkerPool

	job  chan *job
	prev *worker
	next *worker
}

func newWorker(pool *WorkerPool) *worker {
	return &worker{
		pool: pool,
		job:  make(chan *job, 1),
	}
}

func (w *worker) Go(job *job) {
	w.job <- job
}

func (w *worker) Start() {
	run := func(j *job) {
		defer func() {
			if err := recover(); err != nil {
				if j.opts != nil && j.opts.PanicCallack != nil {
					j.opts.PanicCallack(err)
				}
			}
		}()
		j.task()
	}

	for {
		t := time.NewTimer(w.pool.MaxIdelDuration)
		select {
		case <-t.C:
			w.quit()
			return
		case j := <-w.job:
			t.Stop()
			run(j)
			w.idle()
		}
	}
}

func (w *worker) quit() {
	w.pool.removeIdle(w)
}

func (w *worker) idle() {
	w.pool.appendIdle(w)
}
