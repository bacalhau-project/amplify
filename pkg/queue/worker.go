package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// work includes all the details the worker needs to process a job
type work struct {
	ID    string
	Store QueueRepository
	Job   Callable
}

// A worker is a goroutine that accepts jobs from a channel and processes them.
type worker struct {
	ID   int
	jobs chan *work
	ctx  context.Context
	wg   *sync.WaitGroup
}

// createNewWorker creates a new worker with the given ID and job queue.
func createNewWorker(ctx context.Context, id int, jobQueue chan *work, wg *sync.WaitGroup) *worker {
	w := &worker{
		ID:   id,
		jobs: jobQueue,
		ctx:  ctx,
		wg:   wg,
	}

	return w
}

// start enables the worker for processing jobs.
func (w *worker) start() {
	go func() {
		defer w.wg.Done()
		for {
			select {
			case job := <-w.jobs:
				log.Ctx(w.ctx).Info().Str("ID", fmt.Sprint(w.ID)).Str("Job ID", fmt.Sprint(job.ID)).Msg("Worker executing job.")
				err := job.Store.SetStartTime(w.ctx, job.ID, time.Now())
				if err != nil {
					log.Ctx(w.ctx).Error().Str("ID", fmt.Sprint(w.ID)).Str("Job ID", fmt.Sprint(job.ID)).Err(err).Msg("Worker failed to set start time.")
					continue
				}
				err = job.Job(w.ctx)
				if err != nil {
					log.Ctx(w.ctx).Error().Str("ID", fmt.Sprint(w.ID)).Str("Job ID", fmt.Sprint(job.ID)).Err(err).Msg("Worker failed to execute job.")
				}
				err = job.Store.SetEndTime(w.ctx, job.ID, time.Now())
				if err != nil {
					log.Ctx(w.ctx).Error().Str("ID", fmt.Sprint(w.ID)).Str("Job ID", fmt.Sprint(job.ID)).Err(err).Msg("Worker failed to set end time.")
				}
			case <-w.ctx.Done():
				log.Ctx(w.ctx).Info().Str("ID", fmt.Sprint(w.ID)).Msg("Worker received quit command.")
				return
			}
		}
	}()
}
