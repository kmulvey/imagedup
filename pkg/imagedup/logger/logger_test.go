package logger

import (
	"os"
	"testing"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/stretchr/testify/assert"
)

func TestCustomLogger(t *testing.T) {
	t.Parallel()

	var filename = "TestCustomLogger.json"
	assert.NoError(t, os.RemoveAll(filename)) // defensive

	var logger, err = NewDeleteLogger(filename)
	assert.NoError(t, err)

	// one is bigger
	err = logger.LogResult(hash.DiffResult{
		One:     "fileone",
		Two:     "filetwo",
		OneArea: 20,
		TwoArea: 10,
	})
	assert.NoError(t, err)

	// two is bigger
	err = logger.LogResult(hash.DiffResult{
		One:     "fileone",
		Two:     "filetwo",
		OneArea: 10,
		TwoArea: 20,
	})
	assert.NoError(t, err)
	assert.NoError(t, logger.Close())

	deletes, err := ReadDeleteLogFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(deletes))

	assert.Equal(t, "fileone", deletes[0].Big)
	assert.Equal(t, "filetwo", deletes[0].Small)

	assert.Equal(t, "fileone", deletes[1].Small)
	assert.Equal(t, "filetwo", deletes[1].Big)

	assert.NoError(t, os.RemoveAll(filename))
}
