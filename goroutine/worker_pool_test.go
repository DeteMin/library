package goroutine

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool(t *testing.T) {
	runtime.GOMAXPROCS(1)

	t.Run("new worker when idle size is zero", func(t *testing.T) {
		pool := NewWorkerPool(10, time.Second*60)

		pool.Go(func() {
			time.Sleep(time.Second * 10)
		})

		assert.Equal(t, 1, pool.BusySize())
		assert.Equal(t, 9, pool.IdleSize())

	})
}
