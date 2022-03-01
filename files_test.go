package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDupNames(t *testing.T) {
	var dir = "/home/kmulvey/Documents"
	var files, err = listFiles(dir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))
}
