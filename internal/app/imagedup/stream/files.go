package stream

import (
	"context"

	"github.com/kmulvey/imagedup/pkg/types"
)

func StreamFiles(ctx context.Context, files []string, pairChan chan types.Pair) {
	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case <-ctx.Done():
					close(pairChan)
					return
				default:
					pairChan <- types.Pair{One: one, Two: two, I: i, J: j}
					pairTotal.Inc()
				}
			}
		}
	}
	close(pairChan)
}
