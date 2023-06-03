package gomicrobee

import (
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func batchStringToInt(values []string) []int {

	re := make([]int, len(values))
	for i, s := range values {

		x, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}

		re[i] = x
	}
	return re
}

var batchCount int = 0
var batchCountMu sync.Mutex

func asBatchCount[T any](values []T) []int {
	batchCountMu.Lock()
	defer batchCountMu.Unlock()
	batchCount += 1
	re := make([]int, len(values))
	for i := range values {
		re[i] = batchCount
	}
	return re
}

func Test_submit_less_than_limit(t *testing.T) {
	assert := assert.New(t)
	sys := NewSystem[string, int](batchStringToInt, 3, 10*time.Millisecond)

	re1, err := sys.Submit("1")
	assert.Nil(err)

	assert.Equal(1, re1.Get())
	assert.Equal(1, re1.Get()) // you can call Get multiple times
}

func Test_submit_and_process_same_as_limit(t *testing.T) {
	assert := assert.New(t)
	sys := NewSystem[string, int](batchStringToInt, 3, 10*time.Millisecond)

	re1, err := sys.Submit("1")
	assert.Nil(err)
	re2, err := sys.Submit("2")
	assert.Nil(err)
	re3, err := sys.Submit("3")
	assert.Nil(err)

	assert.Equal(1, re1.Get())
	assert.Equal(2, re2.Get())
	assert.Equal(3, re3.Get())
}

func Test_submit_more_than_limit(t *testing.T) {
	assert := assert.New(t)
	batchCount = 0
	sys := NewSystem[string, int](asBatchCount[string], 1, 10*time.Millisecond)

	re1, err := sys.Submit("")
	assert.Nil(err)
	re2, err := sys.Submit("2")
	assert.Nil(err)
	re3, err := sys.Submit("3")
	assert.Nil(err)

	results := []int{re1.Get(), re2.Get(), re3.Get()}
	sort.Ints(results)

	assert.Equal([]int{1, 2, 3}, results)
}

// Check that we can have two processors running concurrently.
func Test_concurrent_batches(t *testing.T) {
	assert := assert.New(t)

	// use two chans to block the main thread and the processors until
	// we have checked they are both running at once.
	procStartedCh := make(chan bool, 2)
	procCanProceed := make(chan bool, 2)

	processor := func(values []string) []string {
		procStartedCh <- true // notify the main thread this process has started.

		<-procCanProceed // wait for the main thread to less this process continue

		return values
	}

	sys := NewSystem[string, string](processor, 1, 10*time.Millisecond)

	re1, err := sys.Submit("a")
	assert.Nil(err)
	re2, err := sys.Submit("b")
	assert.Nil(err)

	// read from procStarted twice to make sure both processors have started.
	<-procStartedCh
	<-procStartedCh

	// now let the two processors proceed
	procCanProceed <- true
	procCanProceed <- true

	assert.Equal("a", re1.Get())
	assert.Equal("b", re2.Get())
}

func Test_sumbit_many(t *testing.T) {
	assert := assert.New(t)

	count := 0
	var mu sync.Mutex

	countBatch := func(xs []int) []int {
		mu.Lock()
		defer mu.Unlock()
		count += len(xs)
		return xs
	}

	sys := NewSystem[int, int](countBatch, 3, 10*time.Millisecond)

	results := make([]JobResult[int], 10000)

	for i := 0; i < 10000; i++ {
		results[i], _ = sys.Submit(i)
	}
	for _, result := range results {
		result.Get()
	}

	assert.Equal(10000, count)
}

func Test_many_producsors(t *testing.T) {
	assert := assert.New(t)

	count := 0
	var mu sync.Mutex

	countBatch := func(xs []int) []int {
		mu.Lock()
		defer mu.Unlock()
		count += len(xs)
		return xs
	}

	sys := NewSystem[int, int](countBatch, 3, 10*time.Millisecond)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			jobs_per_producer := 1000
			results := make([]JobResult[int], jobs_per_producer)

			for j := 0; j < jobs_per_producer; j++ {
				results[j], _ = sys.Submit(j)
			}

			for _, result := range results {
				result.Get()
			}
		}()
	}

	wg.Wait()

	assert.Equal(100000, count)
}

func Test_shutdown(t *testing.T) {
	assert := assert.New(t)

	sys := NewSystem[string, int](batchStringToInt, 3, 10*time.Millisecond)

	re1, err := sys.Submit("1")
	assert.Nil(err)

	sys.Shutdown()

	_, err = sys.Submit("2")
	assert.Equal(ErrShutdown, err)

	assert.Equal(1, re1.Get())
}
