package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDupNames(t *testing.T) {
	var dir = "./testimages"
	var files, err = listFiles(dir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))
}
