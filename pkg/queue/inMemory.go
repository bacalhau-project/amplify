package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// NewInMemoryQueue creates an in memory queue with numWorkers goroutines to
// process items in the queue.
func NewInMemoryQueue(ctx context.Context, numWorkers int) Queuer {
	return &dispatch{
		ctx:        ctx,
		queueLen:   0,
		queue:      make(chan *work),
		workChan:   make(chan *work),
		wg:         &sync.WaitGroup{},
		numWorkers: numWorkers,
	}
}

// Dispatch keeps track of an internal job request queue, a work queue of jobs
// that will be processed, a worker queue of workers, and a channel for status
// reports for jobs and workers.
type dispatch struct {
	ctx        context.Context
	queueLen   int             // TODO: Make this atomic so we can tell when jobs finish
	queue      chan *work      // internal queue
	workChan   chan *work      // used to pass work to workers
	wg         *sync.WaitGroup // used to wait for workers to finish before exiting
	numWorkers int             // number of workers to create
}

// Enqueue adds a job to the internal queue.
func (d *dispatch) Enqueue(store QueueRepository, id string, p Callable) error {
	// Create the work for the worker
	work := &work{
		ID:    id,
		Store: store,
		Job:   p,
	}

	// Add the job to the internal queue
	go func() { d.queue <- work }()

	// Increment the internal counter
	d.queueLen++
	log.Ctx(d.ctx).Info().Str("jobCounter", fmt.Sprint(d.queueLen)).Msg("Job Queued.")

	return nil
}

// Start creates the workers (goroutines) and starts processing the queue.
func (d *dispatch) Start() error {
	if d.numWorkers < 1 {
		return errors.New("start requires >= 1 workers")
	}

	// Create numWorkers:
	for i := 0; i < d.numWorkers; i++ {
		worker := createNewWorker(d.ctx, i, d.workChan, d.wg)
		worker.start()
		d.wg.Add(1)
	}

	// wait for work to be added then pass it off.
	go func() {
		for {
			select {
			case <-d.ctx.Done():
				log.Ctx(d.ctx).Info().Msg("Dispatcher received quit command.")
				return
			case item := <-d.queue:
				log.Ctx(d.ctx).Info().Str("ID", fmt.Sprint(item.ID)).Msg("Dispatching work to worker.")
				go func() {
					d.workChan <- item
				}()
			}
		}
	}()

	return nil
}

func (d *dispatch) Stop() error {
	log.Ctx(d.ctx).Info().Msg("Waiting for workers to finish.")
	d.wg.Wait()
	log.Ctx(d.ctx).Info().Msg("Finished waiting, exiting.")
	return nil
}
