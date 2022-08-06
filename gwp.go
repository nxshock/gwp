package gwp

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/nxshock/go-eta"
)

// WorkerPool represents pool of workers.
type WorkerPool struct {
	jobChan    chan func() error
	resultChan chan error
	stopChan   chan struct{}
	wg         sync.WaitGroup

	estimateCount int
	ShowProgress  bool
	ShowSpeed     bool

	processedCount int     // processed jobs count
	errorCount     int     // processed jobs count that returned error
	currentSpeed   float64 // speed calculated for last minute

	eta *eta.Calculator

	lastProgressMessage string
}

// New creates new pool of workers with specified goroutine count.
// If specified number of workers less than 1, runtime.NumCPU() is used.
func New(threadCount int) *WorkerPool {
	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	workerPool := &WorkerPool{
		jobChan:    make(chan func() error),
		resultChan: make(chan error),
		stopChan:   make(chan struct{}),
		eta:        eta.New(time.Minute, 0)}

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

		newLined := false

		for {
			select {
			case <-tickerUpdateText.C:
				if !newLined {
					fmt.Fprint(os.Stderr, endLine)
					newLined = true
				}

				_ = workerPool.printProgress()
			case <-tickerCalculateEta.C:
				workerPool.currentSpeed = float64(workerPool.processedCount-prevPos) * float64(time.Second) / float64(time.Since(prevTime))
				prevPos = workerPool.processedCount
				prevTime = time.Now()
			case err := <-workerPool.resultChan:
				if err != nil {
					workerPool.errorCount++
				}
				workerPool.processedCount++
				workerPool.eta.Increment(1)
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

func (workerPool *WorkerPool) SetEstimateCount(n int) {
	workerPool.estimateCount = n
	workerPool.eta.TotalCount = n
}

func (workerPool *WorkerPool) printProgress() error {
	if !workerPool.ShowProgress {
		return nil
	}

	buf := new(bytes.Buffer)

	fmt.Fprint(buf, newLine)

	if workerPool.estimateCount == 0 {
		fmt.Fprintf(buf, "Progress: %d", workerPool.processedCount)
	} else {
		fmt.Fprintf(buf, "Progress: %.1f%% (%d / %d)",
			float64(workerPool.processedCount*100)/float64(workerPool.estimateCount), workerPool.processedCount, workerPool.estimateCount)
	}

	if workerPool.errorCount > 0 {
		fmt.Fprintf(buf, "    Errors: %d (%.1f%%)",
			workerPool.errorCount, float64(workerPool.errorCount*100)/float64(workerPool.estimateCount))
	}

	if workerPool.currentSpeed > 0 {
		if workerPool.estimateCount > 0 {
			fmt.Fprintf(buf, "    ETA: %s", fmtDuration(time.Until(workerPool.eta.Average())))
		}

		if workerPool.ShowSpeed {
			fmt.Fprintf(buf, "    Speed: %.2f rps", workerPool.currentSpeed)
		}
	}

	fmt.Fprint(buf, endLine)

	if buf.String() == workerPool.lastProgressMessage {
		return nil
	}

	workerPool.lastProgressMessage = buf.String()

	_, err := buf.WriteTo(os.Stderr)
	if err != nil {
		return err
	}

	return nil
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

	_ = workerPool.printProgress()
}

// ErrorCount returns total error count.
func (workerPool *WorkerPool) ErrorCount() int {
	return workerPool.errorCount
}
