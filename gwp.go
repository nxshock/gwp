package gwp

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// WorkerPool represents pool of workers.
type WorkerPool struct {
	f             chan func()
	r             chan struct{}
	stopChan      chan struct{}
	wg            sync.WaitGroup
	EstimateCount int
}

// New creates new pool of workers with specified goroutine count.
func New(threadCount int) *WorkerPool {
	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	workerPool := &WorkerPool{
		f:        make(chan func()),
		r:        make(chan struct{}),
		stopChan: make(chan struct{})}

	workerPool.wg.Add(threadCount)

	go func() {
		var counter int
		var prevPos int
		prevTime := time.Now()

		const calculateEtaPeriod = time.Minute

		tickerUpdateText := time.NewTicker(time.Second)
		tickerCalculateEta := time.NewTicker(calculateEtaPeriod)
		defer func() {
			tickerUpdateText.Stop()
			tickerCalculateEta.Stop()
		}()

		var currentSpeed float64 // items per sec

		fmt.Fprintf(os.Stderr, endLine)
		for {
			select {
			case <-tickerUpdateText.C:
				if workerPool.EstimateCount > 0 {
					fmt.Fprintf(os.Stderr, newLine)
					fmt.Fprintf(os.Stderr, "%.1f%% (%d / %d) ETA: %s at %.2f speed"+endLine, float64(counter*100)/float64(workerPool.EstimateCount), counter, workerPool.EstimateCount,
						time.Second*time.Duration(float64(workerPool.EstimateCount-counter)/currentSpeed), currentSpeed)
				}
			case <-tickerCalculateEta.C:
				currentSpeed = float64(counter-prevPos) * float64(time.Second) / float64(time.Now().Sub(prevTime))
				prevPos = counter
				prevTime = time.Now()
			case <-workerPool.r:
				counter++
			case <-workerPool.stopChan:
				break
			}
		}
	}()

	for i := 0; i < threadCount; i++ {
		go func() {
			defer workerPool.wg.Done()

			for f := range workerPool.f {
				f()
				workerPool.r <- struct{}{}
			}
		}()
	}

	return workerPool
}

// Add sends specified task for execution.
func (workerPool *WorkerPool) Add(f func()) {
	workerPool.f <- f
}

// CloseAndWait stops accepting tasks and waits for all tasks to complete.
func (workerPool *WorkerPool) CloseAndWait() {
	close(workerPool.f)
	workerPool.wg.Wait()
	workerPool.stopChan <- struct{}{}
	close(workerPool.r)
}
