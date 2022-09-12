package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/pkg/imagedup/types"
)

func (id *ImageDup) streamFiles(ctx context.Context, files []string) {
	var dedup = make(map[string]struct{})
	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case <-ctx.Done():
					close(id.images)
					return
				default:
					if _, found := dedup[one+two]; !found {
						dedup[one+two] = struct{}{}
						dedup[two+one] = struct{}{}
						id.images <- types.Pair{One: one, Two: two, I: i, J: j}
						id.stats.PairTotal.Inc()
					}
				}
			}
		}
	}
	close(id.images)
}
