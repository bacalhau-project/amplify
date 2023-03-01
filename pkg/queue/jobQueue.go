package queue

type Task interface {
	// Job is the Bacalhau Job Specification
	Job() error
	// OnSuccess is the callback function to be executed
	// when the job is successfully processed.
	OnSuccess() error
}

type Dispatcher interface {
	// Push takes an Event and pushes to a queue.
	Push(Task) error
	// Run spawns the workers and waits indefinitely for
	// the events to be processed.
	Run()
}

// EventDispatcher represents the datastructure for an
// EventDispatcher instance. This struct satisfies the
// Dispatcher interface.
type EventDispatcher struct {
	Opts     Options
	Queue    chan models.Notification
	Finished bool
}

// Options represent options for EventDispatcher.
type Options struct {
	MaxWorkers int // Number of workers to spawn.
	MaxQueueSize int // Maximum length for the queue to hold events.
}

// NewEventDispatcher initialises a new event dispatcher.
func NewEventDispatcher(opts Options) (Dispatcher) {
	return EventDispatcher{
		Opts: opts,
		Queue: make(chan Task, opts.MaxQueueSize),
		Finished: false,
	}
}

// Push adds a new event payload to the queue.
func (d *EventDispatcher Push(event Task) error {
	if (d.Finished) {
		return errors.New(`queue is closed`)
	}
	d.Queue <- event
	return nil
}

// Run spawns workers and listens to the queue
// It's a blocking function and waits for a cancellation
// invocation from the Client.
func (d *EventDispatcher Run(ctx context.Context) {
	wg := sync.WaitGroup{}
	for i := 0; i < d.Opts.MaxWorkers; i++ {
		wg.Add(1) // Add a wait group for each worker
		// Spawn a worker
		go func() {
			for {
				select {
				case <-ctx.Done():
					// Ensure no new messages are added.
					d.Finished = true
					// Flush all events
					e.Flush()
					wg.Done()
					return
				case e <- d.Queue:
					e.Process()
				}
			}
		}()
	}
	wg.Wait()
}
