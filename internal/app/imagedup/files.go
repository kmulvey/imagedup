package imagedup

import (
	"context"
	"math"

	"github.com/kmulvey/imagedup/pkg/imagedup/types"
)

func (id *ImageDup) streamFiles(ctx context.Context, files []string) {
	var numImages = float64(len(files))
	id.stats.TotalComparisons.Set((math.Pow(numImages, 2) - numImages) / 2)

	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case <-ctx.Done():
					close(id.images)
					return
				default:
					if id.dedupPairs {
						if found := id.Bitmap.Contains(compress(i, j)); !found {
							id.images <- types.Pair{One: one, Two: two, I: i, J: j}
							id.Bitmap.Add(compress(j, i))
							id.stats.PairTotal.Inc()
						}
					} else {
						id.images <- types.Pair{One: one, Two: two, I: i, J: j}
						id.stats.PairTotal.Inc()
					}
				}
			}
		}
	}
	close(id.images)
}

// compress stores two ints in one. Go stores ints as 8 bytes so we store
// the first int in the bottom four and the second in the top four.
// This has a limitation of only being able to store a max value of 4294967295.
func compress(a, b int) uint64 {
	return uint64(a) | (uint64(b) << 32)
}
