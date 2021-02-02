# gwp

Simple goroutine worker pool written in Go.

Status: Work in progress.

Example:

```go
package main

import (
	"log"
	"time"

	"github.com/nxshock/gwp"
)

// Job function
func f(i int) error {
	log.Printf("Job №%d", i)

	// Simulate work
	time.Sleep(time.Second)

	return nil
}

func main() {
	worker := gwp.New(4)

	for i := 0; i < 100; i++ {
		n := i
		// Send job
		worker.Add(func() error {
			return f(n)
		})
	}

	// Wait all jobs to complete
	worker.CloseAndWait()
}
```
