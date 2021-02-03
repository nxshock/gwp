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
	f             chan func() error
	r             chan error
	stopChan      chan struct{}
	wg            sync.WaitGroup
	EstimateCount int
}

// New creates new pool of workers with specified goroutine count.
// If specified number of workers less than 1, runtume.NumCPU() is used.
func New(threadCount int) *WorkerPool {
	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	workerPool := &WorkerPool{
		f:        make(chan func() error),
		r:        make(chan error),
		stopChan: make(chan struct{})}

	workerPool.wg.Add(threadCount)

	go func() {
		var processedCount int
		var errorCount int
		var prevPos int
		prevTime := time.Now()

		tickerUpdateText := time.NewTicker(defaultProgressUpdatePeriod)
		tickerCalculateEta := time.NewTicker(defaultCalculateEtaPeriod)
		defer func() {
			tickerUpdateText.Stop()
			tickerCalculateEta.Stop()
		}()

		var currentSpeed float64 // jobs per sec

		fmt.Fprintf(os.Stderr, endLine)
		for {
			select {
			case <-tickerUpdateText.C:
				if workerPool.EstimateCount == 0 {
					continue
				}

				fmt.Fprintf(os.Stderr, newLine)
				fmt.Fprintf(os.Stderr, "Progress: %.1f%% (%d / %d)",
					float64(processedCount*100)/float64(workerPool.EstimateCount), processedCount, workerPool.EstimateCount)

				if errorCount > 0 {
					fmt.Fprintf(os.Stderr, "    Errors: %d (%.1f%%)",
						errorCount, float64(errorCount*100)/float64(workerPool.EstimateCount))
				}
				if currentSpeed > 0 {
					fmt.Fprintf(os.Stderr, "    ETA: %s at %.2f rps",
						time.Second*time.Duration(float64(workerPool.EstimateCount-processedCount)/currentSpeed), currentSpeed)
				}
				fmt.Fprint(os.Stderr, endLine)
			case <-tickerCalculateEta.C:
				currentSpeed = float64(processedCount-prevPos) * float64(time.Second) / float64(time.Now().Sub(prevTime))
				prevPos = processedCount
				prevTime = time.Now()
			case err := <-workerPool.r:
				if err != nil {
					errorCount++
				}
				processedCount++
			case <-workerPool.stopChan:
				break
			}
		}
	}()

	for i := 0; i < threadCount; i++ {
		go func() {
			defer workerPool.wg.Done()

			for f := range workerPool.f {
				workerPool.r <- f()
			}
		}()
	}

	return workerPool
}

// Add sends specified task for execution.
func (workerPool *WorkerPool) Add(f func() error) {
	workerPool.f <- f
}

// CloseAndWait stops accepting tasks and waits for all tasks to complete.
func (workerPool *WorkerPool) CloseAndWait() {
	close(workerPool.f)
	workerPool.wg.Wait()
	workerPool.stopChan <- struct{}{}
	close(workerPool.r)
}
