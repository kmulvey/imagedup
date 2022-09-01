package imagedup

import (
	"context"
)

func streamFiles(ctx context.Context, files []string, pairChan chan pair) {
	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case <-ctx.Done():
					close(pairChan)
					return
				default:
					pairChan <- pair{One: one, Two: two, I: i, J: j}
					pairTotal.Inc()
				}
			}
		}
	}
	close(pairChan)
}
