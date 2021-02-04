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
	jobChan       chan func() error
	resultChan    chan error
	stopChan      chan struct{}
	wg            sync.WaitGroup
	EstimateCount int

	processedCount int     // processed jobs count
	errorCount     int     // processed jobs count that returned error
	currentSpeed   float64 // speed calculated for last minute
}

// New creates new pool of workers with specified goroutine count.
// If specified number of workers less than 1, runtume.NumCPU() is used.
func New(threadCount int) *WorkerPool {
	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	workerPool := &WorkerPool{
		jobChan:    make(chan func() error),
		resultChan: make(chan error),
		stopChan:   make(chan struct{})}

	workerPool.wg.Add(threadCount)

	go func() {
		var prevPos int
		prevTime := time.Now()

		tickerUpdateText := time.NewTicker(defaultProgressUpdatePeriod)
		tickerCalculateEta := time.NewTicker(defaultCalculateEtaPeriod)
		defer func() {
			tickerUpdateText.Stop()
			tickerCalculateEta.Stop()
		}()

		fmt.Fprintf(os.Stderr, endLine)
		for {
			select {
			case <-tickerUpdateText.C:
				workerPool.printProgress()
			case <-tickerCalculateEta.C:
				workerPool.currentSpeed = float64(workerPool.processedCount-prevPos) * float64(defaultCalculateEtaPeriod) / float64(time.Now().Sub(prevTime))
				prevPos = workerPool.processedCount
				prevTime = time.Now()
			case err := <-workerPool.resultChan:
				if err != nil {
					workerPool.errorCount++
				}
				workerPool.processedCount++
			case <-workerPool.stopChan:
				return
			}
		}
	}()

	for i := 0; i < threadCount; i++ {
		go func() {
			defer workerPool.wg.Done()

			for f := range workerPool.jobChan {
				workerPool.resultChan <- f()
			}
		}()
	}

	return workerPool
}

func (workerPool *WorkerPool) printProgress() {
	if workerPool.EstimateCount == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, newLine)
	fmt.Fprintf(os.Stderr, "Progress: %.1f%% (%d / %d)",
		float64(workerPool.processedCount*100)/float64(workerPool.EstimateCount), workerPool.processedCount, workerPool.EstimateCount)

	if workerPool.errorCount > 0 {
		fmt.Fprintf(os.Stderr, "    Errors: %d (%.1f%%)",
			workerPool.errorCount, float64(workerPool.errorCount*100)/float64(workerPool.EstimateCount))
	}
	if workerPool.currentSpeed > 0 {
		fmt.Fprintf(os.Stderr, "    ETA: %s at %.2f rps",
			time.Second*time.Duration(float64(workerPool.EstimateCount-workerPool.processedCount)/workerPool.currentSpeed), workerPool.currentSpeed)
	}
	fmt.Fprint(os.Stderr, endLine)
}

// Add sends specified task for execution.
func (workerPool *WorkerPool) Add(f func() error) {
	workerPool.jobChan <- f
}

// CloseAndWait stops accepting tasks and waits for all tasks to complete.
func (workerPool *WorkerPool) CloseAndWait() {
	close(workerPool.jobChan)
	workerPool.wg.Wait()
	workerPool.stopChan <- struct{}{}
	close(workerPool.resultChan)

	workerPool.printProgress()
}
