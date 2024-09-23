package gomicrobee

import (
	"errors"
	"sync"
	"time"
)

// The BatchProcessor function transforms a list of values into a list of results.
// The processor MUST produce a result for each input, and the order of the results
// must be in the same order as the input.
type BatchProcessor[A any, B any] func(jobs []A) []B

// A System is configured with a BatchProcessor, a batch size and an interval.
// the System accepts jobs via `Submit` and returns a `JobResult` which contains a
// way to get the result when it is eventually ready.
type System[A any, B any] interface {
	// Add a new job to the queue of pending jobs.
	// Jobs are added to the queue but their execution is delayed until the batch is processed.
	// Returns a JobResult with the eventual answer.
	// Can return an error if the System was unable to process the job. Eg, it was shutdown.
	Submit(job A) (JobResult[B], error)
	// Stop the system from accepting new jobs and finish any existing jobs.
	Shutdown()
}

// JobResults contain a reference to Get the eventual result.
type JobResult[T any] interface {
	// Block until the result is ready, potentially waiting for the System wait for more jobs and execute the batch.
	// Can be called multiple times to receive the same value.
	Get() T
}

type systemImpl[A any, B any] struct {
	batchProcessor BatchProcessor[A, B]
	batchSize      int
	linger         time.Duration

	// Store a pending batch of submitted jobs in shared memory and lock access to it.
	// Using a shared slice gives us flexability to use the jobs with our two
	// triggers (batchSize and linger time)
	pending  *batch[A, B]
	mu       sync.Mutex
	shutdown bool // if the system has shutdown
}

var _ System[any, any] = (*systemImpl[any, any])(nil)

func NewSystem[A any, B any](
	processor BatchProcessor[A, B],
	// The system will attempt to batch Jobs into groups of this size to be processed via the BatchProcessor
	batchSize int,
	// The system will batch Jobs together in batches within the linger time. This means that if a single job
	// arrives it will wait for other jobs until the linger duratino has elapsed.
	linger time.Duration,
) System[A, B] {
	return &systemImpl[A, B]{processor, batchSize, linger, nil, sync.Mutex{}, false}
}

type jobResultImpl[T any] struct {
	once  sync.Once
	ch    chan T
	value T
}

var _ JobResult[any] = (*jobResultImpl[any])(nil)

func (s *jobResultImpl[T]) Get() T {
	s.once.Do(func() {
		s.value = <-s.ch
	})
	return s.value
}

type jobValue[A any, B any] struct {
	value    A
	resultCh chan B
}

type batch[A any, B any] struct {
	values []jobValue[A, B]
	timer  *timer_plus
}

func (s *systemImpl[A, B]) Submit(value A) (JobResult[B], error) {
	resultCh := make(chan B, 1)
	job := jobValue[A, B]{value, resultCh}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		return nil, ErrShutdown
	}
	if s.pending == nil {
		s.pending = s.newBatch(job)
	} else {
		s.pending.values = append(s.pending.values, job)
	}

	if len(s.pending.values) >= s.batchSize {
		s.pending.stop()
		s.pending = nil
	}

	return &jobResultImpl[B]{once: sync.Once{}, ch: resultCh}, nil
}

// Creates a new batch that will execute after linger or when the timer is stopped.
func (s *systemImpl[A, B]) newBatch(firstValue jobValue[A, B]) *batch[A, B] {
	timer := newTimerPlus(s.linger)

	currentBatch := batch[A, B]{
		values: []jobValue[A, B]{firstValue},
		timer:  &timer,
	}

	go func() {
		select {
		case <-timer.stopCh:
		case <-timer.timer.C:
			s.mu.Lock()
			s.pending = nil
			s.mu.Unlock()
		}

		// Execute the batch in the same go func as where we are waiting for the timer.
		s.executeBatch(currentBatch.values)
	}()

	return &currentBatch
}

func (s *batch[A, B]) stop() {
	s.timer.stop()
}

func (s *systemImpl[A, B]) executeBatch(jobValues []jobValue[A, B]) {
	values := make([]A, len(jobValues))
	for i, jv := range jobValues {
		values[i] = jv.value
	}

	results := s.batchProcessor(values)
	// assert that the processor has enough results.
	// its not ideal that we need to panic, but it avoids having to return the possible error.
	// We are faviouring a simple API with a strong requirement that we return results.
	if len(results) != len(values) {
		panic("batch processor must produce a result for each value.")
	}

	for i, r := range results {
		jobValues[i].resultCh <- r
	}
}

var ErrShutdown = errors.New("shutdown")

func (s *systemImpl[A, B]) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = true
	if len(s.pending.values) > 0 {
		s.pending.stop()
		s.pending = nil
	}
}
