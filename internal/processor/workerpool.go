package processor

import (
	"context"
	"fmt"
	"sync"

)

type Event struct {
	ID string
	Payload []byte
}

func StartWorkerPool(ctx context.Context, numWorkers int, events <-chan Event) {
	var wg sync.WaitGroup
	// complete trust in the external context provided by the call (main.go)
	//ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Millisecond)
	//defer cancel()

	for i:=0; i<numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("[Worker %d] Shutting down gracefully...\n", workerID)
					return

				case e, ok := <-events:
					if !ok { return }
					process(e)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("All workers finished. Worker pool stopped.")
}

func process(e Event) {
	fmt.Printf("Processing event %s\n", e.ID)
}
