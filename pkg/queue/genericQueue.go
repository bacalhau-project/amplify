package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
)

var ErrQueueFull = errors.New("queue is full")
var ErrNotEnoughWorkers = errors.New("queue requires >= 1 workers")

type genericQueue struct {
	ctx        context.Context
	queue      chan func(context.Context)
	numWorkers int
	*sync.WaitGroup
}

func NewGenericQueue(ctx context.Context, numWorkers int, maxQueueSize int) (Queue, error) {
	if numWorkers < 1 {
		return nil, ErrNotEnoughWorkers
	}
	return &genericQueue{
		ctx:        ctx,
		queue:      make(chan func(context.Context), maxQueueSize),
		numWorkers: numWorkers,
		WaitGroup:  &sync.WaitGroup{},
	}, nil
}

func (q *genericQueue) Enqueue(w func(context.Context)) error {
	if len(q.queue) == cap(q.queue) {
		return ErrQueueFull
	}
	log.Ctx(q.ctx).Info().Msg("Enqueuing work.")
	q.queue <- w
	log.Ctx(q.ctx).Info().Msg("Finished enqueuing work.")
	return nil
}

func (q *genericQueue) Start() {
	for i := 0; i < q.numWorkers; i++ {
		go func() {
			defer q.Done()
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
		q.Add(1)
	}
}

func (q *genericQueue) Stop() {
	log.Ctx(q.ctx).Info().Msg("Waiting for workers to finish.")
	q.Wait()
	log.Ctx(q.ctx).Info().Msg("Finished waiting, exiting.")
}
