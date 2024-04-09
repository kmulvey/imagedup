package logger

import (
	"os"
	"testing"

	"encoding/json"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/stretchr/testify/assert"
)

type logFormatType struct {
	Big   string
	Small string
}

func TestCustomLogger(t *testing.T) {
	t.Parallel()

	var filename = "TestCustomLogger.json"

	var logger, err = NewDeleteLogger(filename)
	assert.NoError(t, err)

	logger.LogResult(hash.DiffResult{
		One:     "fileone",
		Two:     "filetwo",
		OneArea: 20,
		TwoArea: 10,
	})

	content, err := os.ReadFile(filename)
	assert.NoError(t, err)

	var result logFormatType
	assert.NoError(t, json.Unmarshal(content, &result))

	assert.Equal(t, "fileone", result.Big)
	assert.Equal(t, "filetwo", result.Small)

	assert.NoError(t, logger.Close())
	assert.NoError(t, os.RemoveAll(filename))
}
