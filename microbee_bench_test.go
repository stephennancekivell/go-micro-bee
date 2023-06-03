package gomicrobee

import (
	"testing"
	"time"
)

func Benchmark_process_batches(b *testing.B) {
	sys := NewSystem[string, int](batchStringToInt, 3, 10*time.Millisecond)
	for n := 0; n < b.N; n++ {
		sys.Submit("1")
	}
}
