package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
)

var ErrQueueFull = errors.New("queue is full")
var ErrNotEnoughWorkers = errors.New("queue requires >= 1 workers")

type inMemQueue struct {
	ctx        context.Context
	ctxCancel  context.CancelFunc
	queue      chan func(context.Context)
	numWorkers int
	wg         *sync.WaitGroup
}

func NewGenericQueue(ctx context.Context, numWorkers int, maxQueueSize int) (Queue, error) {
	if numWorkers < 1 {
		return nil, ErrNotEnoughWorkers
	}
	ctx, cancel := context.WithCancel(ctx)
	return &inMemQueue{
		ctx:        ctx,
		ctxCancel:  cancel,
		queue:      make(chan func(context.Context), maxQueueSize),
		numWorkers: numWorkers,
		wg:         &sync.WaitGroup{},
	}, nil
}

func (q *inMemQueue) Enqueue(w func(context.Context)) error {
	if len(q.queue) == cap(q.queue) {
		return ErrQueueFull
	}
	q.queue <- w
	return nil
}

func (q *inMemQueue) Start() {
	for i := 0; i < q.numWorkers; i++ {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			for {
				select {
				case <-q.ctx.Done():
					log.Ctx(q.ctx).Info().Msg("Worker received quit command.")
					return
				case work := <-q.queue:
					work(q.ctx)
				}
			}
		}()
	}
}

func (q *inMemQueue) Stop() {
	log.Ctx(q.ctx).Info().Msg("Waiting for workers to finish.")
	q.ctxCancel()
	q.wg.Wait()
	log.Ctx(q.ctx).Info().Msg("Finished waiting, exiting.")
}
