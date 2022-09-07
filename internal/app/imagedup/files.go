package imagedup

import (
	"github.com/kmulvey/imagedup/pkg/types"
)

func (id *ImageDup) streamFiles(files []string, pairChan chan types.Pair) {
	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case <-id.Context.Done():
					close(pairChan)
					return
				default:
					pairChan <- types.Pair{One: one, Two: two, I: i, J: j}
					id.stats.PairTotal.Inc()
				}
			}
		}
	}
	close(pairChan)
}
