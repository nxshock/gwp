package gwp

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	for i := 0; i < 10; i++ {
		wp := New(i)

		count := new(int64)

		for j := 0; j < 100; j++ {
			wp.Add(func() error {
				atomic.AddInt64(count, 1)
				return nil
			})
		}

		wp.CloseAndWait()

		assert.EqualValues(t, 100, *count)
	}
}
