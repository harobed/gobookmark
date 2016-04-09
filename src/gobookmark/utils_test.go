package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractTags(t *testing.T) {
	assert.Equal(t, extractTags("[foo][bar] extra")[0], "foo")
	assert.Equal(t, extractTags("[foo][bar ] extra")[1], "bar")
	assert.Len(t, extractTags("extra"), 0)
}

func TestRemoveTags(t *testing.T) {
	assert.Equal(t, removeTags("[foo][bar] extra"), "extra")
}
